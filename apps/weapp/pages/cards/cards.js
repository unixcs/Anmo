// cards — 我的会员卡（与顾客 H5 CardsPage 同构）：
// 金卡视觉选卡 → 下方「使用明细」。remaining_count 是缓存，card_transaction 是历史。
const { api } = require('../../utils/api')
const fmt = require('../../utils/format')

// 卡状态徽标（BRAND-GUIDELINES §4）：有效=success，用完=muted，过期/作废=destructive
const CARD_BADGE = {
  ACTIVE: { variant: 'success', label: '有效' },
  USED_UP: { variant: 'muted', label: '已用完' },
  EXPIRED: { variant: 'destructive', label: '已过期' },
  CANCELLED: { variant: 'destructive', label: '已作废' },
}

// 流水语义：进=success 出=destructive 调整=accent；ADJUSTMENT 符号随 quantity 正负
const TX_META = {
  ISSUE: { label: '开卡', sign: '+', cls: 'in', icon: 'ticket' },
  REDEEM: { label: '核销', sign: '−', cls: 'out', icon: 'scan' },
  REVERSAL: { label: '撤销核销', sign: '+', cls: 'in', icon: 'history' },
  ADJUSTMENT: { label: '调整', sign: '', cls: 'adj', icon: 'settings' },
}

Page({
  data: {
    cards: [],
    loading: true,
    loggedIn: false,
    pickedId: '',
    picked: null, // 选中的卡对象（用于明细头）
    txns: [],
    txLoading: false,
  },

  onShow() {
    getApp().ready((loggedIn) => {
      this.setData({ loggedIn })
      if (loggedIn) this.load()
      else this.setData({ loading: false, cards: [], pickedId: '', picked: null, txns: [] })
    })
  },

  onPullDownRefresh() {
    if (this.data.loggedIn) this.load(() => wx.stopPullDownRefresh())
    else wx.stopPullDownRefresh()
  },

  load(done) {
    this.setData({ loading: true })
    api
      .myCards()
      .then((cards) => {
        this.setData({
          loading: false,
          cards: (cards || []).map((c) => {
            const badge = CARD_BADGE[c.status] || { variant: 'muted', label: c.status }
            return {
              id: c.id,
              cardName: c.card_name || '会员卡',
              remaining: c.remaining_count,
              total: c.total_count,
              status: c.status,
              badge: badge.variant,
              statusText: badge.label,
              validity: fmt.cardValidity(c),
              low: c.remaining_count <= 2 && c.status === 'ACTIVE',
            }
          }),
          pickedId: '',
          picked: null,
          txns: [],
        })
        if (done) done()
      })
      .catch((e) => {
        this.setData({ loading: false })
        if (done) done()
        if (e.needLogin) {
          this.setData({ loggedIn: false })
          return
        }
        wx.showToast({ title: e.message, icon: 'none' })
      })
  },

  pick(e) {
    const id = e.currentTarget.dataset.id
    const card = this.data.cards.find((c) => c.id === id)
    if (!card) return
    if (this.data.pickedId === id) return // 已选中：明细常驻，不重复拉
    this.setData({ pickedId: id, picked: card, txns: [], txLoading: true })
    api
      .myCardTransactions(id)
      .then((txns) => {
        this.setData({
          txns: (txns || []).map((t) => {
            const m = TX_META[t.type] || { label: t.type, sign: '', cls: 'adj', icon: 'settings' }
            const sign = t.type === 'ADJUSTMENT' ? (t.quantity >= 0 ? '+' : '−') : m.sign
            return {
              id: t.id,
              icon: m.icon,
              cls: m.cls,
              label: m.label,
              sign: sign,
              qty: Math.abs(t.quantity || 0),
              remark: t.remark || '',
              at: (t.created_at || '').slice(0, 16).replace('T', ' '),
            }
          }),
        })
      })
      .catch((e2) => wx.showToast({ title: e2.message, icon: 'none' }))
      .then(() => this.setData({ txLoading: false }))
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },

  goBooking() {
    wx.switchTab({ url: '/pages/booking/booking' })
  },

  onShareAppMessage() {
    return { title: '安摩', path: '/pages/home/home?from=share' }
  },
})
