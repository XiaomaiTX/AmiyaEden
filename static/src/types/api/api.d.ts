/**
 * API 接口类型定义模块
 *
 * 保留前端通用类型、状态和契约扩展；后端接口类型从 `@/api/generated/client` 导入。
 *
 * ## 主要功能
 *
 * - 通用类型（分页参数、响应结构等）
 * - 认证类型（登录、用户信息等）
 * - 系统管理类型（用户、职权等）
 * - 全局命名空间声明
 *
 * ## 使用场景
 *
 * - API 请求参数类型约束
 * - API 响应数据类型定义
 * - 接口文档类型同步
 *
 * ## 注意事项
 *
 * - 在 .vue 文件使用需要在 eslint.config.mjs 中配置 globals: { Api: 'readonly' }
 * - 使用全局命名空间，无需导入即可使用
 *
 * ## 使用方式
 *
 * ```typescript
 * const params: Api.Auth.LoginParams = { userName: 'admin', password: '123456' }
 * const response: Api.Auth.UserInfo = await fetchUserInfo()
 * ```
 *
 * @module types/api/api
 * @author Art Design Pro Team
 */

declare namespace Api {
  /** 通用类型 */
  namespace Common {
    /** 分页参数 */
    interface PaginationParams {
      /** 当前页码 */
      current: number
      /** 每页条数 */
      size: number
      /** 总条数 */
      total: number
    }

    /** 通用搜索参数 */
    type CommonSearchParams = Pick<PaginationParams, 'current' | 'size'>

    /** 分页响应基础结构 */
    interface PaginatedResponse<T = any> {
      list: T[]
      total: number
      page: number
      pageSize: number
    }

    /** 启用状态 */
    type EnableStatus = '1' | '2'
  }

  /** 认证类型 */
  namespace Auth {
    /** 登录参数（已废弃，仅保留兼容） */
    interface LoginParams {
      userName: string
      password: string
    }

    /** 登录响应（已废弃，仅保留兼容） */
    interface LoginResponse {
      token: string
      refreshToken: string
    }

    /** EVE 人物信息 */
    type EveCharacter = Pick<
      import('@/api/generated/client').EveCharacter,
      | 'id'
      | 'character_id'
      | 'character_name'
      | 'user_id'
      | 'scopes'
      | 'token_expiry'
      | 'token_invalid'
      | 'corporation_id'
      | 'alliance_id'
    > & { fuxi_legion_tenure_days?: number | null }

    /** 用户信息（路由守卫和权限指令使用） */
    interface UserInfo {
      roles: string[]
      primaryCorporationId?: number
      corpCapabilities?: string[]
      corpRules?: Record<string, string | number | boolean>
      userId: number
      userName: string
      nickname: string
      qq: string
      discordId: string
      profileComplete: boolean
      enforceCharacterESIRestriction: boolean
      isCurrentlyNewbro?: boolean
      isMentorMenteeEligible?: boolean
      characters?: EveCharacter[]
      primaryCharacterId?: number
    }
  }

  /** 任务管理器类型 */
  namespace TaskManager {
    type TaskLastExecution = NonNullable<import('@/api/generated/client').ManagedTaskLastExecution>
  }

  /** 军团技能计划类型 */
  namespace SkillPlan {
    // The shared editor sends sort_order for both calls; the server only applies it on update.
    type CreateSkillPlanParams = import('@/api/generated/client').SkillPlanCreateInput &
      Pick<import('@/api/generated/client').SkillPlanUpdateInput, 'sort_order'>
  }

  /** 军团福利系统类型 */
  namespace Welfare {
    type WelfareEligibilityReason = NonNullable<
      import('@/api/generated/client').WelfareReviewResult['eligibility_reason']
    >
  }

  /** 工作台类型 */
  namespace Dashboard {
    type GalaxyRegistrySystemStatus =
      import('@/api/generated/client').GalaxyRegistrySystemItem['status']
    type GalaxyRegistryEntryStatus = import('@/api/generated/client').GalaxyRegistryEntry['status']
    type GalaxyRegistryValidationStatus =
      import('@/api/generated/client').GalaxyRegistryEntry['validation_status']
    type GalaxyRegistrySystemActiveEntry = NonNullable<
      import('@/api/generated/client').GalaxyRegistrySystemItem['active_entry']
    >
    type GalaxyRegistryEntryListParams = import('@/api/generated/client').ListAdminEntriesParams & {
      status?: GalaxyRegistryEntryStatus | ''
      validation_status?: GalaxyRegistryValidationStatus | ''
    }
  }

