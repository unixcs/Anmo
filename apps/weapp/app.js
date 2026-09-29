// app.js — Anmo 顾客端小程序（V2 原生）。业务全部走 Anmo Go 后端 /api。
const auth = require('./utils/auth')

App({
  globalData: {
    // 门店动态状态/设置由首页拉取，这里只放跨页共享的轻量缓存
    settings: null,
    loggedIn: false,
    pendingServiceId: '', // 首页→预约页的预选服务（tab 页不能带参跳转）
    bindTicket: '', // 静默登录拿到的未绑定票据，短信登录后自动绑定（D25）
    session: null, // 启动登录态 Promise
    sessionSettled: false,
    manualAuth: false, // 用户手动登录/退出过：静默登录结果不再覆盖登录态
  },

  onLaunch() {
    // 微信分享菜单（朋友圈分享需要主动开启）
    if (wx.showShareMenu) {
      wx.showShareMenu({ withShareTicket: false, menus: ['shareAppMessage', 'shareTimeline'] })
    }
    this.startSession()
  },

  // 静默登录：已绑定会员直接续期 Token；未绑定不强制打断，页面按需引导。
  startSession() {
    const g = this.globalData
    const p = auth.bootstrap().then((r) => {
      // 静默登录出结果前用户可能已经手动登录/退出，那种情况下以手动结果为准
      if (!g.manualAuth) g.loggedIn = r.loggedIn
      if (r.bindTicket && !g.bindTicket) g.bindTicket = r.bindTicket
      return r
    })
    p.then(() => {
      g.sessionSettled = true
    })
    g.session = p
    return p
  },

  // 登录成功 / 退出登录时调用：立即刷新登录态，后续 ready() 走同步分支。
  // bindTicket 不清：退出后重新登录时仍要靠它把微信 openid 绑回去（10 分钟窗口）。
  markAuth(loggedIn) {
    const g = this.globalData
    g.loggedIn = loggedIn
    g.manualAuth = true
    g.sessionSettled = true
  },

  // 页面在 onShow 里调这个：登录态判定必须在静默登录出结果之后，否则会先渲染成匿名态
  // 并触发一次无谓的重复拉取。已出结果则同步执行。
  ready(cb) {
    const g = this.globalData
    if (!g.session || g.sessionSettled) {
      cb(g.loggedIn)
      return
    }
    g.session.then(() => cb(g.loggedIn))
  },
})
