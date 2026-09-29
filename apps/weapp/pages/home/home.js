// home — 首页：门店动态状态（§17 三态不落库）+ 动态文案 + 公告 + 服务目录 + 门店卡。
const { api } = require('../../utils/api')
const fmt = require('../../utils/format')

const STATUS_TEXT = { FREE: '空闲中', SERVING: '服务中', BUSY: '忙碌中' }

Page({
  data: {
    heroTitle: '欢迎光临',
    heroBody: '',
    status: null, // { cls, text }
    openTime: '',
    closeTime: '',
    announcements: [],
    services: [],
    blocks: fmt.homeBlocks(null), // 渲染顺序，loadHome 后由后台编排覆盖
    shop: { address: '', phone: '', latitude: '', longitude: '' },
    loggedIn: false,
    loading: true,
  },

  onShow() {
    getApp().ready((loggedIn) => {
      this.setData({ loggedIn })
      // 与顾客 H5 同口径：内容接口都是顾客 JWT 资源，未登录只展示登录引导
      if (loggedIn) {
        this.loadAll()
        if (!this.statusTimer) {
          // 门店状态 60s 轮询（与 H5 同节奏）
          this.statusTimer = setInterval(() => this.loadStatus(), 60 * 1000)
        }
      } else {
        this.setData({ loading: false, services: [], announcements: [] })
      }
    })
  },

  onHide() {
    this.stopPolling()
  },

  onUnload() {
    this.stopPolling()
  },

  stopPolling() {
    if (this.statusTimer) {
      clearInterval(this.statusTimer)
      this.statusTimer = null
    }
  },

  onPullDownRefresh() {
    if (!this.data.loggedIn) {
      wx.stopPullDownRefresh()
      return
    }
    this.loadAll(() => wx.stopPullDownRefresh())
  },

  loadAll(done) {
    this.setData({ loading: true })
    let pending = 3
    const finish = () => {
      if (--pending === 0) {
        this.setData({ loading: false })
        if (done) done()
      }
    }
    this.loadStatus(finish)
    this.loadSettings(finish)
    this.loadHome(finish)
  },

  loadStatus(done) {
    api
      .storeStatus()
      .then((s) => {
        const freeAt = (s.free_at || '').slice(0, 5)
        const text =
          s.status === 'SERVING' && freeAt ? '服务中 · 预计 ' + freeAt + ' 空闲' : STATUS_TEXT[s.status] || s.status
        this.setData({
          status: {
            // 徽标类名对齐 BRAND-GUIDELINES §4（空闲=success / 服务中=warning / 忙碌=destructive）
            cls: s.status === 'FREE' ? 'success' : s.status === 'BUSY' ? 'destructive' : 'warning',
            text,
          },
          openTime: (s.open_time || '').slice(0, 5),
          closeTime: (s.close_time || '').slice(0, 5),
        })
      })
      .catch(() => this.setData({ status: null }))
      .then(() => done && done())
  },

  loadSettings(done) {
    api
      .publicSettings()
      .then((d) => {
        getApp().globalData.settings = d
        this.setData({
          heroTitle: d.home_title || '安摩 · 到店按摩',
          heroBody: d.home_body || '',
          shop: {
            address: d.shop_address || '',
            phone: d.shop_phone || '',
            latitude: d.shop_latitude || '',
            longitude: d.shop_longitude || '',
          },
        })
      })
      .catch(() => {})
      .then(() => done && done())
  },

  // 首页内容块（§31/D16）：公告 + 服务，顺序由后台 blocks 决定
  loadHome(done) {
    Promise.all([api.home(), api.catalog()])
      .then(([home, catalog]) => {
        const announcements = (home.announcements || []).map((a) =>
          a.content ? a.title + ' · ' + a.content : a.title,
        )
        const services = (catalog.services || [])
          .filter((s) => s.status === 'ACTIVE')
          .map((s) => ({
            id: s.id,
            name: s.name,
            desc: s.description,
            minutes: s.duration_minutes,
            price: fmt.yuan(s.default_price),
          }))
        this.setData({ announcements, services, blocks: fmt.homeBlocks(home.blocks) })
      })
      .catch(() => {})
      .then(() => done && done())
  },

  goBooking(e) {
    // booking 是 tab 页，switchTab 不能带参 → 经 globalData 传递预选服务
    getApp().globalData.pendingServiceId = e.currentTarget.dataset.id || ''
    wx.switchTab({ url: '/pages/booking/booking' })
  },

  goAbout() {
    wx.navigateTo({ url: '/pages/about/about' })
  },

  goApts() {
    wx.navigateTo({ url: '/pages/appointments/appointments' })
  },

  goCards() {
    wx.navigateTo({ url: '/pages/cards/cards' })
  },

  goQR() {
    wx.navigateTo({ url: '/pages/qrcode/qrcode' })
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },

  // ---- 微信分享闭环（D26）----
  onShareAppMessage() {
    return {
      title: this.data.heroTitle || '安摩 · 到店按摩',
      path: '/pages/home/home?from=share',
    }
  },

  onShareTimeline() {
    return { title: this.data.heroTitle || '安摩 · 到店按摩' }
  },
})
