/**
 * core/utils/profile — 资料完善度（V2.2 R4）。
 * 口径：name 与 phone 均非空 = 100%，填一个 = 50%，全空 = 0%。
 * weapp utils/format.js 的 profileProgress 为逐行同构实现，改动须两端同步。
 */
export type ProfileMissing = 'name' | 'phone'

export interface ProfileProgress {
  pct: 0 | 50 | 100
  missing: ProfileMissing[]
}

export function profileProgress(member: { name?: string | null; phone?: string | null } | null | undefined): ProfileProgress {
  const m = member || {}
  const missing: ProfileMissing[] = []
  if (!(m.name || '').trim()) missing.push('name')
  if (!(m.phone || '').trim()) missing.push('phone')
  return { pct: (100 - missing.length * 50) as 0 | 50 | 100, missing }
}
