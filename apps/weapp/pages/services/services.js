// services — 服务项目列表（镜像 H5 ServicesPage）：按分类分组展示，预约带预选服务进预约 Tab。
const { api } = require('../../utils/api')
const fmt = require('../../utils/format')

// 卡片行字段与首页服务推荐一致（名称/描述/时长/价格）
function toCard(s) {
  return {
    id: s.id,
    name: s.name,
    desc: s.description,
    minutes: s.duration_minutes,
    price: fmt.yuan(s.default_price),
  }
}

Page({
  data: {
    groups: [], // { id, name, items }；无分类的服务归"其他"组（id 为空串）
    loading: true,
  },

  onShow() {
    // catalog 已对游客开放（V2.2 第五批）：恒加载，无需等登录态
    this.load()
  },

  load(done) {
    if (!done) this.setData({ loading: true })
    api
      .catalog()
      .then((catalog) => {
        const cats = catalog.categories || []
        const services = (catalog.services || []).filter((s) => s.status === 'ACTIVE')
        // 分组口径与 H5 ServicesPage.byCategory 一致：先渲染全部分类，无分类的落"其他"组
        const groups = cats.map((c) => ({
          id: c.id,
          name: c.name,
          items: services.filter((s) => s.category_id === c.id).map(toCard),
        }))
        const others = services.filter((s) => !cats.some((c) => c.id === s.category_id))
        if (others.length) groups.push({ id: '', name: '其他', items: others.map(toCard) })
        this.setData({ groups, loading: false })
      })
      .catch(() => this.setData({ loading: false }))
      .then(() => done && done())
  },

  goBooking(e) {
    // booking 是 tab 页，switchTab 不能带参 → 经 globalData 传递预选服务（与首页同模式）
    getApp().globalData.pendingServiceId = e.currentTarget.dataset.id || ''
    wx.switchTab({ url: '/pages/booking/booking' })
  },

  onPullDownRefresh() {
    this.load(() => wx.stopPullDownRefresh())
  },

  // ---- 微信分享闭环（D26）----
  onShareAppMessage() {
    return { title: '安摩 · 服务项目', path: '/pages/services/services?from=share' }
  },

  onShareTimeline() {
    return { title: '安摩 · 服务项目' }
  },
})
