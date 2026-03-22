package model

// ─────────────────────────────────────────────
//  军团福利系统
// ─────────────────────────────────────────────

// ─── 发放模式 ───

const (
	WelfareDistModePerUser      = "per_user"      // 按自然人（User Account）发放
	WelfareDistModePerCharacter = "per_character"  // 按人物（EVE Character）发放
)

// ─── 状态 ───

const (
	WelfareStatusActive   int8 = 1 // 启用
	WelfareStatusDisabled int8 = 0 // 停用
)

// ─── 数据模型 ───

// Welfare 福利定义
type Welfare struct {
	BaseModel
	Name             string `gorm:"size:256;not null"           json:"name"`
	Description      string `gorm:"type:text"                   json:"description"`
	DistMode         string `gorm:"size:20;not null;default:'per_user'" json:"dist_mode"`
	RequireSkillPlan bool   `gorm:"default:false"               json:"require_skill_plan"`
	SkillPlanID      *uint  `gorm:"index"                       json:"skill_plan_id"`
	Status           int8   `gorm:"default:1"                   json:"status"`
	CreatedBy        uint   `gorm:"not null"                    json:"created_by"`
}

func (Welfare) TableName() string { return "welfare" }

// WelfareDistribution 福利发放记录
type WelfareDistribution struct {
	BaseModel
	WelfareID     uint   `gorm:"not null;index"            json:"welfare_id"`
	UserID        uint   `gorm:"not null;index"            json:"user_id"`
	CharacterID   int64  `gorm:"not null"                  json:"character_id"`
	CharacterName string `gorm:"size:128"                  json:"character_name"`
	QQ            string `gorm:"size:20"                   json:"qq"`
	DiscordID     string `gorm:"size:20"                   json:"discord_id"`
	DistributedBy uint   `gorm:"not null"                  json:"distributed_by"`
}

func (WelfareDistribution) TableName() string { return "welfare_distribution" }
