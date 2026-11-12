package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrAdmissionDenied         = errors.New("admissions.errors.denied")
	ErrAdmissionInvalid        = errors.New("admissions.errors.invalid")
	ErrAdmissionConflict       = errors.New("admissions.errors.conflict")
	ErrAdmissionWalletDisabled = errors.New("admissions.errors.walletDisabled")
	ErrAdmissionInternal       = errors.New("admissions.errors.internal")
)

type AdmissionWriteInput struct {
	Kind                string  `json:"kind"`
	TargetCorporationID int64   `json:"target_corporation_id"`
	CharacterIDs        []int64 `json:"character_ids"`
	ExpectedVersion     uint    `json:"expected_version"`
}

type AdmissionDetail struct {
	Application *model.AdmissionApplication      `json:"application"`
	Evidence    []model.AdmissionEvidence        `json:"evidence"`
	Events      []model.AdmissionProcessingEvent `json:"events"`
	Salary      *model.AdmissionSalary           `json:"salary"`
	CanReview   bool                             `json:"can_review"`
	IsOwner     bool                             `json:"is_owner"`
}

func admissionWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrAdmissionVersionConflict) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAdmissionConflict
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed: admission_") {
		return ErrAdmissionConflict
	}
	// PostgreSQL unique violations can be returned without GORM translation.
	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) && sqlState.SQLState() == "23505" {
		return ErrAdmissionConflict
	}
	for _, known := range []error{ErrAdmissionDenied, ErrAdmissionInvalid, ErrAdmissionConflict, ErrAdmissionEvidence, ErrAdmissionWalletDisabled} {
		if errors.Is(err, known) {
			return known
		}
	}
	return ErrAdmissionInternal
}

