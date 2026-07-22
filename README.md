# AmiyaEden

[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.14.0-brightgreen.svg)](CHANGELOG.md)
[![CI](https://github.com/XiaomaiTX/AmiyaEden/actions/workflows/verify-ci.yaml/badge.svg)](https://github.com/XiaomaiTX/AmiyaEden/actions/workflows/verify-ci.yaml)

> 面向 EVE Online 联盟 / 军团的一体化运营平台。

## 简介

AmiyaEden 把联盟 / 军团的日常运营收敛到一个平台：用 EVE SSO 登录、绑定多人物后，在一个系统里完成舰队行动组织、参与度（PAP）统计、补损（SRP）发放、技能规划、新人培养、内部经济（伏羲币 / 商店）与系统管理。

## 理念

- **以 ESI / SSO 为数据底座**：人物信息、钱包、技能、舰船、资产、合同直接来自 EVE 官方接口，平台只做组织与呈现。
- **按军团 / 职权分层授权**：在 RBAC 职权之上叠加军团能力策略（capability）层，支持同职权在不同军团的能力差异化访问，权限一律后端强制鉴权。
- **运营闭环**：从舰队行动 → PAP → 补损 / 福利审批 → 伏羲币结算，串成一条可追溯的链路，并通过统一审计日志留痕。
- **文档与代码同源**：仓库维护一套带信任顺序的规范化文档树，避免文档与实现脱节。

## 核心功能

| 领域 | 能力 |
| --- | --- |
| 登录与身份 | EVE SSO 登录、多人物绑定与主人物切换 |
| 运营与行动 | 舰队行动、PAP 发放与查询、联盟 PAP、舰队配置（装配 / EFT）、星系登记 |
| 个人与军团信息 | ESI 钱包 / 技能 / 舰船 / 植入体 / 资产 / 合同 / 装配查询、NPC 击杀与收入报告、ESI 授权检查 |
| 申请与审批 | SRP 补损、福利申请、工单系统 |
| 经济与商店 | 伏羲币、伏羲大厅、商店与订单、统一钱包分析 |
| 成员与培养 | 新人引导、导师系统、徽章 |
| 系统管理 | RBAC + 自动职权映射、军团能力策略、军团建筑（燃料 / 计时）、审计日志与导出、任务管理、Webhook、通知 |

> 各模块的当前行为、入口与权限边界见 [docs/features/](docs/features/README.md)。

## 技术栈

- **后端**：Go + Gin + GORM + PostgreSQL + Redis（定时任务由 cron 驱动，鉴权为 EVE SSO + JWT）
- **前端（生产）**：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Element Plus + Tailwind CSS
- **前端（迁移期并行）**：`static-react/` —— 基于 React 19 + Vite + shadcn/ui 的重写，使用独立校验任务、镜像和 Compose 服务，不替换当前 Vue 入口

## 仓库结构

```text
AmiyaEden/
├── server/                 # Go 后端（handler / service / repository / model 分层）
│   ├── bootstrap/          # 配置 / 日志 / DB / Redis / 路由 / Cron 初始化
│   ├── internal/           # handler / middleware / model / repository / router / service
│   ├── jobs/               # 定时任务
│   └── pkg/                # JWT、EVE SSO / ESI、响应工具等
├── static/                 # 生产前端（Vue 3）
├── static-react/           # 进行中的 React 19 重写
├── docs/                   # 规范化文档树（standards / architecture / api / features / specs / guides）
├── scripts/                # 辅助脚本
├── CHANGELOG.md            # 版本变更记录
├── AGENTS.md               # 工程约束入口（委托给 docs/ai/repo-rules.md）
└── docker-compose.example.yml
```

## 快速开始

```bash
cp server/config/config.example.yaml server/config/config.yaml
docker compose -f docker-compose.example.yml up -d postgres redis
cd static && pnpm install && cd ..
make dev      # 同时启动后端（Air 热重载）与前端（Vite dev server）
```

依赖要求、首次初始化、环境变量与排错见 [docs/guides/local-development.md](docs/guides/local-development.md)。
校验与测试命令见 [docs/standards/testing-and-verification.md](docs/standards/testing-and-verification.md)。

## 文档入口

- [文档索引与信任顺序](docs/README.md)
- [工程约束（最高优先级）](AGENTS.md)
- [架构总览](docs/architecture/overview.md)
- [模块导航](docs/architecture/module-map.md)
- [功能状态](docs/features/README.md)
- [API 路由索引](docs/api/route-index.md)
- [版本变更记录](CHANGELOG.md)

## 项目状态

- **License**：[GPL-3.0](LICENSE)。`static/` 衍生自上游 MIT 模板 [Art Design Pro](https://github.com/Daymychen/art-design-pro)，其自身许可证为 MIT。
- **前端实现状态**：`static/`（Vue 3）仍是当前生产实现；`static-react/` 在迁移期通过独立的 `amiya-eden-frontend-react` 镜像与 Vue 并行运行，Vue 保持端口 `80`，React 使用端口 `3000`。
- **部署**：`docker-compose.example.yml` 同时定义 Vue `frontend` 与 React `frontend-react`；`main` 分支发布 `latest`/短 SHA 镜像，`preview` 分支发布 `preview` 镜像。
- **反馈**：通过 [Issue](https://github.com/XiaomaiTX/AmiyaEden/issues/new/choose) 报告问题或提议功能。
- **提交规范**：使用 [Conventional Commits](https://www.conventionalcommits.org/)（由 Commitlint + Commitizen 约束）；PR 信息规范见 [docs/standards/pr-message-standard.md](docs/standards/pr-message-standard.md)。

## 许可证

详见 [LICENSE](LICENSE)。
