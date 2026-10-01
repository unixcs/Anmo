// appointments — 我的预约：分区列表 + 取消 + 改期（底部抽屉，slot-picker 复用）。
// 筛选交给后端（status 参数），排序在前端：活跃预约按时间就近置顶，历史倒序。
const { api } = require('../../utils/api')
const fmt = require('../../utils/format')

const FILTERS = [
  { key: '', label: '全部' },
  { key: 'WAITING', label: '待到店' },
  { key: 'IN_SERVICE', label: '服务中' },
  { key: 'COMPLETED', label: '已完成' },
  { key: 'CANCELLED', label: '已取消' },
  { key: 'NO_SHOW', label: '未到店' },
]

const ACTIVE = { WAITING: 1, IN_SERVICE: 2 }

Page({
  data: {
    filters: FILTERS,
    filter: '',
    list: [],
    loading: true,
    failed: false,
    // 改期抽屉
    reschedule: null, // {id, when, service}
    days: [],
    dayIdx: 0,
    options: null,
    part: '',
    slot: '',
    busy: false,
  },

  onLoad() {
    this.setData({ days: this.buildDays() })
  },

  onShow() {
    getApp().ready((loggedIn) => {
      // 游客分支（V2.2 第七批）：明确登录引导，不冒充「网络不可用」
      if (!loggedIn) {
        this.setData({ loading: false, failed: false, list: [], guest: true })
        return
      }
      this.setData({ guest: false })
      this.load()
    })
  },

  onPullDownRefresh() {
    this.load(() => wx.stopPullDownRefresh())
  },

  buildDays() {
    const days = []
    const bj = fmt.bjNow()
    for (let i = 0; i < 30; i++) {
      const d = new Date(bj.getTime() + i * 86400000)
      const value = fmt.bjDateStr(d)
      days.push({
        value,
        label: i === 0 ? '今天' : i === 1 ? '明天' : fmt.weekdayLabel(value),
        sub: (d.getUTCMonth() + 1) + '/' + d.getUTCDate(),
      })
    }
    return days
  },

  setFilter(e) {
    this.setData({ filter: e.currentTarget.dataset.key })
    this.load()
  },

  retry() {
    this.load()
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login?back=appointments' })
  },

  load(done) {
    this.setData({ loading: true, failed: false })
    api
      .myAppointments(this.data.filter)
      .then((apts) => {
        const rows = (apts || []).sort(compare)
        // 列表端点已内嵌 services 快照（P2-2），无需逐单拉详情
        const list = rows.map((a) => {
          const svcs = (a.services || []).map((s) => s.service_name_snapshot).filter(Boolean)
          const first = (a.services && a.services[0]) || {}
          return {
            id: a.id,
            no: a.appointment_no,
            date: fmt.dateLabel((a.scheduled_start || '').slice(0, 10)),
            dayTag: dayTag(a),
            time: fmt.aptRange(a),
            status: a.status,
            statusText: fmt.statusText(a.status),
            statusClass: fmt.statusClass(a.status),
            service: svcs.join(' · '),
            minutes: first.duration_minutes_snapshot || 0,
            note: a.customer_note || '',
            canCancel: a.status === 'WAITING',
            canReschedule: a.status === 'WAITING',
          }
        })
        this.setData({ list, loading: false })
        if (done) done()
      })
      .catch((e) => {
        // 401（未登录/凭证过期，V2.2 第七批）：落游客引导，不冒充网络错误
        if (e && e.needLogin) {
          this.setData({ list: [], loading: false, failed: false, guest: true })
        } else {
          this.setData({ list: [], loading: false, failed: true })
        }
        if (done) done()
      })
  },

  cancel(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: '取消预约',
      content: '确定取消这个预约吗？取消后需要重新预约。',
      confirmColor: '#a94a43',
      success: (r) => {
        if (!r.confirm) return
        api
          .cancelAppointment(id)
          .then(() => {
            wx.showToast({ title: '已取消' })
            this.load()
          })
          .catch((err) => wx.showToast({ title: err.message, icon: 'none' }))
      },
    })
  },

  // ---- 改期抽屉 ----
  openReschedule(e) {
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find((a) => a.id === id) || {}
    this.setData({
      reschedule: { id, when: item.date + ' ' + item.time, service: item.service },
      dayIdx: 0,
      options: null,
      part: '',
      slot: '',
      days: this.buildDays(), // 跨零点后日期条要重算
    })
    this.loadOptions()
  },

  noop() {},

  closeReschedule() {
    if (this.data.busy) return
    this.setData({ reschedule: null })
  },

  pickDay(e) {
    this.setData({ dayIdx: Number(e.currentTarget.dataset.idx) })
    this.loadOptions()
  },

  loadOptions() {
    const day = this.data.days[this.data.dayIdx]
    this.setData({ options: null, part: '', slot: '', optsErr: false })
    api
      .bookingOptions(day.value)
      .then((o) => {
        // 快速切换日期时丢弃过期响应
        const cur = this.data.days[this.data.dayIdx]
        if (!cur || cur.value !== day.value) return
        this.setData({ options: fmt.trimPastSlots(o, day.value) })
      })
      .catch(() => {
        // F-错误态：改期抽屉里查询失败给出重试入口
        const cur = this.data.days[this.data.dayIdx]
        if (!cur || cur.value !== day.value) return
        this.setData({ optsErr: true })
      })
  },

  onPick(e) {
    this.setData({ part: e.detail.part, slot: e.detail.slot })
  },

  submitReschedule() {
    const { reschedule, days, dayIdx, part, slot, busy } = this.data
    if (busy || !reschedule) return
    if (!part) {
      wx.showToast({ title: '请选择上午或下午', icon: 'none' })
      return
    }
    const day = days[dayIdx]
    const target = slot ? { start_time: day.value + ' ' + slot } : { date: day.value, day_part: part }
    this.setData({ busy: true })
    api
      .rescheduleAppointment(reschedule.id, target)
      .then(() => {
        this.setData({ reschedule: null, busy: false })
        wx.showToast({ title: '已改期' })
        this.load()
      })
      .catch((e) => {
        this.setData({ busy: false })
        // 仅名额/闭店类冲突需要重拉时段；网络等其他错误保留用户已选状态
        if (['APT_SLOT_FULL', 'APT_HALFDAY_FULL', 'APT_CLOSED', 'APT_TOO_SOON'].indexOf(e.code) >= 0) {
          this.loadOptions()
        }
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  goBooking() {
    wx.switchTab({ url: '/pages/booking/booking' })
  },

  goQR() {
    wx.navigateTo({ url: '/pages/qrcode/qrcode' })
  },

  onShareAppMessage() {
    return { title: '安摩', path: '/pages/home/home?from=share' }
  },
})

// 活跃预约（待到店/服务中）按时间就近排前，其余按日期倒序
function compare(a, b) {
  const ra = ACTIVE[a.status] || 9
  const rb = ACTIVE[b.status] || 9
  if (ra !== rb) return ra - rb
  const sa = a.scheduled_start || ''
  const sb = b.scheduled_start || ''
  if (ra !== 9) return sa.localeCompare(sb)
  return sb.localeCompare(sa)
}

function dayTag(a) {
  const off = fmt.dayOffset(a.scheduled_start || '')
  if (off === 0) return '今天'
  if (off === 1) return '明天'
  if (off === null || off < 0) return ''
  return off < 7 ? off + ' 天后' : ''
}
