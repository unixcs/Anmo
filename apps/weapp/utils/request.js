// request.js — 统一请求层：Token 注入、包络解包、401 处理。
// 后端响应包络 { code?, msg?, data }（与顾客 H5 core/api/http.ts 同口径）。
const { BASE_URL } = require('../config')

const TOKEN_KEY = 'anmo_customer_token'

function getToken() {
  return wx.getStorageSync(TOKEN_KEY) || ''
}

function setToken(token) {
  if (token) wx.setStorageSync(TOKEN_KEY, token)
  else wx.removeStorageSync(TOKEN_KEY)
}

function request(method, path, data) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: BASE_URL + path,
      method,
      data,
      timeout: 15000,
      header: Object.assign(
        { 'Content-Type': 'application/json' },
        getToken() ? { Authorization: 'Bearer ' + getToken() } : {},
      ),
      success(res) {
        const body = res.data || {}
        if (res.statusCode >= 200 && res.statusCode < 300) {
          // http 层已解包包络：分页端点 data 本身就是数组
          resolve(body.data !== undefined ? body.data : body)
          return
        }
        if (res.statusCode === 401) {
          setToken('')
          const app = getApp()
          if (app && app.markAuth) app.markAuth(false) // 全局登录态同步失效，避免其他页仍显示已登录
          const err = new Error(body.msg || '请先登录')
          err.code = body.code || 'UNAUTHORIZED'
          err.status = 401
          err.needLogin = true
          reject(err)
          return
        }
        const err = new Error(body.msg || '请求失败(' + res.statusCode + ')')
        err.code = body.code || ''
        err.status = res.statusCode
        reject(err)
      },
      fail(e) {
        // 完整 errMsg 只进 console：toast 会截断长文案，看不到失败原因
        console.error('[anmo:request] ' + method + ' ' + BASE_URL + path, e)
        const err = new Error('连不上 ' + BASE_URL + '：' + ((e && e.errMsg) || '网络不可用'))
        err.code = 'NETWORK'
        err.cause = e
        reject(err)
      },
    })
  })
}

module.exports = {
  get: (path) => request('GET', path),
  post: (path, data) => request('POST', path, data),
  put: (path, data) => request('PUT', path, data),
  getToken,
  setToken,
}
