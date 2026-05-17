package middleware

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/jwt"
	"amiya-eden/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxKeyUserID               = "userID"
	ctxKeyCharacterID          = "characterID"
	ctxKeyRoles                = "roles"
	ctxKeyPrimaryCorporationID = "primaryCorporationID"
	ctxKeyCorpCapabilities     = "corpCapabilities"
	ctxKeyCorpRules            = "corpRules"
	ctxKeyCorpFullAccess       = "corpFullAccess"
)

// JWTAuth JWT 认证中间件
// 解析 Token，加载用户职权与权限到 context
func JWTAuth() gin.HandlerFunc {
	roleSvc := service.NewRoleService()
	corpPolicySvc := service.NewCorporationPolicyService()

	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Fail(c, response.CodeUnauthorized, "未提供认证令牌")
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(token)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized, "令牌无效或已过期")
			c.Abort()
			return
		}

		c.Set(ctxKeyUserID, claims.UserID)
		c.Set(ctxKeyCharacterID, claims.CharacterID)

		// 加载用户职权（带 Redis 缓存）
		roles, err := roleSvc.GetUserRoleNames(c.Request.Context(), claims.UserID)
		if err != nil {
			roles = []string{model.RoleGuest}
		}
		c.Set(ctxKeyRoles, roles)
		corpCtx := corpPolicySvc.BuildUserPolicyContext(claims.UserID)
		c.Set(ctxKeyPrimaryCorporationID, corpCtx.PrimaryCorporationID)
		c.Set(ctxKeyCorpCapabilities, corpCtx.Capabilities)
		c.Set(ctxKeyCorpRules, corpCtx.Rules)
		c.Set(ctxKeyCorpFullAccess, corpCtx.FullAccess)

		c.Next()
	}
}

// RequireRole 要求用户拥有指定职权之一（super_admin 自动通过）
func RequireRole(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := GetUserRoles(c)
		if model.IsSuperAdmin(roles) {
			c.Next()
			return
		}
		for _, code := range codes {
			if model.HasAnyRoleMatch(roles, code) {
				c.Next()
				return
			}
		}
		response.Fail(c, response.CodeForbidden, "权限不足，需要职权: "+strings.Join(codes, "/"))
		c.Abort()
	}
}

// RequireLoginUser 要求请求方是已认证且非 guest 的产品用户
func RequireLoginUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if model.HasNonGuestRole(GetUserRoles(c)) {
			c.Next()
			return
		}
		response.Fail(c, response.CodeForbidden, "权限不足，需要登录用户")
		c.Abort()
	}
}

// RequireCorpCapability 要求请求方拥有指定军团能力（super_admin 自动通过）
func RequireCorpCapability(capability string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := GetUserRoles(c)
		if model.IsSuperAdmin(roles) {
			c.Next()
			return
		}
		if has := service.EvaluateCapability(service.UserCorpPolicyContext{
			Capabilities: GetCorpCapabilities(c),
			Rules:        GetCorpRules(c),
			FullAccess:   IsCorpFullAccess(c),
		}, capability); has {
			c.Next()
			return
		}
		response.Fail(c, response.CodeForbidden, "权限不足")
		c.Abort()
	}
}

// ─── Context 辅助函数 ───

func GetUserID(c *gin.Context) uint {
	return c.GetUint(ctxKeyUserID)
}

func GetCharacterID(c *gin.Context) int64 {
	v, _ := c.Get(ctxKeyCharacterID)
	if id, ok := v.(int64); ok {
		return id
	}
	return 0
}

func GetUserRoles(c *gin.Context) []string {
	v, exists := c.Get(ctxKeyRoles)
	if !exists {
		return nil
	}
	roles, _ := v.([]string)
	return roles
}

func GetPrimaryCorporationID(c *gin.Context) int64 {
	v, _ := c.Get(ctxKeyPrimaryCorporationID)
	if id, ok := v.(int64); ok {
		return id
	}
	return 0
}

func GetCorpCapabilities(c *gin.Context) []string {
	v, exists := c.Get(ctxKeyCorpCapabilities)
	if !exists {
		return []string{}
	}
	if capabilities, ok := v.([]string); ok && capabilities != nil {
		return capabilities
	}
	return []string{}
}

func GetCorpRules(c *gin.Context) map[string]any {
	v, exists := c.Get(ctxKeyCorpRules)
	if !exists {
		return map[string]any{}
	}
	if rules, ok := v.(map[string]any); ok && rules != nil {
		return rules
	}
	return map[string]any{}
}

func IsCorpFullAccess(c *gin.Context) bool {
	v, _ := c.Get(ctxKeyCorpFullAccess)
	if allowed, ok := v.(bool); ok {
		return allowed
	}
	return false
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return c.Query("token")
}
