package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func qqBusinessFixture(t *testing.T) (*QQOfficialService, *gorm.DB, *time.Time) {
	t.Helper()
	qq, db, now := qqFixture(t)
	if err := db.AutoMigrate(&model.Fleet{}, &model.EveCharacter{}, &model.EveCharacterCorpRole{}, &model.SystemConfig{}, &model.CorpStructureInfo{}); err != nil {
		t.Fatal(err)
	}
	qqEnable(t, qq)
	return qq, db, now
}

func qqFleetRuntimeFixture(t *testing.T) (*QQOfficialService, *gorm.DB, *time.Time) {
	t.Helper()
	sample, db, now := qqBusinessFixture(t)
	rt := NewQQOfficialRuntime(db, qqDeploymentKeyFixture)
	secret := "managed-fleet-test-secret"
	if _, err := rt.UpdateSettings(t.Context(), 7, QQOfficialSettingsUpdate{AppID: &sample.appID, AppSecret: &secret}); err != nil {
		t.Fatal(err)
	}
	qq := qqActive(t, rt)
	qq.now = sample.now
	qqEnable(t, qq)
	previousDB, previousQQ := global.DB, DefaultQQOfficialRuntime()
	global.DB = db
	SetDefaultQQOfficialRuntime(rt)
	t.Cleanup(func() { global.DB = previousDB; SetDefaultQQOfficialRuntime(previousQQ) })
	return qq, db, now
}

