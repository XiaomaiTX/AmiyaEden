package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdmissionHandler struct{ svc *service.AdmissionService }

func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{svc: service.NewAdmissionService()}
}

func admissionFailure(c *gin.Context, err error) {
	code := response.CodeBizError
	if errors.Is(err, service.ErrAdmissionDenied) {
		code = response.CodeForbidden
	}
	response.Fail(c, code, err.Error())
}

func (h *AdmissionHandler) Context(c *gin.Context) {
	result, err := h.svc.Context(middleware.GetUserID(c))
	if err != nil {
		admissionFailure(c, service.ErrAdmissionInternal)
		return
	}
	response.OK(c, result)
}

func (h *AdmissionHandler) ListMine(c *gin.Context)   { h.list(c, false) }
func (h *AdmissionHandler) ListReview(c *gin.Context) { h.list(c, true) }
func (h *AdmissionHandler) list(c *gin.Context, reviewer bool) {
	page, size, err := parsePaginationQuery(c, 20, 100)
	if err != nil {
		response.Fail(c, response.CodeParamError, "admissions.errors.invalid")
		return
	}
	var target int64
	if raw := c.Query("target_corporation_id"); raw != "" {
		target, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || target <= 0 {
			response.Fail(c, response.CodeParamError, "admissions.errors.invalid")
			return
		}
	}
	list, total, err := h.svc.List(middleware.GetUserID(c), reviewer, c.Query("status"), c.Query("kind"), target, page, size)
	if err != nil {
		admissionFailure(c, err)
		return
	}
	response.OKWithPage(c, list, total, page, size)
}

func (h *AdmissionHandler) Detail(c *gin.Context) {
	id := requireUintID(c, "id", "application")
	if id == 0 {
		return
	}
	result, err := h.svc.Detail(middleware.GetUserID(c), id)
	if err != nil {
		admissionFailure(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AdmissionHandler) Evidence(c *gin.Context) {
	id := requireUintID(c, "id", "application")
	if id == 0 {
		return
	}
	result, err := h.svc.Evidence(middleware.GetUserID(c), id, c.Param("kind"))
	if err != nil {
		admissionFailure(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/jpeg", result.Data)
}

func admissionMultipart(c *gin.Context) (service.AdmissionWriteInput, map[string][]byte, error) {
	var input service.AdmissionWriteInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)
	if err := c.Request.ParseMultipartForm(5 << 20); err != nil {
		return input, nil, service.ErrAdmissionEvidence
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	if err := json.Unmarshal([]byte(c.PostForm("payload")), &input); err != nil {
		return input, nil, service.ErrAdmissionInvalid
	}
	uploads := map[string][]byte{}
	for kind, files := range c.Request.MultipartForm.File {
		if len(files) != 1 {
			return input, nil, service.ErrAdmissionEvidence
		}
		file, err := files[0].Open()
		if err != nil {
			return input, nil, service.ErrAdmissionEvidence
		}
		data, readErr := io.ReadAll(io.LimitReader(file, service.AdmissionEvidenceMaxBytes+1))
		_ = file.Close()
		if readErr != nil || len(data) > service.AdmissionEvidenceMaxBytes {
			return input, nil, service.ErrAdmissionEvidence
		}
		uploads[kind] = data
	}
	return input, uploads, nil
}

func (h *AdmissionHandler) Create(c *gin.Context)   { h.write(c, false, false) }
func (h *AdmissionHandler) Edit(c *gin.Context)     { h.write(c, true, false) }
func (h *AdmissionHandler) Resubmit(c *gin.Context) { h.write(c, true, true) }
func (h *AdmissionHandler) write(c *gin.Context, edit, resubmit bool) {
	var id uint
	if edit {
		id = requireUintID(c, "id", "application")
		if id == 0 {
			return
		}
	}
	input, uploads, err := admissionMultipart(c)
	if err != nil {
		admissionFailure(c, err)
		return
	}
	if edit {
		result, err := h.svc.Edit(middleware.GetUserID(c), id, input, uploads, resubmit)
		if err != nil {
			admissionFailure(c, err)
			return
		}
		response.OK(c, result)
	} else {
		result, err := h.svc.Create(middleware.GetUserID(c), input, uploads)
		if err != nil {
			admissionFailure(c, err)
			return
		}
		response.OK(c, result)
	}
}

func (h *AdmissionHandler) Invited(c *gin.Context) { h.process(c, true) }
func (h *AdmissionHandler) Reject(c *gin.Context)  { h.process(c, false) }
func (h *AdmissionHandler) process(c *gin.Context, invited bool) {
	id := requireUintID(c, "id", "application")
	if id == 0 {
		return
	}
	var input struct {
		ExpectedVersion uint   `json:"expected_version" binding:"required"`
		Reason          string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		admissionFailure(c, service.ErrAdmissionInvalid)
		return
	}
	result, err := h.svc.Process(middleware.GetUserID(c), id, input.ExpectedVersion, invited, input.Reason)
	if err != nil {
		admissionFailure(c, err)
		return
	}
	response.OK(c, result)
}
