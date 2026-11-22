---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-23
source_of_truth:
  - server/internal/service/announcement.go
  - server/internal/repository/announcement.go
  - static/src/views/announcements/index.vue
  - static-react/src/pages/announcements-page.tsx
---

# 军团公告

全站共享，沿用 Login 成员权限。访客和仅有 recruiter 身份的账号不可见。admin、super_admin、senior_fc 可发布、编辑、隐藏和恢复所有公告。

两端 /announcements 是名人堂下方的一级入口。按发布时间倒序，每页 20 条，未读项强调显示。成员主动点击正文显示后提交阅读，按 user_id 记录，切换角色和设备保留。普通编辑保留已读，勾选提醒重新阅读时提升阅读版本。保存带 revision 乐观锁，陈旧编辑返回独立的冲突提示，管理员需取消后重新打开公告再编辑。阅读提交携带打开时的阅读版本；若期间公告已提升版本、被隐藏或不存在，接口返回明确错误而非静默成功，成员重新打开即可按最新版本已读。重复提交当前版本视为成功。

隐藏公告保留全部内容和图片，仅管理者可见。成员列表、详情、未读计数均排除隐藏公告。恢复后原阅读版本继续有效。

富文本由后端白名单清洗。支持 JPEG/PNG，每张输入最多 8 MiB、24M 像素，每篇最多 6 张；两端编辑器在插入前按全文已有图片数拦截，超限、格式或大小不符时给出本地化提示。图片移除元数据，按比例限制长边 2560，以 JPEG 85 质量保存，单张结果最多 4 MiB。图片字节单独存数据库，正文保留摘要引用，详情通过权限检查后以 data URI 返回。列表不含正文和图片。编辑移除的图片随事务清理，隐藏和重启不影响图片有效性。

后端错误按类别返回中文提示（标题/正文无效、图片无效、图片超限、公告不存在或已隐藏、无权管理、编辑冲突、阅读版本过期），React 页面直接展示该提示，Vue 由 HTTP 层弹出。

正文和图片在一致性事务内读取（SHARE 行锁阻止两次读取之间的并发保存）；普通编辑复用未改变的图片字节，避免重复有损压缩。两端 Nginx 对公告 API 单独允许 72 MiB 请求，后端同样限制请求大小，以容纳多张图片的 base64 编码。

announcement_unread 接入原 badge-counts；该计数查询失败只记录日志并省略此项，不影响其他徽标。登录、刷新、进入公告页及公告保存/已读成功后更新，无后台轮询。

