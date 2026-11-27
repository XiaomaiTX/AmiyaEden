# AmiyaEden Vue 前端统一风格审计

2026-11-26 · 审计基线 `48f895e9` · 源码静态审计 · 仅交付报告，不修改 Vue、CSS、路由、配置或后端。

## 结论

**在 71 个常规页面中，41 个有明确不一致（57.7%），另有 11 个只列建议或待实屏复验。** 在 96 个常规共享/局部组件中，**22 个有明确不一致，10 个列建议或待复验**。最值得先处理的不是配色：是筛选后假空、记录被裁剪、同类账本的分页规格不同，以及鼠标能点但键盘不能操作。**军团公告和名人堂不计入以上数字，分别详见第五节。**

| 统计单位 | A 明确不一致 | B 建议/待复验 | C 未见独立问题 | E 保留专用形态 | 合计 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 常规独立页面 | 41 | 11 | 8 | 11 | 71 |
| 常规共享组件 | 9 | 3 | 47（C/E 合计） | — | 59 |
| 常规页面局部组件 | 13 | 7 | 17（C/E 合计） | — | 37 |

页面和组件是两个计数口径，**不能把 41 与 22 相加当作缺陷总数**。例如两个队长补贴入口都受同一个共享空态缺陷影响，按两页但只计一个组件；全局表头缺陷不机械传导给所有调用页。

## 一、审计范围和判定口径

