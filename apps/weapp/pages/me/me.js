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
    editing: false,
    name: '',
    gender: '',
  },

  onShow() {
    // 先出缓存资料再拉最新，切 tab 不闪空白
    const cached = auth.cachedProfile()
    if (cached) this.setMember(cached)
    getApp().ready((loggedIn) => {
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
    this.setData({
      loggedIn: true,
      member,
      avatar: (member && member.name ? member.name : '未设置昵称').slice(0, 1) || '客',
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
    })
  },

  cancelEdit() {
    this.setData({ editing: false })
  },

  onName(e) {
    this.setData({ name: e.detail.value })
  },

  pickGender(e) {
    this.setData({ gender: e.currentTarget.dataset.g })
  },

  saveProfile() {
    const { name, gender } = this.data
    api
      .updateProfile({ name, gender })
      .then(() => {
        wx.showToast({ title: '已保存' })
        this.setData({ editing: false })
        this.refresh()
      })
      .catch((e) => wx.showToast({ title: e.message, icon: 'none' }))
  },

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
