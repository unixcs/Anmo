// 后端地址（一套后端多个前端）。环境自适应，切换不需要手改：
//  - 开发者工具模拟器（跑在 Windows）: http://127.0.0.1:8080
//    WSL2 Mirrored 网络下 Windows 侧只有 loopback 能进 WSL，局域网 IP 走不通。
//  - 真机预览 / 真机调试（跑在手机）:  https://anmo.oiob.cn（Cloudflare Tunnel 生产域名，V2.2 第五批）。
//    本地真机联调（连电脑后端）用 storage 覆盖指向 LAN_URL：wx.setStorageSync('anmo.base_url', LAN_URL)。
//  - 手动覆盖: wx.setStorageSync('anmo.base_url', 'http://<IP>:8080') 优先级最高，值非法自动忽略（删除用 wx.removeStorageSync）。
const PROD_URL = 'https://anmo.oiob.cn'
const LAN_URL = 'http://192.168.2.224:8080'
const LOOPBACK_URL = 'http://127.0.0.1:8080'
const STORAGE_KEY = 'anmo.base_url'

function resolveBaseUrl() {
  try {
    const override = wx.getStorageSync ? wx.getStorageSync(STORAGE_KEY) : ''
    if (typeof override === 'string' && /^https?:\/\/\S+$/.test(override.trim())) {
      return override.trim()
    }
  } catch (e) { /* storage 不可用则回落平台默认 */ }
  try {
    const platform = wx.getDeviceInfo ? wx.getDeviceInfo().platform : wx.getSystemInfoSync().platform
    return platform === 'devtools' ? LOOPBACK_URL : PROD_URL
  } catch (e) {
    return PROD_URL
  }
}

module.exports = { BASE_URL: resolveBaseUrl(), PROD_URL, LAN_URL, LOOPBACK_URL, STORAGE_KEY }
