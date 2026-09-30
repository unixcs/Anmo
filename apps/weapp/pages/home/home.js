// home — 首页：门店动态状态（§17 三态不落库）+ 动态文案 + 公告 + 服务目录 + 门店卡。
const { api } = require('../../utils/api')
const auth = require('../../utils/auth')
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
        // 游客分支（V2.2 第七批修复）：服务目录已对游客开放（第五批口径，与 services 页/H5 一致），
        // 首页服务推荐照常展示；公告/门店卡/快捷入口等顾客资源仍等登录（保留登录引导卡）
        this.setData({ loading: false })
        this.loadGuestCatalog()
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
        // 首登资料完善提醒（V2.2 R4）：settings 与资料都在手后才判，避免与加载竞态
        this.maybeFirstLoginPrompt()
        if (done) done()
      }
    }
    this.loadStatus(finish)
    this.loadSettings(finish)
    this.loadHome(finish)
  },

  // 首登资料完善提醒（V2.2 R4）：后台开关开启且资料不全时进首页提示一次；
  // 展示即计数（置本地标记），无论用户去完善还是暂不，之后不再弹；已完善不弹。
  maybeFirstLoginPrompt() {
    const s = getApp().globalData.settings || {}
    if (s.profile_first_login_prompt !== '1') return
    if (wx.getStorageSync('anmo_profile_first_prompted')) return
    const check = (profile) => {
      if (!profile || fmt.profileProgress(profile).pct >= 100) return
      wx.setStorageSync('anmo_profile_first_prompted', 1)
      wx.showModal({
        title: '完善资料',
        content: '完善称呼与手机号，方便预约联系',
        confirmText: '去完善',
        cancelText: '暂不',
        confirmColor: '#a94a43',
        success: (r) => {
          if (!r.confirm) return
          getApp().globalData.pendingProfileEdit = true
          wx.switchTab({ url: '/pages/me/me' })
        },
      })
    }
    const cached = auth.cachedProfile()
    if (cached) check(cached)
    else api.myProfile().then((d) => check(d.member)).catch(() => {})
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
        // settings 晚于服务目录到达时重算一次展示列表（两者并行的竞态兜底）
        this.applyServices()
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
        this.applyCatalog(catalog)
        this.setData({ announcements, blocks: fmt.homeBlocks(home.blocks) })
      })
      .catch(() => {})
      .then(() => done && done())
  },

  // 目录 → 首页服务卡片映射（登录/游客两条链路共用）
  applyCatalog(catalog) {
    this.catalogServices = (catalog.services || [])
      .filter((s) => s.status === 'ACTIVE')
      .map((s) => ({
        id: s.id,
        name: s.name,
        desc: s.description,
        minutes: s.duration_minutes,
        price: fmt.yuan(s.default_price),
      }))
    this.applyServices()
  },

  // 游客首页服务推荐（V2.2 第七批）：/api/services 是公开资源，游客也看到在售项目
  loadGuestCatalog() {
    api
      .catalog()
      .then((catalog) => this.applyCatalog(catalog))
      .catch(() => {})
  },

  // 服务推荐展示列表（V2.1）：目录与 settings 都就绪后按 pickHomeServices 口径筛选；
  // 任一方未就绪时先以当前可得数据展示，另一方到达后在各自 then 里重算
  applyServices() {
    if (!this.catalogServices) return
    this.setData({ services: fmt.pickHomeServices(this.catalogServices, getApp().globalData.settings || {}) })
  },

  goBooking(e) {
    // booking 是 tab 页，switchTab 不能带参 → 经 globalData 传递预选服务
    getApp().globalData.pendingServiceId = e.currentTarget.dataset.id || ''
    wx.switchTab({ url: '/pages/booking/booking' })
  },

  goServices() {
    wx.switchTab({ url: '/pages/services/services' })
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
