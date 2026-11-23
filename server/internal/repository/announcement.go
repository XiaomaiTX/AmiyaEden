package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrAnnouncementRevision means another save changed the announcement first.
var ErrAnnouncementRevision = errors.New("announcement revision changed")

type AnnouncementRepository struct{ db *gorm.DB }

func NewAnnouncementRepository() *AnnouncementRepository { return &AnnouncementRepository{global.DB} }
func NewAnnouncementRepositoryWithDB(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db}
}
func (r *AnnouncementRepository) List(user uint, manage bool, page int) ([]AnnouncementRow, int64, error) {
	q := r.db.Model(&model.Announcement{})
	if !manage {
		q = q.Where("hidden = ?", false)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AnnouncementRow
	err := q.Select("announcement.id, announcement.title, announcement.author_id, announcement.hidden, announcement.version, announcement.revision, announcement.created_at, announcement.updated_at, COALESCE(ar.version, 0) < announcement.version AS unread, "+announcementAuthorColumns).
		Joins(announcementAuthorJoin).
		Joins(announcementCharacterJoin).
		Joins("LEFT JOIN announcement_read ar ON ar.announcement_id = announcement.id AND ar.user_id = ?", user).
		Order("announcement.created_at DESC, announcement.id DESC").Limit(10).Offset((page - 1) * 10).Scan(&rows).Error
	return rows, total, err
}

type AnnouncementRow struct {
	model.Announcement
	AnnouncementAuthor
	Unread bool `json:"unread"`
}

// Only public profile fields are exposed; contact details and SSO data stay private.
type AnnouncementAuthor struct {
	AuthorNickname      string `json:"author_nickname"`
	AuthorCharacterID   int64  `json:"author_character_id"`
	AuthorCharacterName string `json:"author_character_name"`
}

const announcementAuthorColumns = `COALESCE(au.nickname, '') AS author_nickname, COALESCE(ac.character_id, 0) AS author_character_id, COALESCE(ac.character_name, '') AS author_character_name`
const announcementAuthorJoin = `LEFT JOIN "user" au ON au.id = announcement.author_id AND au.deleted_at IS NULL`
const announcementCharacterJoin = `LEFT JOIN eve_character ac ON ac.character_id = au.primary_character_id AND ac.user_id = au.id AND ac.deleted_at IS NULL`

func (r *AnnouncementRepository) Author(id uint) (AnnouncementAuthor, error) {
	var author AnnouncementAuthor
	err := r.db.Table("announcement").Select(announcementAuthorColumns).
		Joins(announcementAuthorJoin).Joins(announcementCharacterJoin).
		Where("announcement.id = ?", id).Scan(&author).Error
	return author, err
}

func (r *AnnouncementRepository) Save(a *model.Announcement, expected uint, images []model.AnnouncementImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if a.ID == 0 {
			if err := tx.Create(a).Error; err != nil {
				return err
			}
		} else {
			result := tx.Model(a).Where("revision = ?", expected).Select("title", "body_html", "hidden", "version", "revision", "updated_at").Updates(a)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrAnnouncementRevision
			}
		}
		if err := tx.Where("announcement_id = ?", a.ID).Delete(&model.AnnouncementImage{}).Error; err != nil {
			return err
		}
		for _, im := range images {
			im.AnnouncementID = a.ID
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&im).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Read reports whether a row changed; zero means either already read or no visible announcement at that version.
func (r *AnnouncementRepository) Read(id, user, version uint) (bool, error) {
	result := r.db.Exec("INSERT INTO announcement_read (announcement_id, user_id, version) SELECT id, ?, version FROM announcement WHERE id = ? AND version = ? AND hidden = ? ON CONFLICT (announcement_id, user_id) DO UPDATE SET version = excluded.version WHERE announcement_read.version < excluded.version", user, id, version, false)
	return result.RowsAffected > 0, result.Error
}

// State loads only the fields needed to explain a no-op read.
func (r *AnnouncementRepository) State(id uint) (*model.Announcement, error) {
	var a model.Announcement
	err := r.db.Select("id", "hidden", "version").First(&a, id).Error
	return &a, err
}
func (r *AnnouncementRepository) Unread(user uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Announcement{}).Joins("LEFT JOIN announcement_read ar ON ar.announcement_id = announcement.id AND ar.user_id = ?", user).
		Where("announcement.hidden = ? AND COALESCE(ar.version, 0) < announcement.version", false).Count(&count).Error
	return count, err
}

// Load reads the body and its images as one snapshot: the SHARE row lock blocks a concurrent
// Save between the two reads. Lost updates are prevented separately by Save's revision check.
func (r *AnnouncementRepository) Load(id uint) (*model.Announcement, []model.AnnouncementImage, error) {
	var a model.Announcement
	var images []model.AnnouncementImage
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&a, id).Error; err != nil {
			return err
		}
		return tx.Where("announcement_id = ?", id).Find(&images).Error
	})
	return &a, images, err
}
