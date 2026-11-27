package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/middleware"
	"amiya-eden/internal/model"
	"amiya-eden/jobs"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

// newOpenAPIMiddlewareResolver translates the checked OpenAPI policy into the
// existing Gin authorization chain. The registrar installs this chain before
// decoding requests. Unknown policies fail startup rather than granting access.
func newOpenAPIMiddlewareResolver() func(string, []string) []gin.HandlerFunc {
	jwt := middleware.JWTAuth()
	login := middleware.RequireLoginUser()
	return func(operation string, policy []string) []gin.HandlerFunc {
		chain := make([]gin.HandlerFunc, 0, len(policy))
		for i, step := range policy {
			if (step == "jwt") != (i == 0) {
				panic(fmt.Sprintf("OpenAPI operation %s: authorization chain must start with jwt exactly once", operation))
			}
			switch {
			case step == "jwt":
				chain = append(chain, jwt)
			case step == "login":
				chain = append(chain, login)
			case strings.HasPrefix(step, "role:"):
				roles := strings.Split(strings.TrimPrefix(step, "role:"), ",")
				for _, role := range roles {
					if !model.IsValidRoleCode(role) {
						panic(fmt.Sprintf("OpenAPI operation %s: unknown role %q", operation, role))
					}
				}
				chain = append(chain, middleware.RequireRole(roles...))
			case strings.HasPrefix(step, "cap:"):
				capability := strings.TrimPrefix(step, "cap:")
				if !model.IsEnforcedCorpCapability(capability) {
					panic(fmt.Sprintf("OpenAPI operation %s: unknown or unenforced capability %q", operation, capability))
				}
				chain = append(chain, middleware.RequireCorpCapability(capability))
			default:
				panic(fmt.Sprintf("OpenAPI operation %s: unknown authorization step %q", operation, step))
			}
		}
		return chain
	}
}

// handleOpenAPIError keeps transport decoding and encoding errors in the API
// envelope without exposing implementation details from the generated binder.
func handleOpenAPIError(c *gin.Context, operation string, err error, status int) {
	c.Abort()
	if status >= http.StatusInternalServerError {
		_ = c.Error(fmt.Errorf("OpenAPI operation %s: %w", operation, err))
		if !c.Writer.Written() {
			response.Fail(c, response.CodeBizError, "服务器内部错误，请稍后重试")
		}
		return
	}
	if c.Writer.Written() {
		return
	}
	if strings.Contains(operation, "QQOfficial") || operation == "sendFleetQQPing" || operation == "listFleetQQPingResults" {
		c.JSON(http.StatusBadRequest, apicontract.QQOfficialError{Code: 400, Msg: "invalid_input"})
		return
	}
	code, message := response.CodeParamError, "请求参数错误"
	var parameter *apicontract.GinParameterError
	_ = errors.As(err, &parameter)
	pathFailure := parameter != nil && parameter.Location == "path"
	switch operation {
	case "listEsiRefreshStatuses":
		if parameter != nil && parameter.Location == "query" && (parameter.Parameter == "current" || parameter.Parameter == "size") {
			// The original route returned an empty page before parsing pagination when the queue was unavailable.
			if jobs.GetESIQueue() == nil {
				response.OKWithPage(c, []apicontract.EsiRefreshStatus{}, 0, 1, 20)
				return
			}
			message = fmt.Sprintf("invalid %s query parameter: expected integer", parameter.Parameter)
		}
	case "adminListTickets", "listMyTickets", "listAdminEntries", "listMyEntries":
		if parameter != nil && parameter.Location == "query" {
			switch parameter.Parameter {
			case "current", "size":
				message = fmt.Sprintf("invalid %s query parameter: expected integer", parameter.Parameter)
			case "user_id", "category_id", "system_config_id":
				message = "无效的 " + parameter.Parameter
			}
		}
	case "getEveRegistration", "completeEveRegistration":
		c.Header("Cache-Control", "no-store")
		if strings.TrimSpace(c.GetHeader("X-Registration-Token")) == "" {
			message = "注册会话已失效，请重新登录"
		}
	case "eveSsoCallback":
		c.Header("Cache-Control", "no-store")
		c.Header("Referrer-Policy", "no-referrer")
	case "setPrimaryEveCharacter", "unbindEveCharacter":
		message = "无效的人物 ID"
	case "saveFitting":
		message = "装配参数无效"
	case "setWebhookConfig", "testWebhook":
		message = "请求参数错误: " + err.Error()
	case "updateSalarySettings":
		message = "请填写完整、有效的工资配置"
	case "querySkillReadiness", "getSkillReadinessUserCharacters", "getSkillReadinessCharacterSkills":
		message = "查询参数无效"
	case "createAdmission", "editAdmission", "resubmitAdmission":
		if pathFailure {
			message = "无效的application"
		} else {
			code, message = response.CodeBizError, "admissions.errors.evidence"
		}
	case "inviteAdmission", "rejectAdmission":
		if pathFailure {
			message = "无效的application"
		} else {
			code, message = response.CodeBizError, "admissions.errors.invalid"
		}
	case "getAdmission", "getAdmissionEvidence":
		if pathFailure {
			message = "无效的application"
		}
	case "createAnnouncement", "updateAnnouncement":
		if pathFailure {
			message = "无效的announcement"
		} else {
			message = "公告标题或正文无效，请检查长度后重试"
		}
	case "getAnnouncement", "readAnnouncement":
		if pathFailure {
			message = "无效的announcement"
		} else {
			message = "参数错误"
		}
	case "applyForMentor", "acceptMentorApplication", "rejectMentorApplication", "adminRevokeMentorRelationship", "adminUpdateMentorRewardStages",
		"submitRecruitQQ", "checkDirectReferrer", "confirmDirectReferrer", "previewCaptainRefund", "createCaptainRefund":
		message = "invalid request"
	case "rejectCaptainRefund":
		if pathFailure {
			message = "invalid id"
		} else {
			message = "invalid request"
		}
	case "getMyCaptainRefund", "getAdminCaptainRefund", "approveCaptainRefund":
		if pathFailure {
			message = "invalid id"
		}
	case "updateCaptainRefundSettings":
		message = "invalid settings"
	case "listMyCaptainRefunds", "listAdminCaptainRefunds":
		message = "invalid pagination"
	case "updateHallOwnCard":
		if pathFailure {
			message = "无效的card"
		} else {
			message = "fuxiHall.errors.invalid"
		}
	case "updateHallCard", "deleteHallCard":
		if pathFailure {
			message = "无效的卡片 ID"
		}
	}
	response.Fail(c, code, message)
}
