// login — 短信表单立即可用；微信静默登录后台尝试（plan §11）：
// 拿到 token 直接进入；拿到 bind_ticket 则登录后自动绑定。wx.login 挂起也不阻塞。
const { api } = require('../../utils/api')
const auth = require('../../utils/auth')

let smsTimer = null

Page({
  data: {
    bindTicket: '',
    wxTried: false,
    phone: '',
    code: '',
    counting: 0,
    busy: false,
  },

  onLoad(query) {
    this.back = query.back || ''
    const g = getApp().globalData
    // 静默登录由 app 启动时统一发起：这里只接它的结果（有 ticket 就等短信登录后自动绑定）。
    // 启动那次没拿到票据（网络失败等）才补试一次，避免重复请求。
    if (g.bindTicket) {
      this.setData({ bindTicket: g.bindTicket, wxTried: true })
    } else if (g.session) {
      g.session.then(() => this.setData({ bindTicket: g.bindTicket, wxTried: true }))
    } else {
      this.tryWx()
    }
  },

  onUnload() {
    if (smsTimer) clearInterval(smsTimer)
  },

  tryWx() {
    auth.wxSilentLogin().then((r) => {
      if (r.token) {
        getApp().markAuth(true)
        this.done()
        return
      }
      // 未绑定：记住 ticket，短信登录后自动绑定；拿不到也不阻塞表单
      if (r.bindTicket) {
        getApp().globalData.bindTicket = r.bindTicket
        this.setData({ bindTicket: r.bindTicket, wxTried: true })
      } else {
        this.setData({ wxTried: true })
      }
    })
  },

  onPhone(e) {
    this.setData({ phone: e.detail.value })
  },

  onCode(e) {
    this.setData({ code: e.detail.value })
  },

  sendSms() {
    const { phone, counting, busy } = this.data
    if (counting > 0 || busy) return
    if (!/^1\d{10}$/.test(phone)) {
      wx.showToast({ title: '手机号格式不正确', icon: 'none' })
      return
    }
    this.setData({ busy: true })
    api
      .sendSms(phone)
      .then(() => {
        wx.showToast({ title: '验证码已发送', icon: 'none' })
        this.setData({ counting: 60 })
        smsTimer = setInterval(() => {
          const n = this.data.counting - 1
          this.setData({ counting: n })
          if (n <= 0 && smsTimer) clearInterval(smsTimer)
        }, 1000)
      })
      .catch((e) => wx.showToast({ title: e.message, icon: 'none' }))
      .then(() => this.setData({ busy: false }))
  },

  submit() {
    const { phone, code, bindTicket, busy } = this.data
    if (busy) return
    if (!/^1\d{10}$/.test(phone) || !code) {
      wx.showToast({ title: '请填写手机号和验证码', icon: 'none' })
      return
    }
    this.setData({ busy: true })
    auth
      .smsLogin(phone, code, bindTicket)
      .then(() => {
        getApp().markAuth(true)
        this.done()
      })
      .catch((e) => wx.showToast({ title: e.message, icon: 'none' }))
      .then(() => this.setData({ busy: false }))
  },

  done() {
    if (this.back === 'booking') {
      wx.navigateBack({ fail: () => wx.switchTab({ url: '/pages/booking/booking' }) })
      return
    }
    wx.navigateBack({ fail: () => wx.switchTab({ url: '/pages/me/me' }) })
  },
  onShareAppMessage() {
    return { title: '安摩', path: '/pages/home/home?from=share' }
  },
})
