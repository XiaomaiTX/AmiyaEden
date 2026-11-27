# AmiyaEden

[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-2.1.1-brightgreen.svg)](CHANGELOG.md)
[![CI](https://github.com/XiaomaiTX/AmiyaEden/actions/workflows/verify-ci.yaml/badge.svg)](https://github.com/XiaomaiTX/AmiyaEden/actions/workflows/verify-ci.yaml)

> 面向 EVE Online 联盟 / 军团的一体化运营平台。

EVE 联盟和军团的日常运营，往往散落在表格、聊天记录和游戏里：谁参加了哪次行动、损失该不该补、工资怎么发、新人谁在带。AmiyaEden 把这些收进一个网页系统——用 EVE 账号（SSO）登录、绑定自己的多个角色后，成员可以自助查询资产技能、提交补损和福利申请；指挥官可以组织舰队、记录出勤、审核入团；管理员可以发放薪资、运营内部经济、查看审计日志。

人物信息、钱包、技能、舰船、资产、合同等数据都直接来自 EVE 官方接口（ESI），平台负责组织和呈现，不要求人工誊录。从舰队行动、出勤积分，到补损、工资发放和内部货币结算，是一条有记录、可追溯的链路，每一步操作都会留下审计日志。

## 核心功能

| 领域 | 说明 |
| --- | --- |
| 登录与身份 | EVE SSO 登录、多人物绑定与主人物切换；注册需填写昵称和至少一种联系方式（QQ / Discord） |
| 舰队与出勤 | 舰队行动组织；PAP（出勤积分）发放与查询，支持联盟维度；舰队装配（EFT）；星系登记 |
| 补损与审批 | SRP（舰船损失补偿）申请与审核、福利申请、队长退税、工单系统；审批通过即在事务内发放并记账 |
| 招募与培养 | 入团申请与审核（凭证上传、邀请链接、联盟黑名单筛查）；新人引导、导师系统、徽章 |
| 个人信息 | 来自 ESI 的钱包 / 技能 / 舰船 / 植入体 / 资产 / 合同 / 装配查询；NPC 击杀与收入报告；技能规划（支持导入 EFT）与军团技能就绪度看板 |
| 经济与荣誉 | 伏羲币（内部积分货币）、商店与订单、统一运营薪资发放、伏羲名人堂、钱包收支分析 |
| 信息传达 | 站内公告与未读提醒 |
| 系统管理 | RBAC 职权与自动权限映射、军团维度的能力策略、军团建筑燃料看板、审计日志与导出、后台任务管理、Webhook |

各模块的具体行为、页面入口和权限边界见 [功能文档](docs/features/README.md)。

## 技术栈

- **后端**：Go + Gin + GORM，PostgreSQL + Redis，定时任务由 cron 驱动；鉴权为 EVE SSO + JWT。
- **前端（生产）**：Vue 3 + TypeScript + Vite + Pinia + Element Plus + Tailwind CSS（`static/`）。
- **前端（迁移期并行）**：React 19 + Vite + shadcn/ui（`static-react/`），与 Vue 并行开发和部署，暂不替换现有入口。
- **API 契约**：`/api/v1` 接口集中定义在 `contracts/` 的 OpenAPI 契约中，Go 服务端接口、Vue 与 React 客户端类型由同一份契约生成，前后端不会各写一套理解。
- **国际化**：界面文案中英文双语，两份语言包同步维护。

## 仓库结构

```text
AmiyaEden/
├── server/          # Go 后端（router → middleware → handler → service → repository → model 分层）
├── static/          # 生产前端（Vue 3）
├── static-react/    # 迁移期前端（React 19，与 Vue 并行）
├── contracts/       # OpenAPI 契约与三端代码生成
├── docs/            # 文档：指南（docs/guides）、架构、API、功能行为、工程标准
├── scripts/         # 辅助脚本
├── Makefile         # dev / lint / test / build / verify 统一入口
├── CHANGELOG.md     # 版本变更记录
└── docker-compose.example.yml  # 部署示例
```

## 快速开始

准备环境需要 Go ≥ 1.26（见 `server/go.mod`）、Node.js 24、pnpm ≥ 11、Air（Go 热重载）和 Docker（用来起 PostgreSQL / Redis）。

```bash
# 1. 准备后端配置（填写数据库、JWT 密钥、EVE SSO 凭据等）
cp server/config/config.example.yaml server/config/config.yaml

# 2. 启动依赖服务
docker compose -f docker-compose.example.yml up -d postgres redis

# 3. 安装前端依赖（React 迁移前端不使用可跳过）
cd static && pnpm install && cp .env.example .env.development && cd ..
cd static-react && pnpm install && cd ..

# 4. 一条命令同时启动后端（Air 热重载）和前端（Vite dev server）
make dev
```

后端默认监听 `8080`，Vue 前端开发地址默认是 <http://localhost:5173>。

要完整跑通登录，需要先在 [EVE Developers](https://developers.eveonline.com/) 申请一个 SSO 应用，把 `client_id`、`client_secret` 和回调地址填进 `server/config/config.yaml`。

首次初始化的全部配置项、常见问题和排查方式见 [本地开发指南](docs/guides/local-development.md)。日常校验用 `make lint`、`make test`、`make verify`，约定见 [测试与验证](docs/standards/testing-and-verification.md)。

## 部署

`docker-compose.example.yml` 是一份可直接参考的完整编排：后端、PostgreSQL、Redis，以及 Vue 和 React 两个前端容器；两个前端都通过 Nginx 把 `/api` 代理到后端。后端配置、日志和上传目录通过卷挂载持久化。

镜像发布在 Docker Hub（`xiaomaitx/amiya-eden-backend` / `-frontend` / `-frontend-react`）：`main` 分支推送 `latest` 和提交短 SHA 标签，`preview` 分支推送 `preview` 标签。从 2.0 以前的版本升级前请先备份数据库，并按 [升级清理指南](docs/guides/pre-2.0-cleanup.md) 处理残留数据。

## 文档

- [本地开发指南](docs/guides/local-development.md) —— 环境要求、配置项、常见问题
- [功能文档](docs/features/README.md) —— 各模块当前的行为与权限边界
- [API 约定与路由索引](docs/api/route-index.md) —— 契约在 `contracts/`，改接口先改契约
- [架构总览](docs/architecture/overview.md) 与 [模块导航](docs/architecture/module-map.md)
- [文档索引](docs/README.md)；工程约束（含 AI 编码代理规则）见 [AGENTS.md](AGENTS.md)
- [版本变更记录](CHANGELOG.md)

## 参与贡献

- 问题反馈和功能建议请提 [Issue](https://github.com/XiaomaiTX/AmiyaEden/issues/new/choose)。
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)（Commitlint + Commitizen 约束）；PR 写法见 [PR 信息规范](docs/standards/pr-message-standard.md)。
- 改接口时请连同 `contracts/`、双前端调用层和文档一起更新；界面文案需要中英文同步。

## 许可证

本项目以 [GPL-3.0](LICENSE) 开源。`static/` 衍生自上游 MIT 模板 [Art Design Pro](https://github.com/Daymychen/art-design-pro)。
