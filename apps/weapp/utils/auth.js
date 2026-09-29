// auth.js — 登录态：微信静默登录优先，短信兜底；两个前端共用同一个 member。
// 链路（plan §11 + D24/D25）：
//   wx.login(code) → POST /api/auth/wx/login
//     ├─ 已绑定 openid → 直接发顾客 Token
//     └─ 未绑定 → { needs_bind, bind_ticket } → 短信登录拿到 Token
//        → POST /api/auth/wx/bind { bind_ticket } 绑定（10 分钟内有效）
const { api } = require('./api')
const request = require('./request')

const PROFILE_KEY = 'anmo_member_profile'

// 微信静默登录（尽量无感）。成功返回 { token }，未绑定/未配置返回 { needSms, bindTicket }。
function wxSilentLogin() {
  return new Promise((resolve) => {
    wx.login({
      success(res) {
        if (!res.code) {
          resolve({ needSms: true })
          return
        }
        api
          .wxLogin(res.code)
          .then((d) => {
            if (d && d.needs_bind) {
              resolve({ needSms: true, bindTicket: d.bind_ticket })
              return
            }
            if (d && d.token) {
              request.setToken(d.token)
              resolve({ token: d.token })
              return
            }
            resolve({ needSms: true })
          })
          .catch(() => resolve({ needSms: true }))
      },
      fail() {
        resolve({ needSms: true }) // 无微信环境（如 devtools 未登录）→ 短信兜底
      },
    })
  })
}

// 短信登录（与 H5 完全同一端点）；带 bindTicket 时登录成功后立即绑定微信。
function smsLogin(phone, code, bindTicket) {
  return api.verifySms(phone, code).then((d) => {
    request.setToken(d.token)
    const bind = bindTicket
      ? api.wxBind(bindTicket).catch(() => {}) // 绑定失败不阻断登录
      : Promise.resolve()
    return bind.then(() => d)
  })
}

function cachedProfile() {
  return wx.getStorageSync(PROFILE_KEY) || null
}

function saveProfile(p) {
  wx.setStorageSync(PROFILE_KEY, p)
}

function logout() {
  request.setToken('')
  wx.removeStorageSync(PROFILE_KEY)
}

// 侦察登录态：有 Token 则拉 profile 验活，失效自动清 Token。
function ensureSession() {
  if (!request.getToken()) return Promise.resolve(false)
  return api
    .myProfile()
    .then((d) => {
      saveProfile(d.member)
      return true
    })
    .catch((e) => {
      if (e.needLogin) {
        request.setToken('')
        wx.removeStorageSync(PROFILE_KEY)
      }
      return false
    })
}

// 启动引导：有 Token 直接验活；无 Token 试一次微信静默登录——已绑定会员重开小程序
// 应当无感回到登录态；未绑定则把 bind_ticket 交给页面（短信登录后自动绑定）。
function bootstrap() {
  if (request.getToken()) return ensureSession().then((ok) => ({ loggedIn: ok, bindTicket: '' }))
  return wxSilentLogin().then((r) => {
    if (!r.token) return { loggedIn: false, bindTicket: r.bindTicket || '' }
    return ensureSession().then((ok) => ({ loggedIn: ok, bindTicket: '' }))
  })
}

module.exports = {
  wxSilentLogin,
  smsLogin,
  ensureSession,
  bootstrap,
  cachedProfile,
  saveProfile,
  logout,
}
