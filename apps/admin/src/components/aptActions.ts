// components — 预约状态操作的公共封装（确认弹窗 + API + 提示 + 刷新）
// 改期走独立的 RescheduleDialog（日期/时段控件），由页面级 @reschedule 事件触发。
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  startAppointment,
  completeAppointment,
  cancelAppointment,
  noShowAppointment,
} from '../core/api/admin'

export function useAptActions(refresh: () => void) {
  async function run(fn: () => Promise<unknown>, ok: string) {
    try {
      await fn()
      ElMessage.success(ok)
      refresh()
    } catch (e) {
      ElMessage.error(e instanceof Error ? e.message : '操作失败')
    }
  }

  async function withConfirm(message: string, fn: () => Promise<unknown>, ok: string) {
    try {
      await ElMessageBox.confirm(message, '确认操作', { type: 'warning' })
    } catch {
      return
    }
    await run(fn, ok)
  }

  async function withPrompt(
    title: string,
    placeholder: string,
    validate: (v: string) => string | null,
    fn: (v: string) => Promise<unknown>,
    ok: string,
  ) {
    let value = ''
    try {
      const res = await ElMessageBox.prompt(title, '确认操作', {
        inputPlaceholder: placeholder,
        inputValidator: (v: string) => validate(v.trim()) ?? true,
      })
      value = res.value.trim()
    } catch {
      return
    }
    await run(() => fn(value), ok)
  }

  return {
    startApt: (id: string) => run(() => startAppointment(id), '已开始服务'),
    completeApt: (id: string) =>
      withConfirm('确认完成该预约？完成后才能收款入账。', () => completeAppointment(id), '已完成'),
    cancelApt: (id: string) =>
      withPrompt('请输入取消原因', '如：顾客临时有事', () => null, (reason) => cancelAppointment(id, reason), '已取消'),
    noShowApt: (id: string) =>
      withConfirm('将该预约标记为顾客未到店？', () => noShowAppointment(id), '已标记未到店'),
  }
}
