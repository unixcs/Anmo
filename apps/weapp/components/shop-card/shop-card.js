// shop-card — 门店信息卡：地址点一下唤起系统地图导航，电话点一下直接拨号。
// 小程序里走官方 API（wx.openLocation / wx.makePhoneCall），不做任何地图 SDK。
Component({
  properties: {
    address: { type: String, value: '' },
    phone: { type: String, value: '' },
    latitude: { type: String, value: '' },
    longitude: { type: String, value: '' },
  },

  methods: {
    openMap() {
      const { address, latitude, longitude } = this.data
      if (!address) return
      const lat = parseFloat(latitude)
      const lng = parseFloat(longitude)
      if (!isNaN(lat) && !isNaN(lng) && lat !== 0) {
        wx.openLocation({ latitude: lat, longitude: lng, name: address, scale: 18 })
        return
      }
      // 无经纬度：复制地址，用户自行地图搜索（不做地图 SDK，歧义选不做）
      wx.setClipboardData({ data: address })
    },

    callPhone() {
      if (!this.data.phone) return
      wx.makePhoneCall({ phoneNumber: this.data.phone })
    },
  },
})
