// config.js 单测（node:test，零依赖）：BASE_URL 解析的覆盖/降级路径。
// 每个 case 前 delete require cache 并重设 global.wx stub 再 require('../config.js')，
// 因为 config.js 在模块加载时同步执行 resolveBaseUrl() 求值 BASE_URL。
const { test } = require('node:test')
const assert = require('node:assert')
const path = require('node:path')

const CONFIG_PATH = path.join(__dirname, '..', 'config.js')
const PROD_URL = 'https://anmo.oiob.cn'
const LAN_URL = 'http://192.168.2.224:8080'
const LOOPBACK_URL = 'http://127.0.0.1:8080'

// ---- stub 工具：按用例选择性缺省方法以测降级路径 ----

function loadConfig(wxStub) {
  delete require.cache[require.resolve(CONFIG_PATH)]
  if (wxStub === undefined) {
    delete global.wx
  } else {
    global.wx = wxStub
  }
  return require(CONFIG_PATH)
}

function storageStub(store) {
  return {
    getStorageSync: (key) => (key in store ? store[key] : ''),
    getDeviceInfo: () => ({ platform: 'devtools' }),
    getSystemInfoSync: () => ({ platform: 'devtools' }),
  }
}

test('① storage 覆盖合法 http(s) 值优先生效', () => {
  const override = 'http://192.168.9.9:8080'
  const cfg = loadConfig(storageStub({ 'anmo.base_url': override }))
  assert.strictEqual(cfg.BASE_URL, override)
})

test('② 覆盖值非 http(s) 被忽略，回落平台默认', () => {
  const cfg = loadConfig(storageStub({ 'anmo.base_url': 'abc' }))
  assert.strictEqual(cfg.BASE_URL, LOOPBACK_URL) // stub platform='devtools'
})

test('②b 覆盖值含空白（非完整合法 URL）被忽略，回落平台默认', () => {
  const cfg = loadConfig(storageStub({ 'anmo.base_url': 'http://a b' }))
  assert.strictEqual(cfg.BASE_URL, LOOPBACK_URL)
})

test('③ 无覆盖 + platform=devtools → LOOPBACK_URL', () => {
  const cfg = loadConfig(storageStub({}))
  assert.strictEqual(cfg.BASE_URL, LOOPBACK_URL)
})

test('④ 无覆盖 + platform=ios → PROD_URL（仅 getDeviceInfo，缺省 getSystemInfoSync）', () => {
  const cfg = loadConfig({
    getStorageSync: () => '',
    getDeviceInfo: () => ({ platform: 'ios' }),
  })
  assert.strictEqual(cfg.BASE_URL, PROD_URL)
})

test('④b 真机 + storage 覆盖 LAN_URL → 覆盖生效（本地真机联调路径）', () => {
  const cfg = loadConfig({
    getStorageSync: (key) => (key === 'anmo.base_url' ? LAN_URL : ''),
    getDeviceInfo: () => ({ platform: 'ios' }),
  })
  assert.strictEqual(cfg.BASE_URL, LAN_URL)
})

test('⑤ global.wx 未定义 → PROD_URL（try/catch 兜底）', () => {
  const cfg = loadConfig(undefined)
  assert.strictEqual(cfg.BASE_URL, PROD_URL)
})

test('⑥ 导出 VERSION（V2.2 第六批：about 页版本标识防呆）', () => {
  const cfg = loadConfig(storageStub({}))
  assert.match(cfg.VERSION, /^V\d+\.\d+\.\d+$/)
})
