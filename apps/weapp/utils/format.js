// format.js — 展示格式化（与 H5 core/format 口径一致）。
function pad(n) {
  return String(n).padStart(2, '0')
}

// ---- 北京时间锚定（后端固定 Asia/Shanghai；设备时区不参与业务日期口径）----
const BJ_MS = 8 * 3600000

// 当前北京时间映射的偏移 Date：分量一律用 getUTC* 读取
function bjNow() {
  return new Date(Date.now() + BJ_MS)
}

function bjDateStr(d) {
  return d.getUTCFullYear() + '-' + pad(d.getUTCMonth() + 1) + '-' + pad(d.getUTCDate())
}

// 今天（北京）YYYY-MM-DD
function todayStr() {
  return bjDateStr(bjNow())
}

// 'YYYY-MM-DD' 的北京日历星期序号（0=周日），由日期串直接计算、不经时区往返
function weekdayOf(s) {
  const p = (s || '').split('-').map(Number)
  if (p.length !== 3 || p.some(isNaN)) return null
  return (Math.floor(Date.UTC(p[0], p[1] - 1, p[2]) / 86400000) + 4) % 7
}

const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

function weekdayLabel(s) {
  const wd = weekdayOf(s)
  return wd === null ? '' : WEEK[wd]
}

// 预约时间列：模糊半天显示"上午/下午"，具体时间显示 HH:MM–HH:MM。
function aptTime(a) {
  if (a.slot_type === 'HALF_DAY') return a.day_part === 'AM' ? '上午' : '下午'
  return (a.scheduled_start || '').slice(11, 16)
}

function aptRange(a) {
  if (a.slot_type === 'HALF_DAY') return a.day_part === 'AM' ? '上午（时间由店主安排）' : '下午（时间由店主安排）'
  return (a.scheduled_start || '').slice(11, 16) + '–' + (a.scheduled_end || '').slice(11, 16)
}

function aptDate(a) {
  return (a.scheduled_start || '').slice(0, 10)
}

function yuan(cents) {
  if (cents === null || cents === undefined) return '-'
  return '¥' + (cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)
}

// V1.x 状态机（2026-09-28）：WAITING → IN_SERVICE → COMPLETED；历史状态只读展示
// （状态词与 H5 AppStatusBadge 同口径）。
const STATUS_TEXT = {
  WAITING: '待到店',
  IN_SERVICE: '服务中',
  COMPLETED: '已完成',
  CANCELLED: '已取消',
  NO_SHOW: '未到店',
  PENDING_CONFIRM: '待确认(历史)',
  CONFIRMED: '已确认(历史)',
}

// 徽标类名对齐 BRAND-GUIDELINES §4（app.wxss .badge.{variant}）。
const STATUS_CLASS = {
  WAITING: 'warning',
  IN_SERVICE: 'primary',
  COMPLETED: 'success',
  CANCELLED: 'muted',
  NO_SHOW: 'destructive',
  PENDING_CONFIRM: 'muted',
  CONFIRMED: 'muted',
}

function statusText(s) {
  return STATUS_TEXT[s] || s
}

function statusClass(s) {
  return STATUS_CLASS[s] || 'x'
}

// 卡有效期（不出现 T00:00:00+08:00）。
function cardValidity(c) {
  if (!c.valid_until) return '长期有效'
  return c.valid_until.slice(0, 10) + ' 到期'
}

const CARD_STATUS = {
  ACTIVE: '有效',
  USED_UP: '已用完',
  EXPIRED: '已过期',
  CANCELLED: '已作废',
}

function cardStatusText(s) {
  return CARD_STATUS[s] || s
}

// "MM-DD 周X"（星期由日期串计算，不经时区往返）
function dateLabel(iso) {
  const day = (iso || '').slice(0, 10)
  const label = weekdayLabel(day)
  return label ? day.slice(5, 10) + ' ' + label : day
}

// 相对今天的天数差：0=今天，1=明天（列表分组展示用）
function dayOffset(iso) {
  const d = weekdayOf(iso.slice(0, 10))
  if (d === null) return null
  const today = todayStr()
  const diff = (Date.parse(iso.slice(0, 10) + 'T00:00:00+08:00') - Date.parse(today + 'T00:00:00+08:00')) / 86400000
  return isNaN(diff) ? null : Math.round(diff)
}