func qqBusinessWrite(t *testing.T, db *gorm.DB, rows ...interface{}) {
	t.Helper()
	for _, row := range rows {
		if err := db.Save(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func qqBusinessTarget(t *testing.T, qq *QQOfficialService, n int, fleetCorp, structureCorp int64) model.QQOfficialGroup {
	t.Helper()
	group := qqGroupFixture(t, qq, n)
	group.FleetCorporation, group.StructureCorporation = fleetCorp, structureCorp
	group.NotificationEnabled, group.TestVerified, group.PushState = true, true, qqAllowedPush
	if err := qq.repo.SaveGroup(t.Context(), nil, &group); err != nil {
		t.Fatal(err)
	}
	return group
}

func TestQQBusinessFleetOptInIsAtomicAndDownstreamFailureDoesNotRollback(t *testing.T) {
	qq, db, now := qqFleetRuntimeFixture(t)
	ctx := t.Context()
	fleets := &FleetService{repo: repository.NewFleetRepository()}
	const corp = 101
	char := model.EveCharacter{UserID: 7, CharacterID: 123, CharacterName: "pilot", CorporationID: corp}
	qqBusinessWrite(t, db, &char)
	fleet := model.Fleet{ID: uuid.NewString(), StartAt: now.Add(5 * time.Minute), FCUserID: char.UserID, FCCharacterID: char.CharacterID, Importance: model.FleetImportanceCTA, Title: "@everyone <b>arbitrary</b>", Description: "https://example.com"}
	qqBusinessTarget(t, qq, 2, corp+1, 0)
	if _, err := fleets.persistFleetWithOptionalQQ(ctx, &fleet, true); !errors.Is(err, ErrQQOfficialNoTargets) {
		t.Fatalf("wrong-corporation admission returned %v", err)
	}
	if _, err := fleets.repo.GetByID(fleet.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("opt-in without targets committed a fleet: %v", err)
	}
	notOptedIn := fleet
	notOptedIn.ID = uuid.NewString()
	if n, err := fleets.persistFleetWithOptionalQQ(ctx, &notOptedIn, false); err != nil || n != 0 {
		t.Fatalf("webhook-only/default creation: %d %v", n, err)
	}
	group := qqBusinessTarget(t, qq, 3, corp, 0)
	if n, err := fleets.persistFleetWithOptionalQQ(ctx, &fleet, true); err != nil || n != 1 {
		t.Fatalf("opt-in admission %d %v", n, err)
	}
	rows, total, err := qq.FleetTasks(ctx, fleet.ID, 0, 10)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Status != model.QQQueued || rows[0].GroupID != group.ID || !rows[0].ExpiresAt.Equal(fleet.StartAt) {
		t.Fatalf("incorrect fleet tasks %+v total %d err %v", rows, total, err)
	}
	intent, err := qq.repo.Intent(ctx, nil, qq.appID, qqHash("fleet:created:"+fleet.ID))
	if err != nil || intent.ID != rows[0].IntentID {
		t.Fatalf("non-deterministic create key: %+v %v", intent, err)
	}
	qq.setToken(qqToken(*now), 1)
	qq.client = qqFakeClient{execute: func(_ context.Context, _ string, req qqapi.Request) (qqapi.Result, error) {
		if req.GroupOpenID != group.OpenID || len(req.Text) > 1024 || strings.ContainsAny(req.Text, "@<>") || strings.Contains(req.Text, "example.com") || !strings.Contains(req.Text, "全面集结") {
			t.Fatalf("unexpected curated message %+v", req)
		}
		return qqapi.Result{}, &qqapi.OperationError{Kind: qqapi.Ambiguous}
	}}
	if ran, err := qq.dispatchOnce(ctx); !ran || err != nil {
		t.Fatalf("dispatch did not reach the fake client: %v %v", ran, err)
	}
	row, _, err := qq.Task(ctx, rows[0].ID)
	if err != nil || row.Status != model.QQUnknown {
		t.Fatalf("unknown provider outcome not reflected: %+v %v", row, err)
	}
	if _, err := fleets.repo.GetByID(fleet.ID); err != nil {
		t.Fatalf("downstream failure rolled back fleet: %v", err)
	}
}

func TestQQBusinessManualFleetPingChecksOwnershipReplayAndCurrentCorporation(t *testing.T) {
	qq, db, now := qqFleetRuntimeFixture(t)
	ctx := t.Context()
	fleetSvc := &FleetService{repo: repository.NewFleetRepository()}
	const corp = 101
	char := model.EveCharacter{UserID: 7, CharacterID: 123, CharacterName: "pilot", CorporationID: corp}
	fleet := model.Fleet{ID: uuid.NewString(), StartAt: now.Add(time.Hour), FCUserID: char.UserID, FCCharacterID: char.CharacterID}
	other := fleet
	other.ID = uuid.NewString()
	qqBusinessWrite(t, db, &char, &fleet, &other)
	group := qqBusinessTarget(t, qq, 4, corp, 0)

	key := uuid.NewString()
	if _, err := fleetSvc.SendFleetQQPing(ctx, fleet.ID, key, 99, []string{model.RoleFC}); !errors.Is(err, ErrFleetQQForbidden) {
		t.Fatalf("non-owner FC submitted: %v", err)
	}
	if _, _, err := fleetSvc.ListFleetQQPingResults(ctx, fleet.ID, 99, []string{model.RoleFC}, 0, 1); !errors.Is(err, ErrFleetQQForbidden) {
		t.Fatalf("non-owner FC read ledger: %v", err)
	}
	rows, err := fleetSvc.SendFleetQQPing(ctx, fleet.ID, key, char.UserID, []string{model.RoleFC})
	if err != nil || len(rows) != 1 || !rows[0].ExpiresAt.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("manual submission %+v %v", rows, err)
	}
	if _, err := fleetSvc.SendFleetQQPing(ctx, other.ID, key, char.UserID, []string{model.RoleFC}); !errors.Is(err, ErrQQOfficialConflict) {
		t.Fatalf("key reused for another fleet: %v", err)
	}
	valid, err := qq.businessStillValid(ctx, nil, rows[0], group, *now)
	if !valid || err != nil {
		t.Fatalf("expected originally valid: %t %v", valid, err)
	}
	char.CorporationID++
	qqBusinessWrite(t, db, &char)
	valid, err = qq.businessStillValid(ctx, nil, rows[0], group, *now)
	if valid || err != nil {
		t.Fatalf("old target valid after moving corp: %t %v", valid, err)
	}
	fleet.StartAt = now.Add(-time.Second)
	qqBusinessWrite(t, db, &fleet)
	replay, err := fleetSvc.SendFleetQQPing(ctx, fleet.ID, key, char.UserID, []string{model.RoleFC})
	if err != nil || len(replay) != 1 || replay[0].ID != rows[0].ID || !replay[0].ExpiresAt.Equal(rows[0].ExpiresAt) {
		t.Fatalf("manual replay %+v %v", replay, err)
	}
	if _, err := fleetSvc.SendFleetQQPing(ctx, fleet.ID, uuid.NewString(), char.UserID, []string{model.RoleFC}); !errors.Is(err, ErrQQOfficialExpired) {
		t.Fatalf("late submission %v", err)
	}
	ledger, total, err := fleetSvc.ListFleetQQPingResults(ctx, fleet.ID, 99, []string{model.RoleSeniorFC}, 0, 1)
	if err != nil || total != 1 || len(ledger) != 1 || ledger[0].ID != rows[0].ID {
		t.Fatalf("senior FC scoped ledger %+v %d %v", ledger, total, err)
	}
}

func qqBusinessAuthorizeStructureCorp(t *testing.T, db *gorm.DB, corp, char int64) {
	t.Helper()
	qqBusinessWrite(t, db,
		&model.SystemConfig{Key: model.SysConfigDashboardCorpStructuresAuth, Value: fmt.Sprintf(`{"%d":%d}`, corp, char)},
		&model.SystemConfig{Key: model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays, Value: "1"},
		&model.SystemConfig{Key: model.SysConfigDashboardCorpStructuresTimerNoticeThresholdDays, Value: "1"},
		&model.EveCharacter{UserID: 42, CharacterID: char, CharacterName: "director", CorporationID: corp},
		&model.EveCharacterCorpRole{CharacterID: char, CorporationID: corp, CorpRole: "Director"},
		&model.UserRole{UserID: 42, RoleCode: model.RoleAdmin},
	)
}

func qqBusinessScan(t *testing.T, qq *QQOfficialService) {
	t.Helper()
	if err := qq.ScanStructureAlerts(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestQQBusinessStructureVersionsStalenessAndDispatchGuards(t *testing.T) {
	qq, db, now := qqBusinessFixture(t)
	ctx := t.Context()
	const corp, id, char = model.SystemCorporationID, 123, 999
	qqBusinessAuthorizeStructureCorp(t, db, corp, char)
	group := qqBusinessTarget(t, qq, 5, 0, corp)
	wrongGroup := qqBusinessTarget(t, qq, 6, 0, corp+1)
	fresh := model.CorpStructureInfo{CorporationID: corp, StructureID: id, UpdateAt: now.Add(-24 * time.Hour).Unix(), FuelExpires: now.Add(time.Hour).Format(time.RFC3339), StateTimerEnd: now.Add(2 * time.Hour).Format(time.RFC3339)}
	stale := fresh
	stale.StructureID++
	stale.UpdateAt--
	unauthorized := fresh
	unauthorized.StructureID += 2
	unauthorized.CorporationID++
	qqBusinessWrite(t, db, &fresh, &stale, &unauthorized)
	qqBusinessScan(t, qq)
	qqBusinessScan(t, qq)
	rows, count, err := qq.Tasks(ctx, QQOfficialTaskFilter{Source: QQSourceStructure}, 0, 20)
	if err != nil || count != 2 {
		t.Fatalf("expected exactly two first-version scoped alerts; count %d err %v", count, err)
	}
	tasks := make(map[string]model.QQOfficialTask)
	for _, row := range rows {
		rowCorp, structure, kind, version, ok := parseQQStructureReference(row.Reference)
		if !ok || rowCorp != corp || structure != id || version != 1 || row.GroupID != group.ID {
			t.Fatalf("unexpected alert %+v", row)
		}
		if strings.Contains(row.Text, fmt.Sprint(id)) || strings.ContainsAny(row.Text, "@<>") {
			t.Fatalf("private structure ID or unexpected markup in notification: %q", row.Text)
		}
		tasks[kind] = row
		valid, err := qq.businessStillValid(ctx, nil, row, group, *now)
		if !valid || err != nil {
			t.Fatalf("original alert blocked %t %v", valid, err)
		}
		if valid, err := qq.businessStillValid(ctx, nil, row, wrongGroup, *now); valid || err != nil {
			t.Fatalf("wrong corporation allowed %t %v", valid, err)
		}
	}
	if err := db.Model(&model.EveCharacterCorpRole{}).Where("character_id = ?", char).Update("corporation_id", corp+1).Error; err != nil {
		t.Fatal(err)
	}
	if valid, err := qq.businessStillValid(ctx, nil, tasks[qqStructureFuel], group, *now); valid || err != nil {
		t.Fatalf("stale Director provenance allowed %t %v", valid, err)
	}
	if err := db.Model(&model.EveCharacterCorpRole{}).Where("character_id = ?", char).Update("corporation_id", corp).Error; err != nil {
		t.Fatal(err)
	}
	fuel := fresh.FuelExpires
	fresh.FuelExpires = ""
	fresh.StateTimerEnd = now.Add(time.Hour).Format(time.RFC3339)
	qqBusinessWrite(t, db, &fresh)
	for _, task := range tasks {
		if valid, err := qq.businessStillValid(ctx, nil, task, group, *now); valid || err != nil {
			t.Fatalf("missing or changed event allowed %t %v", valid, err)
		}
	}
	qqBusinessScan(t, qq)
	fresh.FuelExpires = fuel
	qqBusinessWrite(t, db, &fresh)
	qqBusinessScan(t, qq)
	fuelState, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureFuel)
	if err != nil || !fuelState.Active || fuelState.Version != 1 {
		t.Fatalf("missing data incorrectly caused a new fuel episode: %+v %v", fuelState, err)
	}
	timerState, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureTimer)
	if err != nil || !timerState.Active || timerState.Version != 2 {
		t.Fatalf("changed timer did not open a new version: %+v %v", timerState, err)
	}
	fuelConfig := model.SystemConfig{Key: model.SysConfigDashboardCorpStructuresFuelNoticeThresholdDays, Value: "0"}
	qqBusinessWrite(t, db, &fuelConfig)
	if valid, err := qq.businessStillValid(ctx, nil, tasks[qqStructureFuel], group, *now); valid || err != nil {
		t.Fatalf("disabled threshold allowed %t %v", valid, err)
	}
	qqBusinessScan(t, qq)
	fuelConfig.Value = "1"
	qqBusinessWrite(t, db, &fuelConfig)
	qqBusinessScan(t, qq)
	fuelState, err = qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureFuel)
	if err != nil || !fuelState.Active || fuelState.Version != 2 {
		t.Fatalf("fuel re-entry did not open a new version: %+v %v", fuelState, err)
	}
}

func TestQQBusinessStructureCapacityRetainsVersionWithoutExtendingExpiry(t *testing.T) {
	qq, db, now := qqBusinessFixture(t)
	ctx := t.Context()
	const corp, id = model.SystemCorporationID, 123
	qqBusinessAuthorizeStructureCorp(t, db, corp, 999)
	group := qqBusinessTarget(t, qq, 7, 0, corp)
	fresh := model.CorpStructureInfo{CorporationID: corp, StructureID: id, UpdateAt: now.Unix(), FuelExpires: now.Add(time.Hour).Format(time.RFC3339)}
	qqBusinessWrite(t, db, &fresh)
	fill := func() { qqBusinessFillStructureQueue(t, db, qq.appID, group.ID, now.Add(time.Hour)) }
	fill()
	qqBusinessScan(t, qq)
	one, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureFuel)
	if err != nil || !one.Active || !one.Pending || one.Version != 1 || !one.ExpiresAt.Equal(now.Add(3*time.Hour)) {
		t.Fatalf("capacity observation not persisted %+v %v", one, err)
	}
	if err := db.Where("source = ?", "fixture").Delete(&model.QQOfficialTask{}).Error; err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Hour)
	qqBusinessScan(t, qq)
	rows, count, err := qq.Tasks(ctx, QQOfficialTaskFilter{Source: QQSourceStructure}, 0, 10)
	if err != nil || count != 1 || len(rows) != 1 || !rows[0].ExpiresAt.Equal(one.ExpiresAt) || rows[0].Reference != qqStructureReference(corp, id, qqStructureFuel, 1) {
		t.Fatalf("retry extended or changed the event %+v %d %v", rows, count, err)
	}
	fill()
	fresh.StructureID++
	qqBusinessWrite(t, db, &fresh)
	qqBusinessScan(t, qq)
	*now = now.Add(qqStructureAlertLifetime)
	qqBusinessScan(t, qq)
	pending, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, fresh.StructureID, qqStructureFuel)
	if err != nil || !pending.Active || pending.Pending || pending.Version != 1 {
		t.Fatalf("expired pending event unexpectedly changed or retained %+v %v", pending, err)
	}
	_, count, err = qq.Tasks(ctx, QQOfficialTaskFilter{Source: QQSourceStructure}, 0, 10)
	if err != nil || count != 1 {
		t.Fatalf("expired pending event replayed: %d %v", count, err)
	}
}