func (s *AdmissionService) validateSubmission(tx *gorm.DB, userID uint, in AdmissionWriteInput) (*model.User, []model.AdmissionCharacter, error) {
	if (in.Kind != model.AdmissionKindNewMember && in.Kind != model.AdmissionKindAlt) || (in.TargetCorporationID != model.SystemCorporationID && in.TargetCorporationID != model.AdmissionFMACorporationID) || len(in.CharacterIDs) == 0 || len(in.CharacterIDs) > 100 {
		return nil, nil, ErrAdmissionInvalid
	}
	user, err := s.repo.ApplicantTx(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	if !user.ProfileComplete() {
		return nil, nil, ErrAdmissionInvalid
	}
	bound, err := s.repo.BoundCharactersTx(tx, userID)
	if err != nil {
		return nil, nil, err
	}
	allowed := map[int64]struct{}{}
	for _, id := range utils.GetAllowCorporations() {
		allowed[id] = struct{}{}
	}
	member := hasAnyAllowedCharacter(bound, allowed)
	if (in.Kind == model.AdmissionKindAlt) != member {
		return nil, nil, ErrAdmissionDenied
	}
	byID := make(map[int64]model.EveCharacter, len(bound))
	for _, char := range bound {
		byID[char.CharacterID] = char
	}
	seen := map[int64]bool{}
	chars := make([]model.AdmissionCharacter, 0, len(in.CharacterIDs))
	for _, id := range in.CharacterIDs {
		char, ok := byID[id]
		if !ok || seen[id] || char.CorporationID == in.TargetCorporationID {
			return nil, nil, ErrAdmissionInvalid
		}
		seen[id] = true
		chars = append(chars, model.AdmissionCharacter{CharacterID: id, CharacterName: char.CharacterName})
	}
	slices.SortFunc(chars, func(a, b model.AdmissionCharacter) int {
		if a.CharacterID < b.CharacterID {
			return -1
		}
		if a.CharacterID > b.CharacterID {
			return 1
		}
		return 0
	})
	return user, chars, nil
}

func (s *AdmissionService) saveEvidenceTx(tx *gorm.DB, app *model.AdmissionApplication, uploads map[string][]byte, requireAll bool) error {
	if app.Kind == model.AdmissionKindAlt {
		if len(uploads) > 0 {
			return ErrAdmissionEvidence
		}
		return nil
	}
	if app.EvidenceExpiresAt != nil && !time.Now().UTC().Before(*app.EvidenceExpiresAt) {
		if err := s.repo.DeleteEvidenceTx(tx, app.ID); err != nil {
			return err
		}
	}
	for kind, data := range uploads {
		evidence, err := NormalizeAdmissionEvidence(kind, data)
		if err != nil {
			return err
		}
		evidence.ApplicationID = app.ID
		if err = s.repo.PutEvidenceTx(tx, evidence); err != nil {
			return err
		}
	}
	if requireAll {
		metadata, err := s.repo.EvidenceMetadata(tx, app.ID)
		if err != nil {
			return err
		}
		if len(metadata) != 2 {
			return ErrAdmissionEvidence
		}
	}
	return nil
}

func (s *AdmissionService) recordAdmissionEvent(tx *gorm.DB, app *model.AdmissionApplication, actor *model.User, action, from, reason string) error {
	snapshot, err := json.Marshal(app)
	if err != nil {
		return err
	}
	event := &model.AdmissionProcessingEvent{ApplicationID: app.ID, Version: app.Version, ActorID: actor.ID, ActorName: actor.Nickname, Action: action, FromStatus: from, ToStatus: app.Status, Reason: reason, Snapshot: string(snapshot)}
	if err = s.repo.AddEventTx(tx, event); err != nil {
		return err
	}
	return NewAuditService().RecordRequiredEventTx(tx, AuditRecordInput{Category: "corporation_admissions", Action: action, ActorUserID: actor.ID, TargetUserID: app.UserID, ResourceType: "admission_application", ResourceID: fmt.Sprint(app.ID), Result: model.AuditResultSuccess, Details: map[string]any{"application": app, "reason": reason, "salary_ref": fmt.Sprintf("admission:%d", app.ID)}})
}

func (s *AdmissionService) Create(userID uint, in AdmissionWriteInput, uploads map[string][]byte) (*model.AdmissionApplication, error) {
	var app *model.AdmissionApplication
	err := s.repo.InTx(func(tx *gorm.DB) error {
		user, chars, err := s.validateSubmission(tx, userID, in)
		if err != nil {
			return err
		}
		app = &model.AdmissionApplication{UserID: userID, Kind: in.Kind, TargetCorporationID: in.TargetCorporationID, Status: model.AdmissionStatusPending, Version: 1, ApplicantName: user.Nickname, PrimaryCharacterID: user.PrimaryCharacterID, QQ: user.QQ, DiscordID: user.DiscordID, SubmittedAt: time.Now().UTC()}
		if err = s.repo.CreateTx(tx, app); err != nil {
			return err
		}
		for i := range chars {
			chars[i].ApplicationID = app.ID
		}
		app.Characters = chars
		if err = s.repo.ReplaceCharactersTx(tx, app.ID, chars); err != nil {
			return err
		}
		if err = s.repo.ClaimCharactersTx(tx, app.ID, chars); err != nil {
			return err
		}
		if err = s.saveEvidenceTx(tx, app, uploads, true); err != nil {
			return err
		}
		return s.recordAdmissionEvent(tx, app, user, "create", "", "")
	})
	return app, admissionWriteError(err)
}

func (s *AdmissionService) Edit(userID, id uint, in AdmissionWriteInput, uploads map[string][]byte, resubmit bool) (*model.AdmissionApplication, error) {
	var app *model.AdmissionApplication
	err := s.repo.InTx(func(tx *gorm.DB) error {
		var err error
		app, err = s.repo.GetByID(tx, id, true)
		if err != nil {
			return ErrAdmissionDenied
		}
		if app.UserID != userID {
			return ErrAdmissionDenied
		}
		if in.ExpectedVersion == 0 || app.Version != in.ExpectedVersion {
			return ErrAdmissionConflict
		}
		if app.Status == model.AdmissionStatusInvited || (resubmit && app.Status != model.AdmissionStatusRejected) || in.Kind != app.Kind {
			return ErrAdmissionInvalid
		}
		user, chars, err := s.validateSubmission(tx, userID, in)
		if err != nil {
			return err
		}
		from := app.Status
		if err = s.saveEvidenceTx(tx, app, uploads, true); err != nil {
			return err
		}
		if err = s.repo.ReleaseClaimsTx(tx, id); err != nil {
			return err
		}
		for i := range chars {
			chars[i].ApplicationID = id
		}
		app.Characters = chars
		if app.Status == model.AdmissionStatusPending || resubmit {
			if err = s.repo.ClaimCharactersTx(tx, id, chars); err != nil {
				return err
			}
		}
		if err = s.repo.ReplaceCharactersTx(tx, id, chars); err != nil {
			return err
		}
		app.TargetCorporationID = in.TargetCorporationID
		app.ApplicantName = user.Nickname
		app.PrimaryCharacterID = user.PrimaryCharacterID
		app.QQ = user.QQ
		app.DiscordID = user.DiscordID
		action := "edit"
		if resubmit {
			action = "resubmit"
			app.Status = model.AdmissionStatusPending
			app.HandledBy = nil
			app.HandlerName = ""
			app.ClosedAt = nil
			app.EvidenceExpiresAt = nil
			app.RejectionReason = ""
			app.SubmittedAt = time.Now().UTC()
		}
		app.Version++
		if err = s.repo.UpdateTx(tx, app, in.ExpectedVersion); err != nil {
			return err
		}
		return s.recordAdmissionEvent(tx, app, user, action, from, "")
	})
	return app, admissionWriteError(err)
}

func (s *AdmissionService) List(userID uint, reviewer bool, status, kind string, target int64, page, size int) ([]model.AdmissionApplication, int64, error) {
	if kind != "" && kind != model.AdmissionKindNewMember && kind != model.AdmissionKindAlt {
		return nil, 0, ErrAdmissionInvalid
	}
	filter := repository.AdmissionListFilter{UserID: userID, TargetCorporationID: target, Kind: kind}
	switch status {
	case "", model.AdmissionStatusPending, model.AdmissionStatusInvited, model.AdmissionStatusRejected:
		if status != "" {
			filter.Statuses = []string{status}
		}
	case "processed":
		filter.Statuses = []string{model.AdmissionStatusInvited, model.AdmissionStatusRejected}
	default:
		return nil, 0, ErrAdmissionInvalid
	}
	var list []model.AdmissionApplication
	var total int64
	err := s.repo.InTx(func(tx *gorm.DB) error {
		if reviewer {
			scope, err := s.reviewableCorporations(tx, userID)
			if err != nil {
				return err
			}
			if len(scope) == 0 {
				return ErrAdmissionDenied
			}
			filter.UserID = 0
			filter.CorporationIDs = scope
		}
		var err error
		list, total, err = s.repo.List(tx, filter, normalizePage(page), normalizeLedgerPageSize(size))
		return err
	})
	return list, total, admissionWriteError(err)
}

func (s *AdmissionService) authorizedApplication(db *gorm.DB, userID, id uint) (*model.AdmissionApplication, bool, error) {
	app, err := s.repo.ReadByIDTx(db, id)
	if err != nil {
		return nil, false, ErrAdmissionDenied
	}
	scope, err := s.reviewableCorporations(db, userID)
	if err != nil {
		return nil, false, ErrAdmissionInternal
	}
	isOwner := app.UserID == userID
	if !isOwner && !slices.Contains(scope, app.TargetCorporationID) {
		return nil, false, ErrAdmissionDenied
	}
	// Owners never review their own application: processing pays the reviewer.
	return app, !isOwner, nil
}

func (s *AdmissionService) Detail(userID, id uint) (*AdmissionDetail, error) {
	var detail *AdmissionDetail
	err := s.repo.InTx(func(tx *gorm.DB) error {
		var err error
		detail, err = s.detailTx(tx, userID, id)
		return err
	})
	return detail, admissionWriteError(err)
}

func (s *AdmissionService) detailTx(tx *gorm.DB, userID, id uint) (*AdmissionDetail, error) {
	app, canReview, err := s.authorizedApplication(tx, userID, id)
	if err != nil {
		return nil, err
	}
	app.Characters, err = s.repo.Characters(tx, id)
	if err != nil {
		return nil, ErrAdmissionInternal
	}
	events, err := s.repo.Events(tx, id)
	if err != nil {
		return nil, ErrAdmissionInternal
	}
	evidence := []model.AdmissionEvidence{}
	if app.EvidenceExpiresAt == nil || time.Now().UTC().Before(*app.EvidenceExpiresAt) {
		evidence, err = s.repo.EvidenceMetadata(tx, id)
		if err != nil {
			return nil, ErrAdmissionInternal
		}
	}
	salary, err := s.repo.Salary(tx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAdmissionInternal
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		salary = nil
	}
	return &AdmissionDetail{Application: app, Evidence: evidence, Events: events, Salary: salary, CanReview: canReview, IsOwner: app.UserID == userID}, nil
}

func (s *AdmissionService) Evidence(userID, id uint, kind string) (*model.AdmissionEvidence, error) {
	var evidence *model.AdmissionEvidence
	err := s.repo.InTx(func(tx *gorm.DB) error {
		var err error
		evidence, err = s.evidenceTx(tx, userID, id, kind)
		return err
	})
	return evidence, admissionWriteError(err)
}

func (s *AdmissionService) evidenceTx(tx *gorm.DB, userID, id uint, kind string) (*model.AdmissionEvidence, error) {
	app, _, err := s.authorizedApplication(tx, userID, id)
	if err != nil {
		return nil, err
	}
	if app.EvidenceExpiresAt != nil && !time.Now().UTC().Before(*app.EvidenceExpiresAt) {
		return nil, ErrAdmissionEvidence
	}
	evidence, err := s.repo.Evidence(tx, id, kind)
	if err != nil {
		return nil, ErrAdmissionEvidence
	}
	return evidence, nil
}

func (s *AdmissionService) Process(actorID, id, version uint, invited bool, reason string) (*model.AdmissionApplication, error) {
	var app *model.AdmissionApplication
	reason = strings.TrimSpace(reason)
	if !invited && (reason == "" || len(reason) > 4000) {
		return nil, ErrAdmissionInvalid
	}
	err := s.repo.InTx(func(tx *gorm.DB) error {
		var err error
		app, err = s.repo.GetByID(tx, id, true)
		if err != nil {
			return ErrAdmissionDenied
		}
		scope, err := s.reviewableCorporations(tx, actorID)
		if err != nil {
			return err
		}
		if app.UserID == actorID || !slices.Contains(scope, app.TargetCorporationID) {
			return ErrAdmissionDenied
		}
		if version == 0 || app.Version != version {
			return ErrAdmissionConflict
		}
		if app.Status != model.AdmissionStatusPending {
			return ErrAdmissionConflict
		}
		actor, err := s.repo.ApplicantTx(tx, actorID)
		if err != nil {
			return err
		}
		bound, err := s.repo.BoundCharactersTx(tx, app.UserID)
		if err != nil {
			return err
		}
		app.Characters, err = s.repo.Characters(tx, id)
		if err != nil {
			return err
		}
		for _, selected := range app.Characters {
			if !slices.ContainsFunc(bound, func(c model.EveCharacter) bool { return c.CharacterID == selected.CharacterID }) {
				return ErrAdmissionInvalid
			}
		}
		now := time.Now().UTC()
		deadline := admissionEvidenceDeadline(now)
		app.HandledBy = &actorID
		app.HandlerName = actor.Nickname
		app.ClosedAt = &now
		app.EvidenceExpiresAt = &deadline
		app.Version++
		app.RejectionReason = reason
		action := "reject"
		app.Status = model.AdmissionStatusRejected
		if invited {
			action = "invited"
			app.Status = model.AdmissionStatusInvited
			app.RejectionReason = ""
			refID := fmt.Sprintf("admission:%d", id)
			amount, salaryErr := salaryAmountTx(tx, model.SysConfigSalaryRecruitment, model.DefaultOperationSalary)
			if salaryErr != nil {
				return salaryErr
			}
			if err = NewSysWalletService().ApplyWalletDeltaByOperatorTx(tx, actorID, actorID, amount, "军团申请 #"+fmt.Sprint(id), model.WalletRefInvitationSalary, refID); err != nil {
				if errors.Is(err, ErrWalletCapabilityDisabled) {
					return ErrAdmissionWalletDisabled
				}
				return err
			}
			if amount > 0 {
				transaction, err := s.repo.SalaryTransactionTx(tx, actorID, refID)
				if err != nil {
					return err
				}
				if err = s.repo.AddSalaryTx(tx, &model.AdmissionSalary{ApplicationID: id, UserID: actorID, Amount: amount, WalletTransactionID: transaction.ID}); err != nil {
					return err
				}
			}
		}
		if err = s.repo.UpdateTx(tx, app, version); err != nil {
			return err
		}
		if err = s.repo.ReleaseClaimsTx(tx, id); err != nil {
			return err
		}
		return s.recordAdmissionEvent(tx, app, actor, action, model.AdmissionStatusPending, reason)
	})
	return app, admissionWriteError(err)
}

func (s *AdmissionService) CleanupEvidence(now time.Time) (int, error) {
	total := 0
	for {
		count := 0
		err := s.repo.InTx(func(tx *gorm.DB) error {
			apps, err := s.repo.ExpiredWithEvidenceTx(tx, now.UTC(), 100)
			if err != nil {
				return err
			}
			for _, app := range apps {
				if err = s.repo.DeleteEvidenceTx(tx, app.ID); err != nil {
					return err
				}
				count++
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += count
		if count < 100 {
			return total, nil
		}
	}
}
