// login — 微信一键登录（V2.2，D25 修订）：进入页面自动尝试 wx.login →
// 后端直建号/直发 Token；失败给出重试按钮。无表单。
// 手机号撞号的认领与 H5 密码设置都在「我的」页完成（plan §二.3/二.5）。
const auth = require('../../utils/auth')

Page({
  data: {
    trying: false,
    failed: false,
  },

  onLoad(query) {
    this.back = query.back || ''
    // 启动那次静默登录若已失败（网络等），这里再补一次自动尝试
    this.tryWx()
  },

  tryWx() {
    if (this.data.trying) return
    this.setData({ trying: true, failed: false })
    auth.wxSilentLogin().then((r) => {
      if (r.token) {
        getApp().markAuth(true)
        this.done()
        return
      }
      this.setData({ trying: false, failed: true })
    })
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
