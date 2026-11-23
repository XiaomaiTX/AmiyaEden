package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestAnnouncementPublisherProfile(t *testing.T) {
	db := newServiceTestDB(t, "announcement_author", &model.User{}, &model.EveCharacter{}, &model.Announcement{}, &model.AnnouncementRead{}, &model.AnnouncementImage{})
	publisher := model.User{Nickname: "发布者昵称", QQ: "private-contact", PrimaryCharacterID: 90000001}
	editor := model.User{Nickname: "编辑者"}
	for _, user := range []*model.User{&publisher, &editor} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	character := model.EveCharacter{CharacterID: publisher.PrimaryCharacterID, CharacterName: "Primary Pilot", UserID: publisher.ID}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	s := &AnnouncementService{repository.NewAnnouncementRepositoryWithDB(db)}
	roles := []string{model.RoleAdmin}
	a, err := s.Save(publisher.ID, 0, roles, AnnouncementInput{Title: "Notice", BodyHTML: "<p>Body</p>"})
	if err != nil {
		t.Fatal(err)
	}
	assertAuthor := func(nickname, name string, characterID int64) {
		t.Helper()
		detail, detailErr := s.Detail(a.ID, []string{model.RoleUser})
		rows, total, listErr := s.repo.List(editor.ID, false, 1)
		if detailErr != nil || listErr != nil || total != 1 || len(rows) != 1 {
			t.Fatalf("load author: detail=%v list=%v total=%d rows=%d", detailErr, listErr, total, len(rows))
		}
		for _, author := range []repository.AnnouncementAuthor{detail.AnnouncementAuthor, rows[0].AnnouncementAuthor} {
			if author.AuthorNickname != nickname || author.AuthorCharacterName != name || author.AuthorCharacterID != characterID {
				t.Fatalf("unexpected author: %+v", author)
			}
		}
		if detail.AuthorID != publisher.ID || rows[0].AuthorID != publisher.ID {
			t.Fatal("editor replaced publisher")
		}
		encoded, marshalErr := json.Marshal(detail)
		if marshalErr != nil || strings.Contains(string(encoded), "private-contact") {
			t.Fatal("private profile data leaked", marshalErr)
		}
	}
	if a.AuthorNickname != publisher.Nickname || a.AuthorCharacterName != character.CharacterName {
		t.Fatal("save response missing author")
	}
	assertAuthor(publisher.Nickname, character.CharacterName, character.CharacterID)
	if _, err = s.Save(editor.ID, a.ID, roles, AnnouncementInput{Title: "Edited", BodyHTML: "<p>Edited</p>", Revision: a.Revision}); err != nil {
		t.Fatal(err)
	}
	assertAuthor(publisher.Nickname, character.CharacterName, character.CharacterID)
	// A transferred character must no longer be presented as the publisher's primary.
	if err = db.Model(&character).Update("user_id", editor.ID).Error; err != nil {
		t.Fatal(err)
	}
	assertAuthor(publisher.Nickname, "", 0)
	if err = db.Delete(&publisher).Error; err != nil {
		t.Fatal(err)
	}
	assertAuthor("", "", 0)
}

func TestAnnouncementLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.Announcement{}, &model.AnnouncementRead{}, &model.AnnouncementImage{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAnnouncementRepositoryWithDB(db)
	s := &AnnouncementService{repo}
	roles := []string{model.RoleSeniorFC}
	in := AnnouncementInput{Title: "Fleet", BodyHTML: "<p>Hello<script>bad()</script></p>"}
	if _, err = s.Save(1, 0, []string{model.RoleUser}, in); err == nil {
		t.Fatal("member published")
	}
	a, err := s.Save(1, 0, roles, in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(a.BodyHTML, "script") {
		t.Fatal("unsafe HTML")
	}
	check := func(user uint, want int64) {
		t.Helper()
		got, e := repo.Unread(user)
		if e != nil || got != want {
			t.Fatalf("unread got %d want %d: %v", got, want, e)
		}
	}
	check(2, 1)
	if err = s.Read(2, a.ID, a.Version); err != nil {
		t.Fatal(err)
	}
	check(2, 0)
	check(3, 1)
	in.Revision = a.Revision
	a, err = s.Save(1, a.ID, roles, in)
	if err != nil {
		t.Fatal(err)
	}
	check(2, 0)
	if _, err = s.Save(1, a.ID, roles, in); !errors.Is(err, ErrAnnouncementConflict) {
		t.Fatal("stale overwrite not reported as conflict", err)
	}
	in.Revision = a.Revision
	in.Remind = true
	a, err = s.Save(1, a.ID, roles, in)
	if err != nil {
		t.Fatal(err)
	}
	check(2, 1)
	if err = s.Read(2, a.ID, 1); !errors.Is(err, ErrAnnouncementStale) {
		t.Fatal("stale read not reported", err)
	}
	check(2, 1)
	if err = s.Read(2, a.ID+100, 1); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Fatal("missing announcement read not reported", err)
	}
	in.Revision = a.Revision
	in.Remind = false
	in.Hidden = true
	a, err = s.Save(1, a.ID, roles, in)
	if err != nil {
		t.Fatal(err)
	}
	check(2, 0)
	if _, err = s.Detail(a.ID, []string{model.RoleUser}); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Fatal("hidden content leaked", err)
	}
	if err = s.Read(2, a.ID, a.Version); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Fatal("hidden announcement read not reported", err)
	}
	rows, total, err := repo.List(2, false, 1)
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatal("hidden list leaked", err)
	}
	in.Revision = a.Revision
	in.Hidden = false
	a, err = s.Save(1, a.ID, roles, in)
	if err != nil {
		t.Fatal(err)
	}
	check(2, 1)
	if err = s.Read(2, a.ID, a.Version); err != nil {
		t.Fatal(err)
	}
	check(2, 0)
	rows, _, err = repo.List(2, false, 1)
	if err != nil || len(rows) != 1 || rows[0].Unread || rows[0].BodyHTML != "" {
		t.Fatal("list contract", err)
	}
	if err = s.Read(2, a.ID, a.Version); err != nil {
		t.Fatal("repeat read of current version failed", err)
	}
}
func TestAnnouncementPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.Announcement{}, &model.AnnouncementRead{}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 21; i++ {
		if err = db.Create(&model.Announcement{Title: fmt.Sprintf("Notice %d", i), Version: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := repository.NewAnnouncementRepositoryWithDB(db)
	seen := make(map[uint]bool)
	for page, want := range []int{10, 10, 1} {
		rows, total, listErr := repo.List(2, false, page+1)
		if listErr != nil || total != 21 || len(rows) != want {
			t.Fatalf("page %d: got %d rows, total %d: %v", page+1, len(rows), total, listErr)
		}
		for _, row := range rows {
			if seen[row.ID] {
				t.Fatalf("announcement %d repeated across pages", row.ID)
			}
			seen[row.ID] = true
		}
	}
	if len(seen) != 21 {
		t.Fatalf("pagination omitted announcements: got %d", len(seen))
	}
}

func TestAnnouncementImages(t *testing.T) {
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
	body, images, err := normalizeAnnouncementHTML("<p>Hi</p><img src=\"" + uri + "\" onerror=\"bad()\">")
	if err != nil || len(images) != 1 || strings.Contains(body, "onerror") || !strings.Contains(body, "announcement-image:") {
		t.Fatal(body, err)
	}
	// A normal edit reuses already stored bytes without another lossy encode.
	hydrated := "<img src=\"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(images[0].Data) + "\">"
	_, reused, reuseErr := normalizeAnnouncementHTML(hydrated, images...)
	if reuseErr != nil || len(reused) != 1 || !bytes.Equal(images[0].Data, reused[0].Data) {
		t.Fatal("stored image changed", reuseErr)
	}
	for _, bad := range []string{"<img src=\"https://example.com/a.png\">", "<img src=\"data:image/png;base64,bad\">"} {
		if _, _, err = normalizeAnnouncementHTML(bad); !errors.Is(err, ErrAnnouncementImage) {
			t.Fatal("invalid image accepted", err)
		}
	}
	if _, _, err = normalizeAnnouncementHTML(strings.Repeat("<img src=\""+uri+"\">", 7)); !errors.Is(err, ErrAnnouncementTooManyImages) {
		t.Fatal("image limit not reported", err)
	}
}
func TestAnnouncementRejectsBlankBody(t *testing.T) {
	for _, blank := range []string{"", "<p>   </p>", "<p>\n</p>", "<p>&nbsp;</p><p><br></p>"} {
		if _, _, err := normalizeAnnouncementHTML(blank); !errors.Is(err, ErrAnnouncementInvalid) {
			t.Fatalf("blank body %q accepted: %v", blank, err)
		}
	}
	if _, _, err := normalizeAnnouncementHTML("<p> Hi </p>"); err != nil {
		t.Fatal("padded text rejected", err)
	}
}
func TestAnnouncementImagePersistence(t *testing.T) {
	db := newServiceTestDB(t, "announcement_images", &model.User{}, &model.EveCharacter{}, &model.Announcement{}, &model.AnnouncementRead{}, &model.AnnouncementImage{})
	s := &AnnouncementService{repository.NewAnnouncementRepositoryWithDB(db)}
	roles := []string{model.RoleAdmin}
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 100, 100))); err != nil {
		t.Fatal(err)
	}
	input := AnnouncementInput{Title: "Screenshot", BodyHTML: "<img src=\"data:image/png;base64," + base64.StdEncoding.EncodeToString(picture.Bytes()) + "\">"}
	a, err := s.Save(1, 0, roles, input)
	if err != nil {
		t.Fatal(err)
	}
	stored, images, err := s.repo.Load(a.ID)
	if err != nil || len(images) != 1 || strings.Contains(stored.BodyHTML, "data:") {
		t.Fatal("image persistence", err)
	}
	input.BodyHTML = a.BodyHTML
	input.Revision = a.Revision
	input.Hidden = true
	a, err = s.Save(1, a.ID, roles, input)
	if err != nil {
		t.Fatal(err)
	}
	_, retained, err := s.repo.Load(a.ID)
	if err != nil || len(retained) != 1 || !bytes.Equal(images[0].Data, retained[0].Data) {
		t.Fatal("hidden image lost", err)
	}
	input.Revision = a.Revision
	input.Hidden = false
	a, err = s.Save(1, a.ID, roles, input)
	if err != nil {
		t.Fatal(err)
	}
	fresh := &AnnouncementService{repository.NewAnnouncementRepositoryWithDB(db)}
	loaded, err := fresh.Detail(a.ID, []string{model.RoleUser})
	if err != nil || !strings.Contains(loaded.BodyHTML, "data:image/jpeg;base64,") {
		t.Fatal("image cannot be read", err)
	}
}
