# Anmo 品牌与设计规范（BRAND-GUIDELINES）

> 版本 1.0（2026-09-28）。**本文件是双端（H5 / 微信小程序）UI 的唯一视觉与组件规范。**
> 组件体系遵循 shadcn/ui 的设计语言与命名（token 化主题、语义色、受控状态、组合式原语），按双端技术栈落地。
> 任何页面改造不得绕开本文档自行取色/取距。

## 1. 品牌调性

安摩是一间**个人到店推拿按摩店**：沉静、专业、有温度。视觉上追求「宣纸 + 绛红印章 + 黄铜配件」的中式养生质感——不炫技、不冷冰冰、不满屏促销。

- **暖**：米纸底色、暖墨色文字，拒绝冷灰蓝。
- **定**：低饱和绛红做主色，只有一处主按钮/主强调，页面永远只有一个视觉焦点。
- **准**：信息层级靠字重与字号表达，不靠彩色堆砌；数字一律等宽对齐。

## 2. 设计 Token（双端共享，命名对齐 shadcn/ui）

### 2.1 色彩（Light 主题，仅此一套）

| Token | 值 | 用途 |
|---|---|---|
| `--background` | `#F6F4F0` | 页面底（暖宣纸） |
| `--card` | `#FFFFFF` | 卡片/浮层底 |
| `--foreground` | `#221E1B` | 主文字（暖墨） |
| `--muted` | `#EFECE6` | 次级底（骨架、分隔底） |
| `--muted-foreground` | `#8B857C` | 次级文字/占位 |
| `--primary` | `#A94A43` | 主色（绛红）：主按钮、激活态、关键强调 |
| `--primary-foreground` | `#FFFFFF` | 主色上的文字 |
| `--primary-soft` | `#F6E9E6` | 主色柔底：选中 chip、轻强调块 |
| `--accent` | `#A87B2F` | 黄铜金：会员卡、等级、余额类数字 |
| `--accent-soft` | `#F5EAD3` | 金色柔底 |
| `--success` | `#47795B` | 成功/完成 |
| `--success-soft` | `#E4EFE7` | |
| `--warning` | `#B4832A` | 待处理/提醒 |
| `--warning-soft` | `#F6EDDA` | |
| `--destructive` | `#B03A2E` | 取消/撤销/错误 |
| `--destructive-soft` | `#F7E5E1` | |
| `--border` | `#E7E2DA` | 描边/分隔线 |
| `--input` | `#FFFFFF` | 输入框底 |
| `--ring` | `rgba(169,74,67,.30)` | 焦点环 |

规则：
- 一个页面最多出现 primary + accent 两种彩色；状态色只用于状态语义（徽标/提示），不做装饰。
- 禁止纯黑 `#000`、纯灰 `#808080`、默认蓝紫。

### 2.2 字号（移动端 px / 行高）

| Token | 值 | 用途 |
|---|---|---|
| `--font-display` | 26/34 · 600 | 页面大标题、余额大数字 |
| `--font-title` | 20/28 · 600 | 区块标题、卡片主标题 |
| `--font-body` | 15/22 · 400 | 正文、列表主文字 |
| `--font-sub` | 13/18 · 400 | 辅助说明 |
| `--font-caption` | 11/16 · 500 | 徽标、时间戳、极轻注释 |

- 中文字体栈：`-apple-system, "PingFang SC", "HarmonyOS Sans SC", "Microsoft YaHei", sans-serif`
- 金额/时间/单号：`font-variant-numeric: tabular-nums`；金额两位小数省略 `.00`。

### 2.3 间距 / 圆角 / 描边 / 阴影

- 间距栅格 4pt：页面左右留白 `16`，卡片内边距 `16–20`，区块间距 `24`，相关元素间 `8–12`。
- 圆角：`--radius-sm: 8`（小件）、`--radius-md: 12`（按钮/输入框）、`--radius-lg: 16`（卡片/浮层）、`--radius-full: 999`（chip/徽标）。
- 卡片：1px `--border` 描边 + 双层柔影 `0 1px 2px rgba(34,30,27,.05), 0 6px 20px rgba(34,30,27,.06)`；禁止重黑影。
- 浮层（dialog/sheet）：`0 12px 40px rgba(34,30,27,.18)`。

### 2.4 动效

