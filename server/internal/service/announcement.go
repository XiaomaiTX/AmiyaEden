package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	xhtml "golang.org/x/net/html"
	"gorm.io/gorm"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"strings"
	"time"
	"unicode/utf8"
)

// Announcement errors are mapped to user-visible messages by the handler.
var (
	ErrAnnouncementInvalid       = errors.New("announcement title or body invalid")
	ErrAnnouncementImage         = errors.New("announcement image invalid")
	ErrAnnouncementTooManyImages = errors.New("announcement has too many images")
	ErrAnnouncementNotFound      = errors.New("announcement not found")
	ErrAnnouncementForbidden     = errors.New("announcement management forbidden")
	ErrAnnouncementConflict      = errors.New("announcement changed by another save")
	ErrAnnouncementStale         = errors.New("announcement version changed")
)

const announcementMaxImages = 6

type AnnouncementService struct {
	repo *repository.AnnouncementRepository
}

func NewAnnouncementService() *AnnouncementService {
	return &AnnouncementService{repository.NewAnnouncementRepository()}
}
func CanManageAnnouncements(roles []string) bool {
	return model.ContainsAnyRole(roles, model.RoleSuperAdmin, model.RoleAdmin, model.RoleSeniorFC)
}

type AnnouncementInput struct {
	Title    string `json:"title"`
	BodyHTML string `json:"body_html"`
	Hidden   bool   `json:"hidden"`
	Remind   bool   `json:"remind"`
	Revision uint   `json:"revision"`
}

func (s *AnnouncementService) List(user uint, roles []string, page int) (interface{}, error) {
	rows, total, err := s.repo.List(user, CanManageAnnouncements(roles), page)
	if rows == nil {
		rows = []repository.AnnouncementRow{}
	}
	return struct {
		List  []repository.AnnouncementRow `json:"list"`
		Total int64                        `json:"total"`
	}{rows, total}, err
}
func (s *AnnouncementService) Detail(id uint, roles []string) (*model.Announcement, error) {
	a, images, err := s.repo.Load(id)
	if err != nil {
		return nil, announcementLoadError(err)
	}
	if a.Hidden && !CanManageAnnouncements(roles) {
		return nil, ErrAnnouncementNotFound
	}
	for _, im := range images {
		a.BodyHTML = strings.ReplaceAll(a.BodyHTML, "announcement-image:"+im.Digest, "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(im.Data))
	}
	return a, nil
}
func (s *AnnouncementService) Save(user, id uint, roles []string, in AnnouncementInput) (*model.Announcement, error) {
	if !CanManageAnnouncements(roles) {
		return nil, ErrAnnouncementForbidden
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > 200 || len(in.BodyHTML) > 70<<20 {
		return nil, ErrAnnouncementInvalid
	}
	var existing []model.AnnouncementImage
	var err error
	a := &model.Announcement{AuthorID: user, Version: 1}
	if id != 0 {
		a, existing, err = s.repo.Load(id)
		if err != nil {
			return nil, announcementLoadError(err)
		}
		if a.Revision != in.Revision {
			return nil, ErrAnnouncementConflict
		}
		if in.Remind {
			a.Version++
		}
	}
	body, images, err := normalizeAnnouncementHTML(in.BodyHTML, existing...)
	if err != nil {
		return nil, err
	}
	a.Title = title
	a.BodyHTML = body
	a.Hidden = in.Hidden
	a.Revision++
	a.UpdatedAt = time.Now().UTC()
	if err = s.repo.Save(a, in.Revision, images); err != nil {
		if errors.Is(err, repository.ErrAnnouncementRevision) {
			return nil, ErrAnnouncementConflict
		}
		return nil, err
	}
	return s.Detail(a.ID, roles)
}

// Read records the version the user saw. A no-op insert is success only when that version is still current.
func (s *AnnouncementService) Read(user, id, version uint) error {
	changed, err := s.repo.Read(id, user, version)
	if err != nil || changed {
		return err
	}
	a, err := s.repo.State(id)
	if err != nil {
		return announcementLoadError(err)
	}
	if a.Hidden {
		return ErrAnnouncementNotFound
	}
	if a.Version != version {
		return ErrAnnouncementStale
	}
	return nil
}

func announcementLoadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAnnouncementNotFound
	}
	return err
}

