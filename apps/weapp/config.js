// 后端地址（一套后端多个前端）。环境自适应，切换不需要手改：
//  - 开发者工具模拟器（跑在 Windows）: http://127.0.0.1:8080
//    WSL2 Mirrored 网络下 Windows 侧只有 loopback 能进 WSL，局域网 IP 走不通。
//  - 真机预览 / 真机调试（跑在手机）:  http://<电脑局域网IP>:8080
//    手机上的 127.0.0.1 是手机自己，必须用电脑 IP；且需在 Windows 侧放行 LAN→WSL 入站
//    （Hyper-V 防火墙，见 scripts/allow-wsl-lan.ps1）。
//  - 正式发布: https://<已备案域名>，并在小程序后台配置 request 合法域名。
const LAN_URL = 'http://192.168.2.224:8080'
const LOOPBACK_URL = 'http://127.0.0.1:8080'

function resolveBaseUrl() {
  try {
    const platform = wx.getDeviceInfo ? wx.getDeviceInfo().platform : wx.getSystemInfoSync().platform
    return platform === 'devtools' ? LOOPBACK_URL : LAN_URL
  } catch (e) {
    return LAN_URL
  }
}

module.exports = { BASE_URL: resolveBaseUrl(), LAN_URL, LOOPBACK_URL }
