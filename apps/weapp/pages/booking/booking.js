// booking — 预约页（D20 口径与顾客 H5 同一语义）：
// 服务必选 → 日期（30 天内）→ 上下午必选 → 可选具体时间（名额共享）→ 提交。
const { api } = require('../../utils/api')
const auth = require('../../utils/auth')
const fmt = require('../../utils/format')

const AHEAD_DAYS = 30

function buildDays() {
  const days = []
  const bj = fmt.bjNow()
  for (let i = 0; i < AHEAD_DAYS; i++) {
    const d = new Date(bj.getTime() + i * 86400000)
    const value = fmt.bjDateStr(d)
    days.push({
      value,
      label: i === 0 ? '今天' : i === 1 ? '明天' : fmt.weekdayLabel(value),
      sub: (d.getUTCMonth() + 1) + '/' + d.getUTCDate(),
    })
  }
  return days
}

Page({
  data: {
    days: buildDays(),
    dayIdx: 0,
    services: [],
    svcIdx: -1,
    svcErr: false,
    options: null,
    part: '',
    slot: '',
    note: '',
    busy: false,
    profileSheet: false, // 资料完善半屏提示（V2.2 R4）
    success: null, // {no, name, date, time, fuzzy}
    shop: { address: '', phone: '', latitude: '', longitude: '' },
    loggedIn: true,
  },

  onShow() {
    const app = getApp()
    app.ready((loggedIn) => {
      this.setData({ loggedIn })
      if (!loggedIn) return
      // 跨零点重算日期条（页面可能在前台挂过夜）
      const days = this.data.days
      if (!days.length || days[0].value !== fmt.todayStr()) {
        this.setData({ days: buildDays(), dayIdx: 0, options: null, part: '', slot: '' })
      }
      // 首页"选服务去预约"经 globalData 传参（tab 页不能带参跳转）
      const prefer = app.globalData.pendingServiceId || ''
      app.globalData.pendingServiceId = ''
      if (prefer || !this.data.services.length) this.loadServices(prefer)
      if (!this.data.options) this.loadOptions()
    })
  },

  loadServices(prefer) {
    api
      .catalog()
      .then((d) => {
        const list = (d.services || []).filter((s) => s.status === 'ACTIVE')
        // 选中态按 id 对齐：目录刷新（增删/调序）后不串位；已选服务被下架则取消选中
        const cur = this.data.services[this.data.svcIdx]
        const want = prefer || (cur ? cur.id : '')
        const idx = want ? list.findIndex((s) => s.id === want) : -1
        this.setData({
          services: list.map((s) => ({
            id: s.id,
            name: s.name,
            minutes: s.duration_minutes,
            price: fmt.yuan(s.default_price),
          })),
          svcIdx: idx,
          svcErr: false,
        })
      })
      .catch(() => {
        // 加载失败不能伪装成"暂未上架服务"：给出失败态 + 重试
        this.setData({ svcErr: true })
      })
  },

  // wxml 重试入口：bindtap 会把事件对象当参传入，单独包一层保证重新拉全量目录
  retryServices() {
    this.loadServices()
  },

  loadOptions() {
    this.ensureFreshDays()
    const day = this.data.days[this.data.dayIdx]
    this.setData({ options: null, part: '', slot: '' })
    api
      .bookingOptions(day.value)
      .then((o) => {
        // 快速切换日期时丢弃过期响应
        const cur = this.data.days[this.data.dayIdx]
        if (!cur || cur.value !== day.value) return
        fmt.trimPastSlots(o, day.value)
        this.setData({ options: o })
      })
      .catch(() => {})
  },

  // 页面前台跨零点：日期条过期则整体重算（防提交昨天的日期）
  ensureFreshDays() {
    const days = this.data.days
    if (days.length && days[0].value === fmt.todayStr()) return
    this.setData({ days: buildDays(), dayIdx: 0, options: null, part: '', slot: '' })
  },

  onPullDownRefresh() {
    if (!this.data.loggedIn) {
      wx.stopPullDownRefresh()
      return
    }
    this.loadServices()
    this.loadOptions()
    setTimeout(() => wx.stopPullDownRefresh(), 400)
  },

  pickDay(e) {
    this.setData({ dayIdx: Number(e.currentTarget.dataset.idx) })
    this.loadOptions()
  },

  pickService(e) {
    this.setData({ svcIdx: Number(e.currentTarget.dataset.idx) })
  },

  onPick(e) {
    this.setData({ part: e.detail.part, slot: e.detail.slot })
  },

  onNote(e) {
    this.setData({ note: e.detail.value })
  },

  loadShop() {
    api
      .publicSettings()
      .then((s) => {
        this.setData({
          shop: {
            address: s.shop_address || '',
            phone: s.shop_phone || '',
            latitude: s.shop_latitude || '',
            longitude: s.shop_longitude || '',
          },
        })
      })
      .catch(() => {})
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login?back=booking' })
  },

  goAbout() {
    wx.navigateTo({ url: '/pages/about/about' })
  },

  // ---- 资料完善半屏提示（V2.2 R4）----
  // 资料不全且未永久跳过时拦截首次提交；「先跳过」置本地标记后不再弹，其余关闭动作可再次提醒
  // 半屏面板挡泡（catchtap 空实现：阻止点面板误触遮罩关闭）
  noop() {},

  closeProfileSheet() {
    this.setData({ profileSheet: false })
  },

  goProfile() {
    // 已选信息随 tab 页实例保留，完善后返回预约页无需重选
    this.setData({ profileSheet: false })
    getApp().globalData.pendingProfileEdit = true
    wx.switchTab({ url: '/pages/me/me' })
  },

  skipProfile() {
    wx.setStorageSync('anmo_profile_booking_skipped', 1)
    this.setData({ profileSheet: false })
    this.submit() // 继续原提交流程
  },

  submit() {
    const { services, svcIdx, days, dayIdx, part, slot, note, busy, loggedIn } = this.data
    if (busy) return
    if (!loggedIn) {
      this.goLogin()
      return
    }
    const svc = services[svcIdx]
    if (!svc) {
      wx.showToast({ title: '请先选择服务', icon: 'none' })
      return
    }
    if (!part) {
      wx.showToast({ title: '请选择上午或下午', icon: 'none' })
      return
    }
    const day = days[dayIdx]
    // 前台跨零点后日期条已整体重算，旧选择不再可信
    if (!day || days[0].value !== fmt.todayStr()) {
      this.loadOptions()
      wx.showToast({ title: '请重新选择时间', icon: 'none' })
      return
    }
    // 资料完善半屏提示（V2.2 R4）：只提醒不拦截人——跳过一次后永不再弹
    const profile = auth.cachedProfile()
    if (
      profile &&
      fmt.profileProgress(profile).pct < 100 &&
      !wx.getStorageSync('anmo_profile_booking_skipped')
    ) {
      this.setData({ profileSheet: true })
      return
    }
    // 与 H5 同口径：选了槽 → start_time；只选半天 → date + day_part
    const target = slot ? { start_time: day.value + ' ' + slot } : { date: day.value, day_part: part }
    this.setData({ busy: true })
    api
      .createAppointment(svc.id, target, note.trim())
      .then((a) => {
        this.setData({
          busy: false,
          success: {
            no: (a && a.appointment_no) || '',
            name: svc.name,
            date: day.label + ' ' + day.sub,
            time: slot || (part === 'AM' ? '上午' : '下午'),
            fuzzy: !slot,
          },
          part: '',
          slot: '',
        })
        this.loadShop() // 成功页带门店信息卡（§19），拿不到不影响成功态
        this.loadOptions() // 本单已占名额，成功页背后的余量要刷新
      })
      .catch((e) => {
        this.setData({ busy: false })
        if (e.needLogin) {
          this.goLogin()
          return
        }
        // 名额可能刚被别人占掉：重拉一次，避免用户对着过期数据反复试
        if (e.status === 409) this.loadOptions()
        wx.showModal({ title: '预约失败', content: e.message, showCancel: false })
      })
  },

  doneHome() {
    wx.switchTab({ url: '/pages/home/home' })
  },

  doneApts() {
    wx.navigateTo({ url: '/pages/appointments/appointments' })
  },

  again() {
    // 回到表单：名额已变，重新查询
    this.setData({ success: null, note: '' })
    this.loadOptions()
  },
  onShareAppMessage() {
    return { title: '安摩 · 来做个推拿放松一下', path: '/pages/home/home?from=share' }
  },
})
