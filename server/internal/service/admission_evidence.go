package service

import (
	"amiya-eden/internal/model"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"time"
)

const AdmissionEvidenceMaxBytes = 2 << 20

var ErrAdmissionEvidence = errors.New("admissions.errors.evidence")

// NormalizeAdmissionEvidence validates before allocating decoded pixels and
// always re-encodes, stripping source metadata and keeping no original bytes.
func NormalizeAdmissionEvidence(kind string, data []byte) (*model.AdmissionEvidence, error) {
	if (kind != model.AdmissionEvidenceLauncher && kind != model.AdmissionEvidenceSeatProfile) || len(data) == 0 || len(data) > AdmissionEvidenceMaxBytes {
		return nil, ErrAdmissionEvidence
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > 24000000 {
		return nil, ErrAdmissionEvidence
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrAdmissionEvidence
	}
	w, h := cfg.Width, cfg.Height
	if w > 1280 || h > 1280 {
		if w >= h {
			h = max(1, h*1280/w)
			w = 1280
		} else {
			w = max(1, w*1280/h)
			h = 1280
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	bounds := src.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Composite transparent input onto white before JPEG conversion.
			r, g, b, a := src.At(bounds.Min.X+x*cfg.Width/w, bounds.Min.Y+y*cfg.Height/h).RGBA()
			dst.SetRGBA(x, y, color.RGBA{uint8((r + 65535 - a) >> 8), uint8((g + 65535 - a) >> 8), uint8((b + 65535 - a) >> 8), 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, dst, &jpeg.Options{Quality: 70}); err != nil || encoded.Len() > AdmissionEvidenceMaxBytes {
		return nil, ErrAdmissionEvidence
	}
	return &model.AdmissionEvidence{Kind: kind, Data: encoded.Bytes(), Width: w, Height: h, ByteSize: encoded.Len()}, nil
}

// AddDate alone overflows month ends. Closed evidence expires at the same UTC
// time on the last valid day when the destination month is shorter.
func admissionEvidenceDeadline(closed time.Time) time.Time {
	closed = closed.UTC()
	month := time.Date(closed.Year(), closed.Month()+3, 1, closed.Hour(), closed.Minute(), closed.Second(), closed.Nanosecond(), time.UTC)
	last := month.AddDate(0, 1, -1).Day()
	return month.AddDate(0, 0, min(closed.Day(), last)-1)
}
