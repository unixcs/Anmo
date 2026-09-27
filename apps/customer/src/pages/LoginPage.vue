<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../core/api/endpoints'
import { signIn } from '../platform/auth/session'
import { notify } from '../platform/notify/toast'

const router = useRouter()
const route = useRoute()
const phone = ref('')
const code = ref('')
const sent = ref(false)
const busy = ref(false)
const countdown = ref(0)
let timer: number | undefined

function startCountdown(): void {
  countdown.value = 60
  timer = window.setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0 && timer !== undefined) {
      window.clearInterval(timer)
      timer = undefined
    }
  }, 1000)
}

async function sendCode(): Promise<void> {
  if (!/^1\d{10}$/.test(phone.value)) {
    notify('请输入正确的手机号')
    return
  }
  busy.value = true
  try {
    await api.sendSms(phone.value)
    sent.value = true
    startCountdown()
    notify('验证码已发送（演示环境固定为 123456）')
  } catch (e) {
    notify((e as Error).message)
  } finally {
    busy.value = false
  }
}

async function login(): Promise<void> {
  busy.value = true
  try {
    const res = await api.verifySms(phone.value, code.value)
    signIn(res.token)
    const redirect = route.query.redirect as string | undefined
    router.replace(redirect ?? '/me')
  } catch (e) {
    notify((e as Error).message)
  } finally {
    busy.value = false
  }
}

onUnmounted(() => {
  if (timer !== undefined) window.clearInterval(timer)
})
</script>

<template>
  <div class="page login">
    <h1>登录</h1>
    <p class="tip">手机号验证码登录，未注册将自动创建会员</p>
    <input v-model="phone" type="tel" maxlength="11" inputmode="numeric" placeholder="手机号" />
    <div class="code-row">
      <input v-model="code" type="text" maxlength="6" inputmode="numeric" placeholder="验证码" />
      <button class="send" :disabled="busy || countdown > 0" @click="sendCode">
        {{ countdown > 0 ? `${countdown}s` : sent ? '重新发送' : '发送验证码' }}
      </button>
    </div>
    <button class="primary" :disabled="busy || !sent" @click="login">登 录</button>
  </div>
</template>

<style scoped>
.login { max-width: 420px; margin: 0 auto; padding: 24px 20px; display: flex; flex-direction: column; gap: 14px; }
.tip { color: var(--muted-foreground); font-size: 13px; }
input { width: 100%; height: 44px; box-sizing: border-box; border: 1px solid var(--border); border-radius: 8px; padding: 0 12px; font-size: 16px; background: var(--card); }
.code-row { display: flex; gap: 10px; }
.code-row input { flex: 1; min-width: 0; }
.send { flex-shrink: 0; min-width: 104px; height: 44px; padding: 0 14px; }
button { border: none; border-radius: 8px; padding: 0 16px; font-size: 15px; background: var(--border); }
button:disabled { opacity: .55; }
.primary { background: var(--primary); color: var(--card); height: 46px; font-size: 17px; }
</style>
