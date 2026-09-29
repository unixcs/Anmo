/**
 * core/utils/home-services — 首页服务推荐筛选（V2.1）。
 * 口径单一真源见任务 design.md 伪代码；weapp utils/format.js 为逐行同构实现。
 */
import type { ServiceItem } from '../models/models'

const ALLOWED_LIMITS = [2, 4, 6, 8]

function parseLimit(raw: string | undefined): number {
  const n = Number(raw)
  return ALLOWED_LIMITS.includes(n) ? n : 6
}

function parseIds(raw: string | undefined): string[] {
  if (!raw) return []
  try {
    const v: unknown = JSON.parse(raw)
    if (!Array.isArray(v)) return []
    return v.filter((x): x is string => typeof x === 'string')
  } catch {
    return []
  }
}

/**
 * 从 ACTIVE 服务目录里挑出首页推荐：ids 配置非空按其顺序展示（跳过已下架/删除），
 * 否则按后端默认排序全量；最后统一截取前 limit 张（不足则全展示）。
 * settings 拉取失败传 {} 即回落默认（limit 6、不过滤）。
 */
export function pickHomeServices(services: ServiceItem[], settings: Record<string, string>): ServiceItem[] {
  const limit = parseLimit(settings.home_service_limit)
  const ids = parseIds(settings.home_service_ids)
  let picked: ServiceItem[]
  if (ids.length > 0) {
    const byId = new Map(services.map((s) => [s.id, s]))
    picked = ids.map((id) => byId.get(id)).filter((s): s is ServiceItem => !!s)
  } else {
    picked = services
  }
  return picked.slice(0, limit)
}
