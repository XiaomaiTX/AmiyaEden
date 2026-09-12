package handler

import (
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

type MumbleHandler struct {
	svc *service.MumbleIdentityService
}

func NewMumbleHandler() *MumbleHandler {
	return &MumbleHandler{svc: service.NewMumbleIdentityService()}
}

func (h *MumbleHandler) GetCredential(c *gin.Context) {
	status, err := h.svc.GetCredentialStatus(middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, "加载 Mumble 凭据状态失败")
		return
	}
	response.OK(c, status)
}

func (h *MumbleHandler) CreateCredential(c *gin.Context) {
	status, password, err := h.svc.CreateCredential(middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{"credential": status, "password": password})
}

func (h *MumbleHandler) RotateCredential(c *gin.Context) {
	status, password, err := h.svc.RotateCredential(middleware.GetUserID(c))
	if err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, gin.H{"credential": status, "password": password})
}

func (h *MumbleHandler) RevokeCredential(c *gin.Context) {
	if err := h.svc.RevokeCredential(middleware.GetUserID(c)); err != nil {
		response.Fail(c, response.CodeBizError, err.Error())
		return
	}
	response.OK(c, nil)
}

type mumbleAuthenticateRequest struct {
	ServerInstanceID string `json:"server_instance_id" binding:"required,max=128"`
	Username         string `json:"username" binding:"required,max=128"`
	Password         string `json:"password" binding:"required,max=1024"`
	CertificateHash  string `json:"certificate_hash" binding:"max=128"`
	RemoteIP         string `json:"remote_ip" binding:"max=64"`
}

// Authenticate is intentionally a narrow service-to-service endpoint. It
// never exposes a distinct response for missing users and wrong passwords.
func (h *MumbleHandler) Authenticate(c *gin.Context) {
	var req mumbleAuthenticateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	claims, err := h.svc.Authenticate(req.Username, req.Password)
	if err != nil || !claims.Eligible {
		response.OK(c, gin.H{"decision": "deny", "reason_code": "INVALID_CREDENTIALS"})
		return
	}
	response.OK(c, gin.H{"decision": "allow", "user_id": claims.StableUserID, "name": claims.Name, "groups": claims.Groups, "identity_version": claims.IdentityVersion, "policy_version": claims.PolicyVersion})
}

type mumbleResolveRequest struct {
	UserIDs []uint32 `json:"user_ids"`
	Names   []string `json:"names"`
}

func (h *MumbleHandler) ResolveIdentities(c *gin.Context) {
	var req mumbleResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UserIDs)+len(req.Names) == 0 || len(req.UserIDs)+len(req.Names) > 500 {
		response.Fail(c, response.CodeParamError, "请求参数错误")
		return
	}
	identities := make([]gin.H, 0, len(req.UserIDs)+len(req.Names))
	for _, id := range req.UserIDs {
		claims, err := h.svc.Resolve(id)
		if err != nil || !claims.Eligible {
			identities = append(identities, gin.H{"user_id": id, "eligible": false})
			continue
		}
		identities = append(identities, gin.H{"user_id": id, "eligible": true, "name": claims.Name, "groups": claims.Groups, "identity_version": claims.IdentityVersion, "policy_version": claims.PolicyVersion})
	}
	for _, name := range req.Names {
		name = strings.TrimSpace(name)
		if name == "" || len(name) > 128 {
			response.Fail(c, response.CodeParamError, "请求参数错误")
			return
		}
		claims, err := h.svc.ResolveByName(name)
		if err != nil || !claims.Eligible {
			identities = append(identities, gin.H{"name": name, "eligible": false})
			continue
		}
		identities = append(identities, gin.H{"user_id": claims.StableUserID, "eligible": true, "name": claims.Name, "groups": claims.Groups, "identity_version": claims.IdentityVersion, "policy_version": claims.PolicyVersion})
	}
	response.OK(c, gin.H{"identities": identities})
}
