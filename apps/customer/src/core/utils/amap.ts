/**
 * core/utils/amap — 高德地图 URI 构造（V2.2 R3）。
 * 微信内置/普通浏览器不分支特判，统一标准 URI（callnative=1）。
 * 参数前置校验：坐标缺失/非法（经度∉[73,135] 或 纬度∉[3,53]，中国大陆范围）
 * 一律回落搜索页，关键词=完整地址文本，保障跳转 100% 可用。
 */

/** 经纬度合法（中国大陆范围）：数值化，经度∈[73,135] 且 纬度∈[3,53]。 */
export function isValidLngLat(lng?: string, lat?: string): boolean {
  const lo = Number((lng ?? '').trim())
  const la = Number((lat ?? '').trim())
  if (Number.isNaN(lo) || Number.isNaN(la)) return false
  return lo >= 73 && lo <= 135 && la >= 3 && la <= 53
}

/** 门店地址跳转 URI：有合法坐标 → marker（带 name）；否则 → search（关键词=地址全文）。 */
export function amapUri(address: string, lng?: string, lat?: string): string {
  if (isValidLngLat(lng, lat)) {
    const pos = `${(lng ?? '').trim()},${(lat ?? '').trim()}`
    return `https://uri.amap.com/marker?position=${pos}&name=${encodeURIComponent(address)}&src=anmo&callnative=1`
  }
  return amapSearchUri(address)
}

/** 高德搜索页 URI（无坐标兜底/兜底条「在高德搜索」用）。 */
export function amapSearchUri(address: string): string {
  return `https://uri.amap.com/search?keyword=${encodeURIComponent(address)}&src=anmo&callnative=1`
}
