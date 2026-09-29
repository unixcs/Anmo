// auth.js — 登录态：微信登录一步到位（V2.2，D25 修订）。
// 链路（plan §11 + §二）：
//   wx.login(code) → POST /api/auth/wx/login
//     ├─ openid 已绑定 → 直接发顾客 Token
//     └─ 未绑定 → 同事务直建号（空壳）→ 直接发顾客 Token
//   手机号撞号时在「我的」补手机号 → 409 MEMBER_PHONE_TAKEN → claim 弹层
//   → POST /api/auth/wx/claim 凭 H5 密码转绑老账号（空壳删除、登录态切换）。
const { api } = require('./api')
const request = require('./request')

const PROFILE_KEY = 'anmo_member_profile'

// 微信登录（尽量无感）。成功返回 { token }；失败（无微信环境/网络/后端错）
// 返回 { ok: false }，由页面引导重试。
function wxSilentLogin() {
  return new Promise((resolve) => {
    wx.login({
      success(res) {
        if (!res.code) {
          resolve({ ok: false })
          return
        }
        api
          .wxLogin(res.code)
          .then((d) => {
            if (d && d.token) {
              request.setToken(d.token)
              resolve({ token: d.token })
              return
            }
            resolve({ ok: false })
          })
          .catch(() => resolve({ ok: false }))
      },
      fail() {
        resolve({ ok: false }) // 无微信环境（如 devtools 未登录）
      },
    })
  })
}

// claim 转绑成功后切换登录态到老账号：换 Token 并立刻验活拉新 profile。
function adoptToken(token) {
  request.setToken(token)
  return ensureSession()
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

// 启动引导：有 Token 直接验活；无 Token 试一次微信登录——首登直建号，
// 重开小程序应当无感回到登录态。
function bootstrap() {
  if (request.getToken()) return ensureSession().then((ok) => ({ loggedIn: ok }))
  return wxSilentLogin().then((r) => {
    if (!r.token) return { loggedIn: false }
    return ensureSession().then((ok) => ({ loggedIn: ok }))
  })
}

module.exports = {
  wxSilentLogin,
  adoptToken,
  ensureSession,
  bootstrap,
  cachedProfile,
  saveProfile,
  logout,
}