| 场景 | 规格 |
|---|---|
| 按压反馈 | `transform: scale(.97)` + 透明度 .92，120ms ease-out |
| 页面内容进场 | 淡入 + 8px 上移，200ms |
| 骨架屏 | 底色 `--muted` 与高光 `#F7F4EF` 交替，1.4s 循环 |
| 弹层 | 220ms ease-out 上滑/缩放；遮罩 rgba(34,30,27,.45) |
| 计时器/刷新 | 数据变化不做花哨动画，状态灯呼吸 2s |

缓动统一：`cubic-bezier(.22,.61,.36,1)`。

### 2.5 图标

- 线性图标体系：1.8px 描边、圆角端点、24 viewBox（shadcn/lucide 风格）。
- 双端同一套 path：H5 用内联 SVG 组件；weapp 用 SVG data-uri + CSS mask 以 `currentColor` 染色。
- **全端禁止 emoji 作为功能图标**（tab 栏、菜单、空态一律图标化）。

## 3. 组件体系（shadcn 命名 → 双端实现）

| 组件 | 变体/状态 | H5 | weapp |
|---|---|---|---|
| Button | primary / secondary / outline / ghost / destructive；loading、disabled、按压 | `ui/AppButton.vue` | `.btn` 类族 |
| Card | 默认 / 内嵌分割 | `ui/AppCard.vue` | `.card` |
| Cell | 列表行：图标+文字+值+chevron；整行可点 | `ui/AppCell.vue` | `.cell` |
| Chip | 单选筛选、状态标签 | `ui/AppChip.vue` | `.chip` |
| Badge | status：success/warning/destructive/muted/primary | `ui/AppBadge.vue` | `.badge` |
| Input / Textarea | 清晰 label、占位 `--muted-foreground`、焦点 ring | `ui/AppInput.vue` | `.input` |
| Dialog | 居中确认框：标题/正文/取消+主操作 | `ui/AppDialog.vue` | `.dialog` + wx showModal 样式页 |
| Sheet | 底部动作面板（改期/选择器容器） | `ui/AppSheet.vue` | `.sheet` |
| Toast | 顶部轻提示（成功/失败），替代原生 wx.showToast | `platform/notify` | `.toast` |
| Empty | 图标 + 主句 + 次句 + 可选 CTA | `ui/AppEmpty.vue` | `.empty` |
| Skeleton | 文本/卡片两型 | `ui/AppSkeleton.vue` | `.skeleton` |
| SegmentedTabs | 2–5 项分段选择 | `ui/AppTabs.vue` | `.seg` |
| StatTile | 我的页统计块（卡次数/预约数） | `ui/AppStat.vue` | `.stat` |
| QRBlock | 核销码展示：白底大留白圆角卡 + 会员号 + 状态灯 | 专属 | 专属 |

状态规则：每个可交互组件必须有 default / hover(可选) / active(按压) / disabled / loading 五态或声明"无此态"。

## 4. 页面模板

- **页头**：大标题（display）+ 可选副句；返回箭头 32px 命中区。
- **列表页**：页头 → 筛选 chips → 卡片列表（间距 12）→ 底部安全区留白 24。
- **表单页**：分组 Card（每组一个语义），主按钮吸底 + 安全区。
- **空态**：Empty 组件，禁用纯文字裸空态。
- **状态徽标映射**：待到店=warning、服务中=primary、已完成=success、已取消=muted、爽约=destructive。

## 5. 双端实现映射

- H5（apps/customer）：token 全部进 `src/style.css` `:root`（shadcn 方式）；组件在 `src/components/ui/`；图标 `ui/AppIcon.vue`。
- 小程序（apps/weapp）：token 进 `app.wxss` `page{}`（同名去 `--` 前缀变量在 wxss 中同样可用）；公共类在 `app.wxss`（.btn/.card/.cell/.chip/.badge/.input/.empty/.skeleton）；图标 `components/app-icon/`。
- 两端类名与变体命名保持一致，页面结构互为镜像，文案一致（同一份业务词汇：预约单、核销、扣次、待到店…）。

## 6. 核销码（QRBlock）业务规范（与 P2-2 修复绑定）

- **顾客永远只出示一个码**：`ANMO-MEMBER:<member_id>`（D21 协议不变），仅对持 ACTIVE 卡顾客展示。
- 今日预约以**文字列表**呈现在码下方（时间 + 服务名 + 状态），不做预约单码。
- 商家扫码后由商家端列出该会员今日预约进行选择（ScanRedeemDialog 已有该闭环）。
- 无卡顾客：锁定态展示说明 + 引导，不出现任何码。
