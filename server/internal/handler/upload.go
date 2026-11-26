package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"amiya-eden/internal/apicontract"
	"amiya-eden/pkg/response"
)

const (
	maxUploadSize = 2048 << 10 // 2MB
)

var uploadAllowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// UploadHandler 文件上传处理器
type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func readImageDataURL(file io.Reader, maxBytes int64, allowedMIME map[string]bool) (apicontract.ImageUploadData, *contractOperationFailure) {
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return apicontract.ImageUploadData{}, &contractOperationFailure{response.CodeBizError, "读取文件失败"}
	}
	if int64(len(data)) > maxBytes {
		return apicontract.ImageUploadData{}, &contractOperationFailure{response.CodeParamError, fmt.Sprintf("图片大小不能超过 %d KB", maxBytes>>10)}
	}

	mime := http.DetectContentType(data)
	if !allowedMIME[mime] {
		return apicontract.ImageUploadData{}, &contractOperationFailure{response.CodeParamError, "仅支持 jpeg/png/webp 格式"}
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))
	return apicontract.ImageUploadData{Url: dataURL}, nil
}

func (h *UploadHandler) UploadImage(_ context.Context, request apicontract.UploadImageRequestObject) (apicontract.UploadImageResponseObject, error) {
	data, failure := readStrictImageUpload(request.Body, maxUploadSize, uploadAllowedMIME)
	out := &apicontract.UploadImage200JSONResponse{}
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromImageUploadSuccess(apicontract.ImageUploadSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func (h *UploadHandler) UploadEvidence(_ context.Context, request apicontract.UploadEvidenceRequestObject) (apicontract.UploadEvidenceResponseObject, error) {
	data, failure := readStrictImageUpload(request.Body, evidenceMaxBytes, evidenceAllowedMIME)
	out := &apicontract.UploadEvidence200JSONResponse{}
	if failure != nil {
		return out, out.FromBusinessError(contractBusinessError(failure.code, failure.message))
	}
	return out, out.FromImageUploadSuccess(apicontract.ImageUploadSuccess{Code: response.CodeOK, Msg: "success", Data: data})
}

func readStrictImageUpload(reader *multipart.Reader, maxBytes int64, allowedMIME map[string]bool) (apicontract.ImageUploadData, *contractOperationFailure) {
	fileFailure := func(err error) (apicontract.ImageUploadData, *contractOperationFailure) {
		return apicontract.ImageUploadData{}, &contractOperationFailure{response.CodeParamError, "获取文件失败: " + err.Error()}
	}
	if reader == nil {
		return fileFailure(http.ErrMissingFile)
	}
	// The generated binder limits the entire body; preserve the legacy FormFile
	// behavior of selecting the first file in the file field.
	form, err := reader.ReadForm(maxBytes + 1024)
	if err != nil {
		return fileFailure(err)
	}
	defer func() { _ = form.RemoveAll() }()
	files := form.File["file"]
	if len(files) == 0 {
		return fileFailure(http.ErrMissingFile)
	}
	file, err := files[0].Open()
	if err != nil {
		return fileFailure(err)
	}
	defer closeUploadFile(file)
	return readImageDataURL(file, maxBytes, allowedMIME)
}