func qqBusinessFillStructureQueue(t *testing.T, db *gorm.DB, appID, groupID string, expires time.Time) {
	t.Helper()
	rows := make([]model.QQOfficialTask, 0, qqGroupQueueMax*4/5)
	for range cap(rows) {
		rows = append(rows, model.QQOfficialTask{ID: uuid.NewString(), IntentID: uuid.NewString(), AppID: appID, GroupID: groupID, Status: model.QQQueued, ExpiresAt: expires, Source: "fixture"})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
}

func TestQQBusinessStructurePendingCannotReplayAfterOptOut(t *testing.T) {
	for _, scope := range []string{"app", "group"} {
		t.Run(scope, func(t *testing.T) {
			qq, db, now := qqBusinessFixture(t)
			ctx := t.Context()
			const corp, id = model.SystemCorporationID, 123
			qqBusinessAuthorizeStructureCorp(t, db, corp, 999)
			group := qqBusinessTarget(t, qq, 7, 0, corp)
			fresh := model.CorpStructureInfo{CorporationID: corp, StructureID: id, UpdateAt: now.Unix(), FuelExpires: now.Add(time.Hour).Format(time.RFC3339)}
			qqBusinessWrite(t, db, &fresh)
			qqBusinessFillStructureQueue(t, db, qq.appID, group.ID, now.Add(time.Hour))
			qqBusinessScan(t, qq)
			prior, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureFuel)
			if err != nil || !prior.Pending {
				t.Fatalf("expected capacity-pending event before opt-out: %+v %v", prior, err)
			}
			for _, enabled := range []bool{false, true} {
				if scope == "app" {
					_, err = qq.UpdateSettings(ctx, 7, QQOfficialSettingsUpdate{Enabled: &enabled})
				} else {
					_, err = qq.UpdateGroup(ctx, 7, group.ID, QQOfficialGroupUpdate{NotificationEnabled: &enabled})
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Where("source = ?", "fixture").Delete(&model.QQOfficialTask{}).Error; err != nil {
				t.Fatal(err)
			}
			qqBusinessScan(t, qq)
			current, err := qq.repo.StructureAlert(ctx, nil, qq.appID, corp, id, qqStructureFuel)
			if err != nil || !current.Active || current.Pending || current.Version != prior.Version {
				t.Fatalf("configuration fence did not close pending version: %+v %v", current, err)
			}
			_, count, err := qq.Tasks(ctx, QQOfficialTaskFilter{Source: QQSourceStructure}, 0, 10)
			if err != nil || count != 0 {
				t.Fatalf("pre-opt-out notification replayed: %d %v", count, err)
			}
		})
	}
}
