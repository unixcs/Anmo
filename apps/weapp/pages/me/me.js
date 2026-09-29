// me — 个人中心：登录态、资料、各功能入口。
const { api } = require('../../utils/api')
const auth = require('../../utils/auth')
const fmt = require('../../utils/format')

Page({
  data: {
    loggedIn: false,
    member: null, // {name, member_no, phone}
    avatar: '客',
    stats: { cards: 0, remaining: 0, upcoming: 0 },
    progress: { pct: 100, missing: [] }, // 资料完善度（V2.2 R4），未加载/已完善不渲染提示条
    progressText: '',
    editing: false,
    name: '',
    gender: '',
    phoneInput: '', // 仅当 member.phone 为空时显示（微信会员一次性补手机号）
    claim: { show: false, phone: '', password: '', busy: false }, // 撞号认领弹层（V2.2 R4）
    pwd: { show: false, password: '', busy: false }, // 「设置 H5 密码」弹层（V2.2 R5）
  },

  onShow() {
    // 先出缓存资料再拉最新，切 tab 不闪空白
    const cached = auth.cachedProfile()
    if (cached) this.setMember(cached)
    // 预约页「去完善」/首登提醒跳转过来：自动展开资料编辑（V2.2 R4）
    const app = getApp()
    if (app.globalData.pendingProfileEdit) {
      app.globalData.pendingProfileEdit = ''
      if (cached) this.startEdit()
    }
    app.ready((loggedIn) => {
      if (loggedIn) {
        this.refresh()
        this.loadStats()
        return
      }
      this.setData({ loggedIn: false, member: null, avatar: '客', stats: { cards: 0, remaining: 0, upcoming: 0 } })
    })
  },

  setMember(member) {
    // 头像字与展示名同源（未设置昵称 → "未"），与 H5 同口径
    // 资料完善度文案（V2.2 R4，与 H5 MePage 同文案）：按缺失字段变化
    const prog = fmt.profileProgress(member)
    const missing = prog.missing
    this.setData({
      loggedIn: true,
      member,
      avatar: (member && member.name ? member.name : '未设置昵称').slice(0, 1) || '客',
      progress: prog,
      progressText:
        missing.indexOf('name') >= 0 && missing.indexOf('phone') >= 0
          ? '完善称呼与手机号'
          : missing.indexOf('name') >= 0
            ? '完善称呼'
            : '完善手机号，方便预约联系',
    })
  },

  // 统计瓦片：有效卡数 / 剩余总次数 / 即将到店（拿不到就不显示数字，不阻塞资料）
  loadStats() {
    api
      .myCards()
      .then((cards) => {
        const list = cards || []
        this.setData({
          'stats.cards': list.filter((c) => c.status === 'ACTIVE').length,
          'stats.remaining': list.reduce(
            (s, c) => s + (c.status === 'ACTIVE' ? c.remaining_count || 0 : 0),
            0
          ),
        })
      })
      .catch(() => {})
    api
      .myAppointments('')
      .then((apts) => {
        // 即将到店 = 今天（北京）及以后的活跃预约（与 H5 同口径，过期遗留的 WAITING 不计）
        const today = fmt.todayStr()
        const upcoming = (apts || []).filter(
          (a) =>
            (a.status === 'WAITING' || a.status === 'IN_SERVICE') &&
            (a.scheduled_start || '').slice(0, 10) >= today
        ).length
        this.setData({ 'stats.upcoming': upcoming })
      })
      .catch(() => {})
  },

  refresh() {
    api
      .myProfile()
      .then((d) => {
        auth.saveProfile(d.member)
        this.setMember(d.member)
      })
      .catch((e) => {
        if (e.needLogin) getApp().markAuth(false)
      })
  },

  go(e) {
    wx.navigateTo({ url: e.currentTarget.dataset.url })
  },

  startEdit() {
    this.setData({
      editing: true,
      name: this.data.member ? this.data.member.name : '',
      gender: this.data.member ? this.data.member.gender : '',
      phoneInput: '',
    })
  },

  cancelEdit() {
    this.setData({ editing: false })
  },

  onName(e) {
    this.setData({ name: e.detail.value })
  },

  onPhoneInput(e) {
    this.setData({ phoneInput: e.detail.value })
  },

  pickGender(e) {
    this.setData({ gender: e.currentTarget.dataset.g })
  },

  saveProfile() {
    const { name, gender, phoneInput, member } = this.data
    const patch = { name, gender }
    // 手机号一次性设置（V2.2 R4）：仅当前为空时可填；撞号 → 弹认领弹层
    if (member && !member.phone && phoneInput) patch.phone = phoneInput.trim()
    api
      .updateProfile(patch)
      .then(() => {
        wx.showToast({ title: '已保存' })
        this.setData({ editing: false, phoneInput: '' })
        this.refresh()
      })
      .catch((e) => {
        if (e.code === 'MEMBER_PHONE_TAKEN') {
          // 该手机号已有 H5 账号：弹 claim 密码弹层，凭 H5 密码转绑老账号
          this.setData({
            editing: false,
            claim: { show: true, phone: patch.phone || '', password: '', busy: false },
          })
          return
        }
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  // ---- 撞号认领弹层（POST /api/auth/wx/claim，V2.2 R4）----
  closeClaim() {
    this.setData({ claim: { show: false, phone: '', password: '', busy: false } })
  },

  onClaimPhone(e) {
    this.setData({ 'claim.phone': e.detail.value })
  },

  onClaimPassword(e) {
    this.setData({ 'claim.password': e.detail.value })
  },

  confirmClaim() {
    const { phone, password, busy } = this.data.claim
    if (busy) return
    if (!/^1\d{10}$/.test(phone)) {
      wx.showToast({ title: '请填写正确的手机号', icon: 'none' })
      return
    }
    if (!password || password.length < 6) {
      wx.showToast({ title: '请填写该账号的 H5 密码（6 位以上）', icon: 'none' })
      return
    }
    this.setData({ 'claim.busy': true })
    api
      .wxClaim(phone, password)
      .then((d) => {
        // 成功：以返回 token 重置本地登录态（登录态切换到老账号），空壳已由后端删除
        return auth
          .adoptToken(d.token)
          .then(() => {
            getApp().markAuth(true)
            wx.showToast({ title: '已关联老账号', icon: 'none' })
            this.setData({ claim: { show: false, phone: '', password: '', busy: false } })
            this.refresh()
            this.loadStats()
          })
      })
      .catch((e) => {
        this.setData({ 'claim.busy': false })
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  // ---- 设置 H5 密码弹层（PUT /api/me/h5-password，V2.2 R5）----
  goH5Password() {
    const m = this.data.member
    if (!m || !m.phone) {
      // 无手机号时引导完善（后端也要求 phone 非空）
      wx.showToast({ title: '请先完善手机号', icon: 'none' })
      this.startEdit()
      return
    }
    this.setData({ pwd: { show: true, password: '', busy: false } })
  },

  closePwd() {
    this.setData({ pwd: { show: false, password: '', busy: false } })
  },

  onPwdInput(e) {
    this.setData({ 'pwd.password': e.detail.value })
  },

  confirmPwd() {
    const { password, busy } = this.data.pwd
    if (busy) return
    if (!password || password.length < 6) {
      wx.showToast({ title: '密码需 6~64 位', icon: 'none' })
      return
    }
    this.setData({ 'pwd.busy': true })
    api
      .setH5Password(password)
      .then(() => {
        wx.showToast({ title: '已设置，可用于 H5 登录', icon: 'none' })
        this.setData({ pwd: { show: false, password: '', busy: false } })
      })
      .catch((e) => {
        this.setData({ 'pwd.busy': false })
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  noop() {},

  logout() {
    wx.showModal({
      title: '退出登录',
      content: '退出后需要重新登录才能预约',
      confirmColor: '#a94a43',
      success: (r) => {
        if (!r.confirm) return
        auth.logout()
        getApp().markAuth(false)
        this.setData({ loggedIn: false, member: null, avatar: '客', stats: { cards: 0, remaining: 0, upcoming: 0 } })
      },
    })
  },

  onShareAppMessage() {
    return { title: '安摩', path: '/pages/home/home?from=share' }
  },
})