  /** 导师帮扶 */
  namespace Mentor {
    type MentorRelationshipStatus =
      import('@/api/generated/client').MentorRelationshipView['status']
    type MenteeStatusFilter = MentorRelationshipStatus | 'all'
    type RewardConditionType =
      import('@/api/generated/client').MentorRewardStageInput['condition_type']
  }

  namespace Ticket {
    type TicketItem = import('@/api/generated/client').TicketRecord &
      Partial<import('@/api/generated/client').TicketAdminItem>
    type UpsertCategoryParams = import('@/api/generated/client').TicketCategoryInput & {
      name: string
      name_en: string
    }
  }

  namespace ToolBookmark {
    type FormFields<T> = { [K in keyof T]: NonNullable<T[K]> }
    type UpsertParams = FormFields<import('@/api/generated/client').ToolBookmarkCreateInput>
  }

  /** 联盟黑名单：人物、QQ、Discord 三组相互独立的核查值 */
  namespace Blacklist {
    type ListParams = Common.CommonSearchParams &
      import('@/api/generated/client').ListBlacklistParams
  }

  /** 系统配置 */
  namespace SysConfig {
    type CorporationCapability =
      | 'srp.user'
      | 'srp.manage'
      | 'welfare.user'
      | 'welfare.approval'
      | 'welfare.settings'
      | 'menu.srp'
      | 'menu.welfare'
      | 'menu.dashboard'
      | 'menu.operation'
      | 'menu.role'
      | 'menu.newbro'
      | 'menu.fuxi_hall'
      | 'menu.ticket'
      | 'menu.shop'
      | 'menu.system'
      | 'menu.info'
      | 'menu.skill_planning'
      | 'ticket.manage'
      | 'shop.manage'
      | 'system.manage'
      | 'info.wallet.read'
      | 'info.npc_kills.self'
      | 'info.npc_kills.corp'
      | 'info.skills.read'
      | 'info.assets.read'
      | 'info.contracts.read'
      | 'info.fittings.manage'
      | 'shop.wallet.read'
      | 'wallet.user.enabled'
      | 'shop.order.create'
      | 'shop.order.read_self'
      | 'dashboard.npc_kills.corp'
      | 'dashboard.corp_structures.read'
      | 'dashboard.corp_structures.manage'
      | 'operation.fleet.read_self'
      | 'operation.fleet.manage'
      | 'operation.fleet.pap.manage'
      | 'skill_planning.corp.read'
      | 'skill_planning.corp.manage'
      | 'skill_planning.personal.read'
      | 'skill_planning.personal.manage'
      | 'newbro.user.actions'
      | 'newbro.admin.read'
      | 'newbro.admin.manage'
      | 'mentor.user.actions'
      | 'mentor.mentor.actions'
      | 'mentor.admin.manage'
      | 'system.task.read'
      | 'system.task.run'
      | 'system.basic_config.read'
      | 'system.basic_config.manage'
      | 'system.wallet.read'
      | 'system.wallet.adjust'
      | 'system.audit.read'
      | 'system.audit.export'
      | 'system.tool_bookmark.read'
      | 'system.tool_bookmark.manage'
      | 'ticket.user.create'
      | 'ticket.user.reply'
      | 'ticket.admin.read'
      | 'ticket.admin.manage'
      | 'shop.admin.product.manage'
      | 'shop.admin.order.manage'
      | 'fuxi_hall.public.read'
      | 'fuxi_hall.admin.manage'
    // Local status panels may not initialize the error details before fetching.
    type SDEQueryErrorFields =
      | 'last_query_error'
      | 'last_query_error_at'
      | 'last_query_error_source'
    type SDEStatus = Omit<import('@/api/generated/client').SystemSDEStatus, SDEQueryErrorFields> &
      Partial<Pick<import('@/api/generated/client').SystemSDEStatus, SDEQueryErrorFields>>
  }

  namespace FuxiHall {
    type Contributions = Omit<import('@/api/generated/client').HallContributions, 'corp_pap'> &
      Partial<Pick<import('@/api/generated/client').HallContributions, 'corp_pap'>>
    type PresentationFields =
      | 'qq'
      | 'can_edit'
      | 'contributions_updated_at'
      | 'tenure_months'
      | 'avatar_image'
    type Card = Omit<
      import('@/api/generated/client').HallCard,
      PresentationFields | 'contributions'
    > &
      Partial<Pick<import('@/api/generated/client').HallCard, PresentationFields>> & {
        contributions?: Contributions
        contribution_offsets?: Contributions
      }
    type PublicPageResponse = Omit<import('@/api/generated/client').HallPublic, 'cards'> & {
      cards: Card[]
    }
  }
}