// Decode only bounded PNG/JPEG input, strip metadata and encode a durable JPEG.
func normalizeAnnouncementImage(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > 8<<20 {
		return nil, ErrAnnouncementImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > 24000000 {
		return nil, ErrAnnouncementImage
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrAnnouncementImage
	}
	w, h := cfg.Width, cfg.Height
	if w > 2560 || h > 2560 {
		if w >= h {
			h = max(1, h*2560/w)
			w = 2560
		} else {
			w = max(1, w*2560/h)
			h = 2560
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := src.At(b.Min.X+x*cfg.Width/w, b.Min.Y+y*cfg.Height/h).RGBA()
			dst.SetRGBA(x, y, color.RGBA{uint8((r + 65535 - a) >> 8), uint8((g + 65535 - a) >> 8), uint8((bl + 65535 - a) >> 8), 255})
		}
	}
	var out bytes.Buffer
	if err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil || out.Len() > 4<<20 {
		return nil, ErrAnnouncementImage
	}
	return out.Bytes(), nil
}
func normalizeAnnouncementHTML(input string, existing ...model.AnnouncementImage) (string, []model.AnnouncementImage, error) {
	retained := make(map[string][]byte, len(existing))
	for _, im := range existing {
		retained[im.Digest] = im.Data
	}
	clean := sanitizeRichTextHTML(input)
	doc, err := xhtml.Parse(strings.NewReader("<div>" + clean + "</div>"))
	if err != nil {
		return "", nil, ErrAnnouncementInvalid
	}
	root := findFirstHTMLElement(doc, "div")
	images := []model.AnnouncementImage{}
	textSize := 0
	// Whitespace (including &nbsp;) still counts toward the size bound but does not make a body non-empty.
	hasText := false
	var visit func(*xhtml.Node) error
	visit = func(n *xhtml.Node) error {
		if n.Type == xhtml.TextNode {
			textSize += len(n.Data)
			hasText = hasText || strings.TrimSpace(n.Data) != ""
		}
		if n.Type == xhtml.ElementNode && n.Data == "img" {
			var src string
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					src = attr.Val
				}
			}
			if len(images) >= announcementMaxImages {
				return ErrAnnouncementTooManyImages
			}
			parts := strings.SplitN(src, ",", 2)
			if len(parts) != 2 || (parts[0] != "data:image/png;base64" && parts[0] != "data:image/jpeg;base64") {
				return ErrAnnouncementImage
			}
			raw, e := base64.StdEncoding.DecodeString(parts[1])
			if e != nil {
				return ErrAnnouncementImage
			}
			rawHash := sha256.Sum256(raw)
			data := retained[hex.EncodeToString(rawHash[:])]
			if data == nil {
				data, e = normalizeAnnouncementImage(raw)
				if e != nil {
					return e
				}
			}
			hash := sha256.Sum256(data)
			digest := hex.EncodeToString(hash[:])
			images = append(images, model.AnnouncementImage{Digest: digest, Data: data})
			n.Attr = []xhtml.Attribute{{Key: "src", Val: "announcement-image:" + digest}}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if e := visit(child); e != nil {
				return e
			}
		}
		return nil
	}
	if err = visit(root); err != nil {
		return "", nil, err
	}
	if textSize > 128000 || (!hasText && len(images) == 0) {
		return "", nil, ErrAnnouncementInvalid
	}
	var out bytes.Buffer
	for n := root.FirstChild; n != nil; n = n.NextSibling {
		if err = xhtml.Render(&out, n); err != nil {
			return "", nil, fmt.Errorf("render announcement: %w", err)
		}
	}
	return out.String(), images, nil
}
