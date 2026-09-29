// qrcode — 到店核销码（BRAND-GUIDELINES §6 / D21）：
// 顾客永远只出示一个码 ANMO-MEMBER:<member_id>（仅持 ACTIVE 卡顾客展示）。
// 今日预约以文字列表呈现在码下方，商家扫码后自选预约结算；
// 不再生成 ANMO-APT 预约单码（历史多码 bug 的根源，协议已收敛）。
const { api } = require('../../utils/api')
const fmt = require('../../utils/format')
const qrcode = require('../../utils/qrcode')

const QR_SIZE = 220 // 逻辑 px（canvas 实际按 dpr 放大）

Page({
  data: {
    name: '',
    memberNo: '',
    memberId: '',
    hasActiveCard: false,
    nowText: '',
    todayApts: [], // {id, time, statusText, statusCls, service}
    ready: false,
  },

  onLoad() {
    this.tick()
    this.timer = setInterval(() => this.tick(), 1000)
  },

  onShow() {
    this.keepScreenOn(true)
    // 登录态可能在离开期间变化（登录后返回本页），每次显示都重新判定再拉取
    getApp().ready((loggedIn) => {
      if (loggedIn) {
        this.load()
        return
      }
      this.needLogin()
    })
  },

  onHide() {
    this.keepScreenOn(false)
  },

  onUnload() {
    if (this.timer) clearInterval(this.timer)
    this.keepScreenOn(false)
  },

  tick() {
    const d = new Date()
    this.setData({ nowText: fmt.pad(d.getHours()) + ':' + fmt.pad(d.getMinutes()) })
  },

  keepScreenOn(on) {
    wx.setKeepScreenOn({ keepScreenOn: on })
  },

  needLogin() {
    this.setData({ ready: false, hasActiveCard: false, todayApts: [] })
    wx.navigateTo({ url: '/pages/login/login' })
  },

  load() {
    Promise.all([api.myProfile(), api.myCards(), api.myAppointments()])
      .then(([profile, cards, apts]) => {
        const m = profile.member
        const today = fmt.todayStr()
        const list = (apts || [])
          .filter(
            (a) =>
              (a.status === 'WAITING' || a.status === 'IN_SERVICE') &&
              fmt.aptDate(a) === today,
          )
          .sort((a, b) => (a.scheduled_start < b.scheduled_start ? -1 : 1))
          .map((a) => ({
            id: a.id,
            time: fmt.aptTime(a),
            statusText: fmt.statusText(a.status),
            statusCls: a.status === 'IN_SERVICE' ? 'primary' : 'warning',
            // 列表项已内嵌 services 快照（与 H5 同口径），无需逐单拉详情
            service: (a.services || [])
              .map((s) => s.service_name_snapshot)
              .filter(Boolean)
              .join(' · '),
          }))
        this.setData(
          {
            name: m.name || '未设置昵称',
            memberNo: m.member_no,
            memberId: m.id,
            // D21：无有效卡不出会员码
            hasActiveCard: (cards || []).some((c) => c.status === 'ACTIVE'),
            todayApts: list,
            ready: true,
          },
          // 画布在 wx:if 里：必须等这一帧渲染完再取节点，否则二维码画不出来
          () => this.renderMemberQR(),
        )
      })
      .catch((e) => {
        if (e.needLogin) {
          this.needLogin()
          return
        }
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  renderMemberQR() {
    if (!this.data.hasActiveCard || !this.data.memberId) return
    this.drawQR('qr-member', 'ANMO-MEMBER:' + this.data.memberId)
  },

  drawQR(canvasId, text) {
    wx
      .createSelectorQuery()
      .select('#' + canvasId)
      .fields({ node: true, size: true })
      .exec((res) => {
        const info = res && res[0]
        if (!info || !info.node) return
        const canvas = info.node
        const dpr = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()).pixelRatio || 2
        canvas.width = QR_SIZE * dpr
        canvas.height = QR_SIZE * dpr
        const ctx = canvas.getContext('2d')
        ctx.scale(dpr, dpr)

        let qr
        try {
          qr = qrcode(0, 'M')
          qr.addData(text)
          qr.make()
        } catch (e) {
          wx.showToast({ title: '二维码生成失败', icon: 'none' })
          return
        }
        const n = qr.getModuleCount()
        const cell = Math.floor((QR_SIZE - 2 * 8) / (n + 8)) || 2
        const offset = Math.floor((QR_SIZE - cell * n) / 2)
        ctx.fillStyle = '#ffffff'
        ctx.fillRect(0, 0, QR_SIZE, QR_SIZE)
        ctx.fillStyle = '#221E1B'
        for (let r = 0; r < n; r++) {
          for (let c = 0; c < n; c++) {
            if (qr.isDark(r, c)) {
              ctx.fillRect(offset + c * cell, offset + r * cell, cell, cell)
            }
          }
        }
      })
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },
  onShareAppMessage() {
    return { title: '安摩 · 到店按摩', path: '/pages/home/home?from=share' }
  },
})