const TX_TEXT = {
  ISSUE: '开卡',
  REDEEM: '核销',
  ADJUSTMENT: '调整',
  REVERSAL: '撤销核销',
  EXPIRE: '过期',
}

function txText(t) {
  return TX_TEXT[t] || t
}

// 后端 booking-options 会列出当天全部时段槽（不按当前时刻裁剪）。
// 这里只裁掉"已经开始"的槽；提前量等业务门槛仍由后端判定。
// 整段半天都过去了则标记 ended，让半天选择一并禁用（模糊预约对已结束的半天无意义）。
// 时刻用北京时间（UTC+8）判定，与后端口径一致。
function trimPastSlots(options, date) {
  if (!options || !date) return options
  const bj = bjNow()
  if (bjDateStr(bj) !== date) return options
  const hm = pad(bj.getUTCHours()) + ':' + pad(bj.getUTCMinutes())
  const cut = (half) => {
    if (!half || half.closed || !half.slots || !half.slots.length) return half
    const keep = half.slots.filter((s) => s.time > hm)
    if (!keep.length) half.ended = true
    half.slots = keep
    return half
  }
  cut(options.am)
  cut(options.pm)
  return options
}

// 卡流水数量方向（与 H5 同口径）：REDEEM 存正数但语义是扣次；ADJUSTMENT 自带符号；其余（ISSUE/REVERSAL）为加次
function txQty(t) {
  if (t.type === 'ADJUSTMENT') return (t.quantity > 0 ? '+' : '') + t.quantity
  if (t.type === 'REDEEM') return '-' + Math.abs(t.quantity)
  return '+' + Math.abs(t.quantity)
}

// 首页内容块编排（§31/D16，与 H5 同口径）：后台 blocks 决定顺序，未知/暂不支持的类型忽略。
// banner 依赖外部图片域名（发布期才可用），本端不渲染；全部被过滤掉时回落默认顺序。
const HOME_BLOCK_TYPES = ['announcement', 'service_list']
function homeBlocks(blocks) {
  const wanted = (blocks || []).map((b) => b && b.type)
  const picked = HOME_BLOCK_TYPES.filter((t) => wanted.indexOf(t) >= 0)
  return picked.length ? picked : HOME_BLOCK_TYPES.slice()
}

// 首页服务推荐筛选（V2.1，与 H5 core/utils/home-services 逐行同口径）：
// limit ∈ {2,4,6,8} 否则 6；home_service_ids 配置非空时按其顺序取存在的服务
// （跳过已下架/删除），否则全量（后端已按 sort,created_at 排序）；
// 最后统一截取前 limit 张（不足则全展示）。settings 缺失/解析失败传 {} 即回落默认。
function pickHomeServices(services, settings) {
  const s = settings || {}
  const limitNum = Number(s.home_service_limit)
  const limit = [2, 4, 6, 8].indexOf(limitNum) >= 0 ? limitNum : 6
  let ids = []
  if (s.home_service_ids) {
    try {
      const v = JSON.parse(s.home_service_ids)
      if (Array.isArray(v)) ids = v.filter((x) => typeof x === 'string')
    } catch (e) {
      ids = []
    }
  }
  let picked
  if (ids.length > 0) {
    const byId = {}
    services.forEach((svc) => {
      byId[svc.id] = svc
    })
    picked = ids.map((id) => byId[id]).filter((svc) => !!svc)
  } else {
    picked = services
  }
  return picked.slice(0, limit)
}

// 资料完善度（V2.2 R4，与 H5 core/utils/profile 逐行同构）：
// name 与 phone 均非空 = 100%，填一个 = 50%，全空 = 0；missing 按序给出缺失字段。
function profileProgress(member) {
  const m = member || {}
  const missing = []
  if (!(m.name || '').trim()) missing.push('name')
  if (!(m.phone || '').trim()) missing.push('phone')
  return { pct: 100 - missing.length * 50, missing }
}

module.exports = {
  pad,
  bjNow,
  bjDateStr,
  todayStr,
  weekdayOf,
  weekdayLabel,
  aptTime,
  aptRange,
  aptDate,
  dateLabel,
  dayOffset,
  yuan,
  statusText,
  statusClass,
  cardValidity,
  cardStatusText,
  txText,
  trimPastSlots,
  homeBlocks,
  pickHomeServices,
  profileProgress,
  txQty,
}