- **页面：**按 [Vue 动态模块](<static/src/router/modules/index.ts#L22-L39>)和 [静态路由](<static/src/router/routes/staticRoutes.ts#L13-L67>)展开，共 **81 条有页面组件的叶记录、73 个不同的页面源文件**。入团六条路由共用一个页面（不重复加五次）；三个异常页各有静态和动态声明（不重复加三次，且不假设两组同时可达）。排除军团公告、名人堂两个指定入口后，常规统计的分母为 **71 个页面、79 条叶记录**。菜单目录、外壳、路由重定向不算页面；角色限制和隐藏菜单不等同于未挂载。
- **组件：**按 Git 跟踪文件清点 **208 个 Vue SFC**。用 Vue SFC 解析器、路由、显式/异步引用、模板自动导入和[全局组件开关](<static/src/config/modules/component.ts#L23-L79>)核对：**169** 个属于常规面＝71 个页面＋**96** 个组件（59 个共享＋37 个页面局部）＋2 个应用外壳；**7** 个只服务两项单列模块＝2 个页面＋4 个局部组件＋1 个共享富文本组件；**32** 个未发现可达入口或当前开关关闭＝25 个共享组件＋7 个视图文件。静态引用不保证某个角色、某份数据下必定渲染；不把自动生成的组件声明当作挂载证据。
- **A：明确不一致。**有成文规则，或能从模板、CSS、绑定与调用链证明数据、控件或反馈不一致。**B：建议统一或待实屏复验。**偏离同类多数模式、仅偏离推荐项，或需特定视口/数据确认。**C：未见所审维度的独立问题。E：有理由的专用布局，未见需并入常规统计的独立问题。**每个页面和组件分别按最高等级去重；独立写在子组件的原因登记一次，归属页面可标受影响；全局组件的问题不机械记到所有引用页面。A 不代表已在浏览器复现；C/E 也不是像素验收通过。
- **不把页面一律改成表格。**用同类用途横向对照：列表与列表比、账本与账本比；目录卡、选择器、报表、表单、详情子表、公开入口和系统外壳不能硬套后台 CRUD。色值仅在绕开主题或造成状态表达不一致时登记，不因图表系列色、业务状态色、品牌用色而判违规。接口、安全和与风格无关的业务逻辑不做全量审查。

### 采用的基线

| 场景 | 基线与对照 |
| --- | --- |
| 分页表格 | [表格页规范：共享表格/账本](<docs/standards/frontend-table-pages.md#L15-L36>)：Vue 用 `ArtTable`＋`useTable` 或实际等价封装；日志、流水、历史等无界数据用 `ledger`，首次请求 200，不重复声明预设。当前 [Vue 账本预设](<static/src/components/core/tables/art-table/presentation.ts#L1-L2>)的页大小选项为 200/500/1000，不等于所有表格都固定 200。 |
| 页面骨架 | [表格页规范：布局与溢出](<docs/standards/frontend-table-pages.md#L48-L81>)推荐筛选在卡片外、`ElCard.art-table-card`＋`ArtTableHeader`、对话框和卡片平级。自然增长交给[应用外壳](<static/src/views/index/style.scss#L11-L43>)；全高型必须沿 Tab、Card body 到内容补全高度链。现有正向对照：[合同账本](<static/src/views/info/contracts/index.vue#L1-L53>)、[资产卡片的显式滚动](<static/src/views/info/assets/index.vue#L1-L76>)。 |
| 卡片记录和详情 | [记录卡片页规范](<docs/standards/frontend-record-card-pages.md#L17-L43>)：自然增长或显式内部滚动；仪表盘固定摘要、对话框滚动、有界摘要可以不同。[选导师](<static/src/views/newbro/select-mentor/index.vue#L1-L25>)和[界面盒样式](<static/src/assets/styles/core/app.scss#L84-L159>)可作为对照。 |
| 操作与状态 | [主题、行内复制与排序规范](<docs/standards/frontend-table-pages.md#L76-L109>)；以 `ElButton`／[ArtCopyButton](<static/src/components/core/forms/art-copy-button/index.vue#L1-L12>)、`ElEmpty`、主题 token 为基线。用户文案按[仓库规则](<docs/ai/repo-rules.md#L81-L83>)双语；明显的时间戳外观差异参考[时间戳标准](<docs/standards/timestamp-formatting.md#L13-L30>)。 |

多数使用事实（只数常规 71 个页面；模板 AST＋显式页面子组件引用，按页面去重）：**62** 页有 `ElCard`，**45** 页有 `ArtTable` 或 `ElTable`，其中 **35** 页有 `ArtTable`（可与 `ElTable` 并存）；**30** 页有 `ArtTableHeader`，只有 **2** 页通过局部组件使用 `ArtSearchBar`，**21** 页有 `ElTabs`。这是组件采用率，**不是**“没有使用就不合规”的规则。例如筛选的可见位置、重置行为比是否导入 `ArtSearchBar` 更重要。

## 二、跨页面根因与优先级

1. **P1｜内容不可达。**共享 [`.art-table-card` 与 `.art-full-height`](<static/src/assets/styles/core/app.scss#L178-L201>)默认组合存在 `height: 100%`／`overflow: hidden`；当页内再套定高 Tab、卡片或列表时，必须看谁最终滚动。优先复查商城商品页、商品/订单多 Tab、导师学员、小屏复杂表单和双列详情。不要简单给所有根节点加 `art-full-height` 或所有层加 `overflow: auto`，这会制造嵌套滚动。
2. **P1｜筛选后假空态。**[useTable 请求合并](<static/src/hooks/core/useTable.ts#L297-L306>)默认采用 `pagination.current`；[getData/getDataByPage](<static/src/hooks/core/useTable.ts#L414-L437>)分别保页/回首页。页面只改 `searchParams.current` 或筛选后直接 `getData()`，可从第 2 页以上查询新集合的旧页；再遇 [ArtTable 空行隐藏分页](<static/src/components/core/tables/art-table/index.vue#L262-L264>)，会让有数据的集合看似无结果。搜索/切换筛选和显式“刷新当前页”不能用同一个语义。
3. **P1｜账本分成多套外观。**同为历史、流水或审批的页面出现原生 `ElTable`＋手写 `ElPagination`、共享表格非 ledger、仅加 ledger 样式却以 20/50/100 为初始请求等差异。应先确认记录是否可长期增长，再统一默认请求、分页器以及空态。保留图表、详情子表及业务预览。
4. **P2｜工具条、反馈、主题和断点分叉。**桌面表格工具条、页头/菜单触发器和部分可展开控件使用鼠标专用 `div`；同类页面混用按钮操作、空态/加载和固定色。弹窗方面，[全局规则](<static/src/assets/styles/core/el-ui.scss#L211-L216>)给 ≤640px 兜底，但在 641px 与弹窗固定宽度之间没有对应通用上限。宜先统一共享控件与浮层契约，再按场景调整页面；不是先逐页换颜色。

## 三、逐页审计矩阵（常规 71 页）

下列每行代表一个源页面，A/B/C/E 按最高级别登记，非缺陷行也列入分母。A 行若同时有较低优先级观察，不再加一页。表内证据是用于定位的实现入口，不是逐页修改授权。

### 仪表盘、角色凭据（8 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| C | `/characters` · [角色管理](<static/src/views/dashboard/characters/index.vue#L1-L75>) | 个人资料、推荐绑定、角色卡允许是表单与卡片组合；操作、警告、空态使用主题基元。 |
| A | `/mumble` · [凭据页](<static/src/views/dashboard/mumble-credentials/index.vue#L1-L24>) | 一次性口令复制的反馈和实现未复用共享 hook，根因在凭据卡；专用凭据表单保留。 |
| E | `/dashboard/console` · [工作台](<static/src/views/dashboard/console/index.vue#L2-L29>) | 响应式 KPI、舰队、两块 PAP 和 SRP 摘要，不是历史账本；原生/无分页子表不因此判错。 |
| E | `/dashboard/npc-kills` · [军团刷怪汇总](<static/src/views/dashboard/npc-kills/index.vue#L106-L145>) | 日期区间分析与有界报表子表，区别于 `/info/npc-kills` 的逐条流水。 |
| A | `/dashboard/corporation-structures` · [军团建筑](<static/src/views/dashboard/corporation-structures/index.vue#L618-L632>) | 主清单用共享分页；燃料官分配清单另行[切片与维护页码](<static/src/views/dashboard/corporation-structures/index.vue#L849-L854>)。应保留分配的跨页草稿和完整候选数据，仅统一呈现分页；服务配置的原生子表可保留。 |
| A | `/dashboard/skill-readiness` · [技能达标](<static/src/views/dashboard/skill-readiness/index.vue#L73-L106>) | 展开用户行内嵌角色表可保留，不能仅因用原生表判问题；但[页级分页](<static/src/views/dashboard/skill-readiness/index.vue#L185-L197>)和尺寸在本页另实现，未复用共享分页层，[秒级时间戳](<static/src/views/dashboard/skill-readiness/index.vue#L575-L580>)也用了本地格式。 |
| A | `/dashboard/fuel-officer-structures` · [燃料官建筑](<static/src/views/dashboard/fuel-officer-structures/index.vue#L1-L31>) | 使用了 `ArtTable`，但[分页、API 请求、加载](<static/src/views/dashboard/fuel-officer-structures/index.vue#L107-L134>)另起维护，分页/错误空态与其他建筑表分叉；缺默认表头组件属于推荐项。 |
| B | `/dashboard/galaxy-registry` · [星系登记](<static/src/views/dashboard/galaxy-registry/index.vue#L180-L202>) | 个人/管理历史有 ledger；实时登记、SDE 搜索、排行榜不是账本。页面部分表格复用了工具栏但多种空态/表头同屏，可做同角色窄屏对照；[金额局部 formatter](<static/src/views/dashboard/galaxy-registry/index.vue#L1351-L1357>)另列展示规范复核。 |

### 我的信息（10 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| C | `/info/wallet` · [个人钱包](<static/src/views/info/wallet/index.vue#L1-L91>) | 钱包余额、分析折线与 ledger 职责分开，自然增长；分析图若溢出优先查共享图表，不打散专用布局。 |
| B | `/info/skill` · [技能](<static/src/views/info/skill/index.vue#L56-L98>) | 分组折叠用点击 `div`，同数据另有键盘可达的 `ElSelect`，不是同样的直接操作；保留技能目录布局。另有可见 `Unknown` 回退。 |
| A | `/info/npc-kills` · [个人刷怪](<static/src/views/info/npc-kills/index.vue#L230-L246>) | 汇总保留报表，逐条历史虽有 `ArtTable`，但没启 ledger 且[默认 20](<static/src/views/info/npc-kills/index.vue#L361-L373>)。 |
| A | `/info/ships` · [舰船](<static/src/views/info/ships/index.vue#L94-L149>) | 分组纯点击 `div`，卡片能力详情依赖悬停类，触屏和键盘没有等价发现路径；另外有 `ID/Lv/Unknown` 本地文案。舰船图标/市场目录保留。 |
| B | `/info/implants` · [植入体](<static/src/views/info/implants/index.vue#L194-L215>) | 专用属性分组合理；疲劳时间拼接 `h/m` 未接应用语言，按低风险展示差异列入。 |
| A | `/info/fittings` · [装配](<static/src/views/info/fittings/index.vue#L70-L121>) | 分组和打开详情只靠点击 `div`，无键盘等价入口；另有 680px 详情弹窗在中间断点的风险。装配树/卡片不应改成 CRUD。 |
| A | `/info/assets` · [资产](<static/src/views/info/assets/index.vue#L47-L87>) | 卡片内纵向滚动规则可保留；位置和容器展开仅用 `div`，无默认焦点和键盘操作。位置内分页不是页面级账本。 |
| B | `/info/contracts` · [合同](<static/src/views/info/contracts/index.vue#L1-L55>) | 正式 ledger/全高表格可作为对照；只因[详情弹窗](<static/src/views/info/contracts/modules/contract-detail-dialog.vue#L1-L12>)700px 无中间断点上限列 B，详情子表可保留。 |
| B | `/info/esi-check` · [ESI 检测](<static/src/views/info/esi-check/index.vue#L1-L71>) | 专用能力矩阵可以保留；[总览快捷入口](<static/src/views/info/esi-check/modules/overview-matrix.vue#L18-L31>)主要靠点击，详情另有可选同人物的键盘入口。 |
| E | `/info/tool-bookmarks` · [工具书签](<static/src/views/info/tool-bookmarks/index.vue#L1-L78>) | 目录卡片、添加/编辑对话框与空态合理，不强制改为表格。 |

### 新人、技能计划（9 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| C | `/newbro/select-mentor` · [选导师](<static/src/views/newbro/select-mentor/index.vue#L1-L38>) | 与记录卡片规范一致，让内容自然增长；保留选择卡和申请区。 |
| A | `/newbro/captain` · [队长补贴入口](<static/src/views/newbro/captain/index.vue#L1-L4>) | 历史空态重复及工具条层级来自共同 Page，不双计组件根因；队长表单结构可保留。 |
| A | `/newbro/manage` · [队长补贴管理入口](<static/src/views/newbro/manage/index.vue#L1-L4>) | 同一共享 Page 的空态重复和表头关联问题；管理模式额外表单不要求复制申请页外观。 |
| A | `/newbro/mentor` · [我的学员](<static/src/views/newbro/mentor/index.vue#L121-L195>) | 主学员表原生分页；全高 tabs/[高度链](<static/src/views/newbro/mentor/index.vue#L369-L381>)不完整。奖励阶段静态子表不是待整改账本。 |
| A | `/newbro/mentor-manage` · [导师管理](<static/src/views/newbro/mentor-manage/index.vue#L125-L175>) | 关系列表手写原生分页；奖励记录用了 ledger 样式，但[默认 50 并手写页大小](<static/src/views/newbro/mentor-manage/index.vue#L367-L378>)。阶段编辑表的内联可编辑行为保留。 |
| A | `/newbro/recruit-link` · [专属招募](<static/src/views/newbro/recruit-link/index.vue#L113-L151>) | 管理端 ledger 被覆盖 `[20,50,100]`，默认[请求 20](<static/src/views/newbro/recruit-link/index.vue#L377-L390>)；个人链接嵌入式名单不等于账本，成功记录弹窗 760px 另作 B。 |
| A | `/skill-planning/completion-check` · [技能完成度检查](<static/src/views/skill-planning/completion-check/index.vue#L476-L494>) | 选择器仅合并公共、个人的第一页各 200，并从已选项清掉不在切片的值；与[全量候选要求](<docs/features/current/skill-planning.md#L55-L60>)不符，不能靠再提高 page size 根治。 |
| A | `/skill-planning/skill-plans` · [公共技能计划](<static/src/views/skill-planning/skill-plans/index.vue#L308-L367>) | 16 张卡一页是产品布局；问题是重排在途可编辑旧 `sort_order` 并由编辑表单回写，不是卡片模式问题。760px 编辑弹窗另作 B。 |
| A | `/skill-planning/personal-skill-plans` · [个人技能计划](<static/src/views/skill-planning/personal-skill-plans/index.vue#L296-L356>) | 与公共计划相同的重排/编辑窗口。服务端[给整个范围重编号](<server/internal/service/skill_plan.go#L1037-L1059>)，不是前端把切片强改 0..n−1，但也不保留原排序槽位；先厘清契约再修。 |

### 公开入口、异常与嵌入（7 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| A | `/auth/login` · [EVE SSO 登录](<static/src/views/auth/login/index.vue#L1-L31>) | 登录为无焦点/键盘绑定的点击图片；SSO 品牌图片和独立登录外壳保留，需统一可操作语义。 |
| E | `/auth/callback` · [SSO 回调](<static/src/views/auth/callback/index.vue#L1-L44>) | 回调中转及首次联系信息有专用布局，无需表格、外壳或内容标题模板。 |
| E | `/r/:code` · [招募落地页](<static/src/views/auth/recruit/index.vue#L1-L54>) | 公共招募/SSO 场景，独立页面合理，不应套登录后后台布局。 |
| E | `/403`、`/exception/403` · [403](<static/src/views/exception/403/index.vue#L1-L16>) | 与其他异常共用 `ArtException`；路由冲突另列，不视为 UI 布局问题。 |
| E | `/:pathMatch(.*)*`、`/exception/404` · [404](<static/src/views/exception/404/index.vue#L1-L16>) | 同上。 |
| E | `/500`、`/exception/500` · [500](<static/src/views/exception/500/index.vue#L1-L16>) | 同上。 |
| E | `/outside/iframe/:path` · [嵌入壳](<static/src/views/outside/Iframe.vue#L1-L36>) | 站内承载外部页面，外部正文不受 Vue 主题支配；不把外站布局计为 Vue 不一致。 |

### 系统管理（13 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| C | `/system/salary` · [工资](<static/src/views/system/salary/index.vue#L110-L137>) | 这是响应式配置卡和表单，允许不同于表格页；未见独立布局偏离。 |
| A | `/system/blacklist` · [黑名单](<static/src/views/system/blacklist/index.vue#L178-L200>) | 搜索/重置后保留旧页；20 条对于有界配置不是不合规的原因。 |
| A | `/system/user` · [用户管理](<static/src/views/system/user/index.vue#L511-L539>) | 筛选进入 `getData` 的保页路径；可见的条件标签与结果不一定对应到第 1 页。角色卡/移动适配与普通行密度可有差别。 |
| A | `/system/task-manager` · [任务管理](<static/src/views/system/task-manager/index.vue#L1-L46>) | 子 Tab 中，执行历史和 ESI 状态筛选留旧页；ESI 状态快照用 ledger 却以 20 起始（不在默认选项），应选与快照匹配的预设，而不是强制取 200。任务定义、监控和 ESI 控制不强制同一种呈现。 |
| A | `/system/wallet` · [系统钱包](<static/src/views/system/wallet/index.vue#L190-L223>) | 三个列表各自存在筛选留旧页；跨 Tab “查看指定用户流水”也可能沿用旧页。父级高度链较完整，分析区窄屏日期控件另列 B。 |
| A | `/system/audit` · [审计日志](<static/src/views/system/audit/index.vue#L281-L303>) | ledger 与 200 正确；搜索/重置仍沿用旧页，空态可能吞掉分页恢复。 |
| E | `/system/pap-exchange` · [PAP 汇率](<static/src/views/system/pap-exchange/index.vue#L96-L127>) | 原生表是内嵌可编辑配置子表而非分页流水；不要求转 ledger。 |
| A | `/system/pap` · [联盟 PAP](<static/src/views/system/pap/index.vue#L166-L184>) | 改月份只写搜索参数后保页请求；已有 200 和共享账本。 |
| A | `/system/auto-role` · [自动角色](<static/src/views/system/auto-role/index.vue#L19-L65>) | 两个 Tab 的普通映射主清单均另用原生表并固定 `border/stripe`；另一套[职务映射](<static/src/views/system/auto-role/index.vue#L68-L129>)也相同。Tag 和删除操作可由共享表表达；不需要 ledger。 |
| A | `/system/user-center` · [个人中心可见入口](<static/src/router/modules/system.ts#L132-L141>) | 已注册的旧模板：[资料卡](<static/src/views/system/user-center/index.vue#L3-L32>)固定栏宽与示例展示，[个人/密码表单](<static/src/views/system/user-center/index.vue#L112-L150>)和[示例初始值](<static/src/views/system/user-center/index.vue#L173-L193>)与当前 EVE SSO／角色资料呈现不一致；优先确认产品去留，不建议美化或扩展旧密码流程。 |
| C | `/system/webhook` · [Webhook 配置](<static/src/views/system/webhook/index.vue#L1-L65>) | 有界配置表单，不要求套表格或统一成 ledger。 |
| C | `/system/newbro-settings` · [新人设置入口](<static/src/views/system/newbro-settings/index.vue#L1-L4>) | 配置聚合复用模块；和补贴申请历史不是同一布局类别。 |
| B | `/system/basic-config` · [基础配置](<static/src/views/system/basic-config/index.vue#L1107-L1156>) | 表单按业务分区是合理的；仍有独立双列字段布局，先验证窄屏长英文 label 和卡片密度，再决定统一。 |

### 入团、舰队与 PAP（7 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| A | `/admissions/{new,alt,history,review,applications/:id/edit,applications/:id}` · [入团共用页](<static/src/views/admissions/index.vue#L46-L108>) | 历史/审批的同一页面手写页码、20 条非 ledger；[时间格式](<static/src/views/admissions/index.vue#L436-L440>)也独立。提交、编辑与申请说明保留表单式布局，不将六条路由分别计数。 |
| A | `/operation/fleets` · [舰队列表](<static/src/views/operation/fleets/index.vue#L324-L335>) | 搜索沿用旧页；20 和非 ledger 是否在后续划为历史，应区分活动列表与明确的历史视图，不以该条判 A。 |
| B | `/operation/fleet-configs` · [舰队配置](<static/src/views/operation/fleet-configs/index.vue#L1-L33>) | 有界主表采用共享结构；[复杂配置弹窗](<static/src/views/operation/fleet-configs/modules/fleet-config-dialog.vue#L1-L18>)760px 和内嵌管理控件需复验中间断点。 |
| A | `/operation/fleet-detail/:id` · [舰队详情](<static/src/views/operation/fleet-detail/index.vue#L140-L185>) | 邀请码栏固定浅灰及[复制邀请链接自建剪贴板/反馈](<static/src/views/operation/fleet-detail/index.vue#L486-L498>)。复制的是合成 URL，可保留文字动作，改共享 hook 即可；原生邀请子表是详情例外。 |
| E | `/operation/corporation-pap` · [军团 PAP](<static/src/views/operation/corporation-pap/index.vue#L63-L91>) | 已用共享账本，默认 200，专用请求同时返回汇总与分页列表；[搜索/重置及换页逻辑](<static/src/views/operation/corporation-pap/index.vue#L238-L293>)具备等价分页语义，不仅凭未导入 `useTable` 判不合规。 |
| A | `/operation/pap` · [个人 PAP](<static/src/views/operation/pap/index.vue#L88-L110>) | 内部和联盟两块流水分别原生分页，见[第二块](<static/src/views/operation/pap/index.vue#L226-L252>)；共享列行为、默认请求和分页呈现没有复用。 |
| A | `/operation/join` · [加入舰队](<static/src/views/operation/join/index.vue#L1-L22>) | 单一表单居中是合理特例；问题是邀请码正文固定使用浅色 `bg-gray-50`，未随主题变化；420px 卡在窄屏能否随外层收缩另作 B。 |

### 商城、补损、福利（10 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| A | `/shop/browse` · [商城浏览](<static/src/views/shop/browse/index.vue#L169-L189>) | 全高 Tab 裁剪但商品网格无纵滚；多商品时尾部和分页有不可达风险。我的订单 20 条非 ledger、切换状态留在旧页。不要把商品卡改成表格。 |
| A | `/shop/manage` · [商品管理](<static/src/views/shop/manage/index.vue#L1-L17>) | 有界配置表自身可保留，筛选留旧页；缺陷在嵌套商品管理模块。 |
| A | `/shop/order-manage` · [订单管理](<static/src/views/shop/order-manage/index.vue#L40-L57>) | Tab 内容隐藏溢出、最小高度链不完整；当前订单和历史都存在筛选错页。历史 200 已对齐，待处理/当前订单不靠默认 200 代表风格。 |
| C | `/shop/wallet` · [商城钱包](<static/src/views/shop/wallet/index.vue#L21-L47>) | 余额摘要配共享 ledger/分页；未见独立根因。 |
| A | `/srp/srp-apply` · [补损申请](<static/src/views/srp/apply/index.vue#L104-L125>) | 申请表可保留，个人申请历史虽是 `ArtTable`，却[默认 20](<static/src/views/srp/apply/index.vue#L196-L213>)且不是 ledger；现有 API 请求上限须一并考虑。 |
| A | `/srp/srp-manage` · [补损管理](<static/src/views/srp/manage/index.vue#L117-L157>) | 审核列表用 ledger，但高级筛选/重置保留旧页，且候选由当前页数据生成，见 [useSrpManage](<static/src/hooks/srp/useSrpManage.ts#L366-L476>)；另有宽弹窗。 |
| B | `/srp/srp-prices` · [补损价格](<static/src/views/srp/prices/index.vue#L1-L49>) | 有界价格区已用共享表；多 Tab 下须补查 wrapper 高度及空态/分页，不能按历史账本计。 |
| A | `/welfare/my` · [我的福利](<static/src/views/welfare/my/index.vue#L238-L280>) | 历史 ledger 的 200 与福利目录的 10 可并存；问题是目录筛选使用保页刷新，可能在新条件下直接空态。 |
| A | `/welfare/approval` · [福利审批](<static/src/views/welfare/approval/index.vue#L348-L365>) | 待审与历史均有共享 ledger 和 200；标签/筛选条件切换的加载策略仍保留旧页，见处理器。 |
| A | `/welfare/settings` · [福利设置](<static/src/views/welfare/settings/index.vue#L497-L510>) | 筛选留旧页、技能计划候选一次只取 200；重排保存期间[可打开编辑旧 `sort_order`](<static/src/views/welfare/settings/index.vue#L659-L677>)，[重排等待窗口](<static/src/views/welfare/settings/index.vue#L754-L782>)没有禁止入口。服务端保留排序槽位，不是“页内重新编号”的问题。 |

### 工单（7 页）

| 等级 | 页面与定位 | 结论 |
| --- | --- | --- |
| A | `/ticket/my-tickets` · [我的工单](<static/src/views/ticket/my-tickets/index.vue#L54-L109>) | 持续增长记录非 ledger、20 条；筛选后 `getData()` 留旧页。 |
| A | `/ticket/create` · [新建工单](<static/src/views/ticket/create/index.vue#L1-L30>) | 类别始终显示 `name`，不按语言选择 `name_en`，而[工单管理使用 locale](<static/src/views/system/ticket-management/index.vue#L54-L73>)；类别 320px＋label 110px 需实测窄屏。 |
| B | `/ticket/detail/:id` · [工单详情](<static/src/views/ticket/detail/index.vue#L1-L44>) | 专用对话流合理；共享回复卡的无空格长文本/单行头部需做窄屏复验。 |
| A | `/ticket/management` · [工单管理](<static/src/views/system/ticket-management/index.vue#L83-L96>) | 20 条非 ledger，切换查询条件[沿用旧页](<static/src/views/system/ticket-management/index.vue#L172-L187>)；不应靠把 Tab 样式调成一致来掩盖。 |
| B | `/ticket/categories` · [工单类别](<static/src/views/system/ticket-categories/index.vue#L1-L34>) | 有界共享表格合理；对话框保存无 pending 指示，[删除](<static/src/views/system/ticket-categories/index.vue#L164-L175>)没有多数管理页的二次确认，待确认是否有意简化。 |
| C | `/ticket/statistics` · [工单统计](<static/src/views/system/ticket-statistics/index.vue#L1-L43>) | 不分页的指标和分组卡，暂无独立偏离。 |
| B | `/ticket/admin-detail/:id` · [管理端工单详情](<static/src/views/system/ticket-detail/index.vue#L27-L58>) | 原生状态历史是详情子表可保留；与普通详情一致受回复卡长文本风险影响，另检小屏操作组，不能硬套后台列表。 |

**关联契约提醒（不另计）：**入团、工单、个人补损及商城我的订单现有 API 有 100 条上限；不能只把界面改成 200 就宣称统一。定位：[入团](<server/internal/handler/admission.go#L43-L58>)、[工单](<server/internal/handler/ticket.go#L47-L102>)、[补损](<server/internal/handler/srp.go#L406-L413>)、[我的订单](<server/internal/service/shop.go#L273-L276>)。商城管理历史已走另一套 ledger 规格。

## 四、常规组件发现

以下记录的是**组件本身**的原因；组件所影响页面见前表。全局样式文件、hook、store 是证据或共同根因，不按 Vue 组件再计一次。

| 等级 | 组件与证据 | 结论 |
| --- | --- | --- |
| A | [ArtTable](<static/src/components/core/tables/art-table/index.vue#L242-L293>) | 当 `total > 0` 但当前页为空时，分页整条隐藏；无法直接回到有效页。另在复杂 Card/Tab 中按[紧邻父层查询表头](<static/src/components/core/tables/art-table/index.vue#L309-L342>)并使用[44px 回退](<static/src/hooks/core/useTableHeight.ts#L57-L81>)，嵌套或错绑需实屏检查；分页只在 ≤640px [开启换行](<static/src/components/core/tables/art-table/style.scss#L97-L106>)。 |
| A | [ArtTableHeader](<static/src/components/core/tables/art-table-header/index.vue#L8-L61>) | 刷新/全屏是无键盘语义的点击 `div`；工具条 ≤640px 整块隐藏是共享策略，但无等价入口时不能把页面结果算成有该功能。其[卸载退出全屏](<static/src/components/core/tables/art-table-header/index.vue#L266-L318>)清理 DOM 却不清理 store，且未覆盖 KeepAlive 停用；[全屏状态与其他表格联动并被持久化](<static/src/store/modules/table.ts#L72-L96>)。 |
| A | [ArtButtonTable](<static/src/components/core/forms/art-button-table/index.vue#L1-L16>) | 行内按钮是点击 `div`，无默认键盘焦点；图标型动作也没有组件默认可访问名。不要为每页再另造鼠标按钮；应统一键盘、加载、禁用与危险动作语义。 |
| A | [ArtIconButton](<static/src/components/core/widget/art-icon-button/index.vue#L1-L23>) | 通用图标按钮是 `div`，在[页头侧栏/刷新/主题开关](<static/src/components/core/layouts/art-header-bar/index.vue#L29-L44>)和[主题入口](<static/src/components/core/layouts/art-header-bar/index.vue#L81-L86>)实际作为按钮使用。裸实例无默认焦点/键盘激活；不要以某些 Dropdown 触发器注入键盘行为替代所有消费者的验证。 |
| A | [ArtSearchBar](<static/src/components/core/forms/art-search-bar/index.vue#L71-L96>) | 查询/重置是原生按钮基元，展开/收起却是点击 `div`；只用键盘不能直接展开隐藏字段。现用在用户管理和 PAP 搜索模块；不要求全站所有搜索条扩展相同功能。 |
| A | [AuthTopBar](<static/src/components/core/views/login/AuthTopBar.vue#L11-L68>) | 登录顶部主题切换和配色选项为点击 `div`，配色展开另依赖悬停样式；独立登录布局可保留，键盘可用性与同类入口应一致。 |
| A | [ArtGlobalSearch](<static/src/components/core/layouts/art-global-search/index.vue#L47-L89>) | 搜索具有方向键/回车导航，不应说整个搜索不能键盘用；但删除某条历史是只绑定鼠标的嵌套 `div`。另有[专用暗色硬编码](<static/src/components/core/layouts/art-global-search/index.vue#L377-L406>)及窄屏英文页脚不换行风险（B）。 |
| A | [队长补贴历史 Page](<static/src/components/captain-refund/Page.vue#L59-L79>) | 使用共享 ledger 和 200 没有问题；空列表同时渲染 `ArtTable` 内置空态与页面 `ElEmpty`，一处状态出现两次。工具条在 `ElCard` header 插槽、表体在 body，ArtTable 的父层查找不会找到该工具条；涉及 `/newbro/captain`、`/newbro/manage`，不包括单独使用 Settings 的页面。 |
| A | [MumbleCredentialCard](<static/src/components/mumble-credential-card.vue#L13-L24>) | 用户名/节点分享已经采用共享复制；一次性口令允许保留独立文字按钮，但[直接调用剪贴板加自建 toast](<static/src/components/mumble-credential-card.vue#L164-L170>)，未复用规则要求的 hook。不得改变一次性口令展示规则。 |
| B | [KmPreviewDialog](<static/src/components/business/KmPreviewDialog.vue#L1-L10>) | 共享的 680px 固宽，641–679px 段无全局移动端兜底；需验证横向边界与装备名长文本，不判定 ≤640px 必裁。影响 SRP 申请和管理。 |
| B | [ArtLineChart](<static/src/components/core/charts/art-line-chart/index.vue#L1-L8>) | 容器统一写成 `100% + 10px`，而[水平柱图](<static/src/components/core/charts/art-h-bar-chart/index.vue#L1-L8>)使用整宽。外栏有留白时可能是有意补偿，先检查满宽图卡、双列边界和 tooltip，不能直接判所有折线图溢出。 |
| B | [TicketReplyItem](<static/src/components/ticket/TicketReplyItem.vue#L28-L48>) | 回复头为单行 flex，正文只 `pre-wrap`，未给超长单词/链接单独断行策略；窄屏长昵称＋时间＋内部备注，以及无空格长链接需实测。使用主题色的条目面本身没有问题。 |

### 页面局部组件的独立发现

| 等级 | 组件与证据 | 结论 |
| --- | --- | --- |
| A | [PAP 结算弹窗](<static/src/views/system/pap/modules/pap-settle.vue#L53-L91>) | 结算完成只更新弹窗内结果，未通知[父列表](<static/src/views/system/pap/index.vue#L25-L37>)刷新；关闭再开可能仍见旧成功结果。 |
| A | [钱包列表](<static/src/views/system/wallet/modules/wallet-list.vue#L125-L135>) | 搜索留旧页。 |
| A | [钱包日志](<static/src/views/system/wallet/modules/wallet-logs.vue#L164-L175>) | 筛选留旧页。 |
| A | [钱包交易](<static/src/views/system/wallet/modules/wallet-transactions.vue#L214-L231>) | 筛选及按用户跳转的查询留旧页。 |
| A | [钱包分析](<static/src/views/system/wallet/modules/wallet-analysis.vue#L485-L494>) | 日期控件固定 280px 且不可缩，[Tab 禁止横向滚动](<static/src/views/system/wallet/index.vue#L218-L223>)；在 320px 视口扣除外壳和卡片内边距后可用宽度小于 280px。现有纵滚应保留。 |
| A | [ESI 状态 Tab](<static/src/views/system/task-manager/modules/EsiStatusesTab.vue#L254-L270>) | 筛选留旧页；[20 的初次请求](<static/src/views/system/task-manager/modules/EsiStatusesTab.vue#L148-L157>)与已用 ledger 的可选页量不一致。 |
| A | [任务历史 Tab](<static/src/views/system/task-manager/modules/HistoryTab.vue#L226-L242>) | 历史 ledger 正确；应用筛选/重置的保页语义不一致。 |
| A | [舰队搜索](<static/src/views/operation/fleets/modules/fleet-search.vue#L3-L18>) | 优先级选项直接写入英文（例如 `Other`），与双语状态文案不一致；实时单筛可以保留。父级的错页不在这里重复计原因。 |
| A | [商城商品网格](<static/src/views/shop/browse/modules/shop-products.vue#L69-L81>) | 特定体量的卡分页合理，但页码、总量、尺寸固定同一行；[局部容器](<static/src/views/shop/browse/modules/shop-products.vue#L201-L206>)未做换行，网格也有[最小 220px 列](<static/src/views/shop/browse/modules/shop-products.vue#L125-L130>)。外层纵向裁剪归商城页，不归组件重复统计。 |
| A | [商城我的订单](<static/src/views/shop/browse/modules/shop-orders.vue#L18-L29>) | 可增长历史采用 20＋普通分页；[状态筛选](<static/src/views/shop/browse/modules/shop-orders.vue#L181-L187>)不复位。 |
| A | [商品管理](<static/src/views/shop/manage/modules/manage-products.vue#L290-L306>) | 搜索/重置进入保页路径，有界管理无需 ledger。 |
| A | [当前订单管理](<static/src/views/shop/manage/modules/manage-orders.vue#L238-L254>) | 搜索/重置进入保页路径；待批记录是否应按无界 ledger，随产品确认保留周期。 |
| A | [历史订单管理](<static/src/views/shop/manage/modules/manage-order-history.vue#L185-L199>) | 已用正确账本；搜索与刷新当前页用了相同保页路径。 |
| B | [PAP 搜索栏](<static/src/views/system/pap/modules/pap-search.vue#L1-L23>) | 三个动作共用搜索网格一格，部分操作共用 loading，窄屏和反馈需要区分。 |
| B | [ESI 控制栏](<static/src/views/system/task-manager/modules/EsiControls.vue#L340-L352>) | 标题和动作不换行，长英文及窄屏要实测。 |
| B | [ESI 总览快捷矩阵](<static/src/views/info/esi-check/modules/overview-matrix.vue#L18-L31>) | 快捷入口的点击方式与通用按钮不同，但用户可使用详情内可键盘操作的同人物选择器。 |
| B | [合同详情](<static/src/views/info/contracts/modules/contract-detail-dialog.vue#L1-L12>)、[舰队配置对话框](<static/src/views/operation/fleet-configs/modules/fleet-config-dialog.vue#L1-L18>) | 700/760px 固宽；641px 以上尚不到目标宽度的区间需验证。合同明细表允许，配置弹窗的显式内滚保留。**两个组件**。 |
| B | [公共计划编辑](<static/src/views/skill-planning/skill-plans/modules/skill-plan-dialog.vue#L1-L17>)、[个人计划编辑](<static/src/views/skill-planning/personal-skill-plans/modules/personal-skill-plan-dialog.vue#L1-L17>) | 两个对话框都固定 760px，需要核对中间断点。排序竞态的打开状态在页面，不能靠只改 CSS 或删数字排序输入解决。**两个组件**。 |

### 外壳全局风险（不并入组件、页面数量）

- [外壳移动断点](<static/src/views/index/style.scss#L49-L93>)在不同段使用 `100dvh` 和文档滚动；[高度 hook](<static/src/hooks/core/useLayoutHeight.ts#L86-L116>)仍以 `100vh` 计算全高。属于 641–1180px、移动地址栏和软键盘变化的共享风险，需检查实际可视高度和最后一行，不重复记到所有页面。
- [异常静态路由](<static/src/router/routes/staticRoutes.ts#L34-L50>)与[动态异常子路由](<static/src/router/modules/exception.ts#L12-L45>)三组同名，[注册器](<static/src/router/core/RouteRegistry.ts#L40-L55>)只查父路由名。使用仓库锁定的 Vue Router 做内存 history 核验：添加动态父路由后，同名静态记录被替换，catch-all 丢失；注销动态记录也不会恢复静态记录。这是错误页面**入口**问题，不是错误页样式缺陷，故不抬高常规风格计数。

## 五、指定单列：军团公告与名人堂

### 军团公告 `/announcements`

- 保留专用横幅、左档案/右正文、未读与选中表达及[产品约定每页 10 条](<docs/features/current/announcements.md#L15-L23>)；这是阅读应用，不是 CRUD 表格。[页面](<static/src/views/announcements/index.vue#L2-L81>)使用带 `aria-current` 的真实按钮与轻量翻页，[样式](<static/src/views/announcements/announcements.css#L377-L399>)在 ≤760px 变单列并给档案显式纵向滚动，已有[焦点样式](<static/src/views/announcements/announcements.css#L123-L139>)。
- **单列 A：**[列表时间](<static/src/views/announcements/index.vue#L61-L69>)、[正文时间](<static/src/views/announcements/index.vue#L136-L142>)直接 `toLocaleString()`，未用统一时间戳格式。**单列 B：**[编辑区](<static/src/views/announcements/index.vue#L82-L104>)另接原生 WangEditor，发布/提交采用专用样式与默认按钮；只建议验证编辑器在暗色、窄屏、长正文和保存状态的可读性，不能以此要求删除公告品牌外观或更换内容编辑流程。

### 名人堂 `/fuxi-hall`

- [路由与 Tabs](<static/src/views/fuxi-hall/index.vue#L1-L31>)是**一个入口**；公开画廊与管理是同路由两个子视图，[路由测试](<static/src/router/modules/index.test.ts#L17-L26>)也明确约束这一点。公开场景的横幅、分层展陈、成员卡和[独立明暗 token](<static/src/assets/fuxi-hall/hall.css#L1-L19>)可保留；[楼层目录](<static/src/views/fuxi-hall/hall-of-fame/index.vue#L5-L45>)有按钮/目标关联、可聚焦楼层，并[处理 sticky 页头偏移](<static/src/views/fuxi-hall/hall-of-fame/index.vue#L93-L100>)。卡片的贡献、身份、详情不宜替换成管理表格。
- **单列 A：**[成员卡角色名复制](<static/src/views/fuxi-hall/components/FuxiHallMemberCard.vue#L18-L29>)与[QQ 复制](<static/src/views/fuxi-hall/components/FuxiHallMemberCard.vue#L49-L60>)虽已让共享 hook 处理反馈，但仍重复实现相同的紧凑行内复制按钮；应保留字段可见性与现有键盘语义。另有[成员卡贡献更新时间](<static/src/views/fuxi-hall/components/FuxiHallMemberCard.vue#L61-L72>)把时间戳改为浏览器本地日期，未发现要求仅显示日历日期的规范例外；[管理表排序上下按钮](<static/src/views/fuxi-hall/manage/index.vue#L92-L99>)只提供箭头，没有像[楼层按钮](<static/src/views/fuxi-hall/manage/index.vue#L39-L49>)那样提供可读操作名称。**单列 B：**[管理表](<static/src/views/fuxi-hall/manage/index.vue#L75-L149>)为无分页原生 `ElTable`，应按有界管理/配置而不是无界账本审查；检查窄屏固定操作列、大批记录、右侧预览及长名称。**只由名人堂消费的** [ArtWangEditor 样式](<static/src/components/core/forms/art-wang-editor/style.scss#L59-L96>)仍有局部固定色（B，验明暗后再决定是否适配）。
- 上述两项不进入常规 A/B 数量。审计未改公告、名人堂或任何其他页面。

## 六、覆盖校验与不计项目

- **常规组件的剩余 64 个**不是遗漏：上述两张组件表既登记明确根因也登记复验项，其余组件在本次源码维度未见需单独归因的问题。主要覆盖：Logo/SVG/主题图标、横向/混合/侧栏菜单与递归项、工作 Tab 与右键菜单、用户菜单、设置面板、抽屉和其余设置子件、表单及导入导出、图表和数值动画、登录插画和异常模板，以及业务选择器、补贴配置/计算、工单状态徽标。保留有界摘要与详情子表，而不是把 64 个都标成视觉验收通过。
- **37 个常规局部组件都纳入覆盖。**未单独计数的 17 个：工作台 [CardList](<static/src/views/dashboard/console/modules/card-list.vue>)、[FleetList](<static/src/views/dashboard/console/modules/fleet-list.vue>)、[PapChart](<static/src/views/dashboard/console/modules/pap-chart.vue>)、[SrpList](<static/src/views/dashboard/console/modules/srp-list.vue>)（固定摘要、显式滚动或共享图表）；[建筑服务详情](<static/src/views/dashboard/corporation-structures/modules/structure-services-dialog.vue>)、[个人钱包分析](<static/src/views/info/wallet/modules/wallet-analytics.vue>)、[ESI 人物详情](<static/src/views/info/esi-check/modules/character-detail.vue>)；[导师职责](<static/src/views/newbro/components/mentor-responsibilities-card.vue>)、[招募设置面板](<static/src/views/newbro/components/admin-settings-panel.vue>)、[SSO 联系信息补录](<static/src/views/auth/callback/contact-registration.vue>)；[入团表单](<static/src/views/admissions/admission-form.vue>)、[入团说明](<static/src/views/admissions/admission-instructions.vue>)、[入团证据图片](<static/src/views/admissions/evidence-image.vue>)；[用户搜索](<static/src/views/system/user/modules/user-search.vue>)、[用户管理对话框](<static/src/views/system/user/modules/user-manage-dialog.vue>)、[ESI 监控](<static/src/views/system/task-manager/modules/EsiMonitorTab.vue>)、[任务定义 Tab](<static/src/views/system/task-manager/modules/TasksTab.vue>)。例如用户搜索组件自身发事件正确，父级错页不归咎于子件。
- **筛选留旧页有 20 处独立调用文件，涉及 16 个常规页面。**分别为系统的审计、黑名单、用户、PAP、钱包（3）、任务（2）；舰队列表；商城浏览、商品管理、订单管理（2）；补损管理；我的工单、工单管理；福利的我的、审批、设置。该数是一个具体问题的横向覆盖，不等于额外增加 16 页。重排编辑、候选集合和滚动另按前表。
- **未在用的 32 个 SFC 不计入现有 UI 缺陷。**七个视图文件是 [旧注册](<static/src/views/auth/register/index.vue>)和工作台旧模板 [AboutProject](<static/src/views/dashboard/console/modules/about-project.vue>)、[ActiveUser](<static/src/views/dashboard/console/modules/active-user.vue>)、[DynamicStats](<static/src/views/dashboard/console/modules/dynamic-stats.vue>)、[NewUser](<static/src/views/dashboard/console/modules/new-user.vue>)、[SalesOverview](<static/src/views/dashboard/console/modules/sales-overview.vue>)、[TodoList](<static/src/views/dashboard/console/modules/todo-list.vue>)。25 个共享文件集中在旧横幅（2）、返回顶部（1）、卡模板（8）、图表模板（4）、表单模板（3）、未用/关闭的布局插件（3）、媒体模板（2）、结果模板（1）、工单类别 Tag（1）。关闭的锁屏/聊天有单独配置，只有模板或类型声明不算引用。未导出的[角色路由草案](<static/src/router/modules/role.ts#L1-L17>)不计；iframe 有声明但当前模块未发现配置实例，见[取地址的方式](<static/src/views/outside/Iframe.vue#L23-L33>)。
- **文案观察不等于颜色/排版缺陷。**前表仅把明确的同场景反馈或语言差异计为 A/B；比如技能和舰船分组的 `Unknown`、[装配图片的固定英文 alt](<static/src/views/info/fittings/index.vue#L113-L121>)可收敛到 i18n，但游戏内名称、角色名、用户输入、缩写不一概替换。在[个人中心](<static/src/views/system/user-center/index.vue#L173-L193>)先决定弃用还是产品化，而不是逐个补齐旧模板。


## 七、建议验收顺序（待授权，不是本次实施）

1. **先修共同原因**：约定唯一滚动拥有者、共享表头与表的关联、空结果保留可恢复分页、筛选行为回到第 1 页。以有数据的第 2 页切换到只余 1 条的过滤条件，验证空态、回页和刷新各自正确。
2. **按业务形态整批验收**：账本验证首请求 200＋ledger；有界配置使用默认密度，报表/子表/选择器保留专用结构；没有成文基线的卡片、表单、详情只统一间距、状态反馈和可达性，不统一成同一种排版。
3. **回归矩阵**：320、375、640、641、768、1024、1440px；至少测 0、1、跨页三种数据量，长中英文、亮/暗两主题、键盘 Tab/Enter/Escape，及站内 Tab、浏览器返回与多表实例。检查最后一行、分页和提交按钮可到达，且没有整页横向滚动。公告/名人堂单独做选中与未读、楼层定位、编辑状态和品牌对比度回归。

**本次验证边界：**使用仓库快照与本机依赖做 SFC/路由统计、源码调用链和异常路由内存验证；未使用 DSH 界面替代项目页面，没有 Vue 站点 URL、测试身份或浏览器截图，未做登录后的真实视口、主题、数据量或像素验收。本次只新建这份报告，未运行生产构建或全量测试；不将需要实屏的风险描述成实测结果。收尾时检查到工作区有并行任务的其他改动（包括 Vue 路由测试），本次未修改、还原或纳入审计归因；应用的 Vue 页面和样式与审计基线一致。
