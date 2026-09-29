// about — 关于门店：门店信息（地址导航/电话拨号）+ 门店介绍文案，支持分享。
// 门店内容接口是顾客 JWT 资源（V1 口径），匿名从分享进来时只给静态说明 + 登录引导。
const { api } = require('../../utils/api')

const STATUS_TEXT = { FREE: '空闲中', SERVING: '服务中', BUSY: '忙碌中' }
// 状态点配色（BRAND-GUIDELINES §4）：空闲=success 服务中=primary 忙碌=warning
const STATUS_DOT = { FREE: 'success', SERVING: 'primary', BUSY: 'warning' }

Page({
  data: {
    shop: { address: '', phone: '', latitude: '', longitude: '' },
    heroTitle: '',
    heroBody: '',
    hours: '',
    status: '',
    statusClass: '',
    loggedIn: false,
  },

  onLoad() {
    // 首页已缓存的设置先顶上，避免分享落地时白屏
    const cached = getApp().globalData.settings
    if (cached) this.applySettings(cached)
    getApp().ready((loggedIn) => {
      this.setData({ loggedIn })
      if (loggedIn) this.load()
    })
  },

  applySettings(d) {
    this.setData({
      shop: {
        address: d.shop_address || '',
        phone: d.shop_phone || '',
        latitude: d.shop_latitude || '',
        longitude: d.shop_longitude || '',
      },
      heroTitle: d.home_title || '',
      heroBody: d.home_body || '',
      // 营业时间（对齐 H5 AboutPage）：两端都配置了才展示
      hours: d.open_time && d.close_time ? d.open_time + ' - ' + d.close_time : '',
    })
  },

  load() {
    api
      .publicSettings()
      .then((d) => {
        getApp().globalData.settings = d
        this.applySettings(d)
      })
      .catch(() => {})
    api
      .storeStatus()
      .then((s) =>
        this.setData({ status: STATUS_TEXT[s.status] || '', statusClass: STATUS_DOT[s.status] || '' })
      )
      .catch(() => this.setData({ status: '', statusClass: '' }))
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },

  onPullDownRefresh() {
    if (this.data.loggedIn) this.load()
    setTimeout(() => wx.stopPullDownRefresh(), 400)
  },

  onShareAppMessage() {
    return { title: '安摩 · 门店信息', path: '/pages/about/about?from=share' }
  },

  onShareTimeline() {
    return { title: '安摩 · 门店信息' }
  },
})
