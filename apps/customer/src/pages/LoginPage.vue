<script setup lang="ts">
import { ref } from 'vue'
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

async function sendCode(): Promise<void> {
  if (!/^1\d{10}$/.test(phone.value)) {
    notify('请输入正确的手机号')
    return
  }
  busy.value = true
  try {
    await api.sendSms(phone.value)
    sent.value = true
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
</script>

<template>
  <div class="page login">
    <h1>登录</h1>
    <p class="tip">手机号验证码登录，未注册将自动创建会员</p>
    <input v-model="phone" type="tel" maxlength="11" placeholder="手机号" />
    <div class="code-row">
      <input v-model="code" type="text" maxlength="6" placeholder="验证码" />
      <button :disabled="busy || sent" @click="sendCode">{{ sent ? '已发送' : '发送验证码' }}</button>
    </div>
    <button class="primary" :disabled="busy || !sent" @click="login">登 录</button>
  </div>
</template>

<style scoped>
.login { max-width: 420px; margin: 0 auto; padding: 24px 20px; display: flex; flex-direction: column; gap: 14px; }
.tip { color: #999; font-size: 13px; }
input { height: 44px; border: 1px solid #ddd; border-radius: 8px; padding: 0 12px; font-size: 16px; }
.code-row { display: flex; gap: 10px; }
.code-row input { flex: 1; }
.code-row button { white-space: nowrap; }
button { border: none; border-radius: 8px; padding: 10px 16px; font-size: 15px; background: #eee; }
.primary { background: #c85f5f; color: #fff; height: 46px; font-size: 17px; }
</style>
