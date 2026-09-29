<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../core/api/endpoints'
import { signIn } from '../platform/auth/session'
import { notify } from '../platform/notify/toast'
import AppButton from '../components/ui/AppButton.vue'

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
  <div class="login">
    <div class="brand">
      <span class="seal">安摩</span>
      <h1>欢迎回来</h1>
      <p>手机号验证码登录，未注册将自动创建会员</p>
    </div>

    <div class="form">
      <div class="field">
        <label for="phone">手机号</label>
        <input
          id="phone"
          v-model="phone"
          class="input num"
          type="tel"
          maxlength="11"
          inputmode="numeric"
          placeholder="11 位手机号"
          autocomplete="tel"
        />
      </div>
      <div class="field">
        <label for="code">验证码</label>
        <div class="code-row">
          <input
            id="code"
            v-model="code"
            class="input num"
            type="text"
            maxlength="6"
            inputmode="numeric"
            placeholder="6 位验证码"
            @keyup.enter="login"
          />
          <AppButton variant="outline" :loading="busy" :disabled="countdown > 0" @click="sendCode">
            {{ countdown > 0 ? `${countdown}s` : sent ? '重新发送' : '发送验证码' }}
          </AppButton>
        </div>
      </div>

      <AppButton variant="primary" size="lg" block :loading="busy" :disabled="!sent" @click="login">
        登 录
      </AppButton>
      <p class="agree">登录即代表同意到店服务相关约定 · 演示环境验证码固定为 123456</p>
    </div>
  </div>
</template>

<style scoped>
.login {
  min-height: 100dvh;
  max-width: 420px;
  margin: 0 auto;
  padding: 64px 24px 32px;
  display: flex;
  flex-direction: column;
  animation: anmo-rise 0.2s var(--ease) both;
}

.brand {
  text-align: center;
  margin-bottom: 36px;
}

.seal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 16px;
  background: var(--primary);
  color: var(--primary-foreground);
  font: 600 22px/1 var(--font-stack);
  letter-spacing: 3px;
  text-indent: 3px;
  box-shadow: var(--shadow-card);
  margin-bottom: 16px;
}

.brand h1 {
  font: var(--font-display);
  margin: 0 0 6px;
}

.brand p {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 0;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.code-row {
  display: flex;
  gap: 10px;
}

.code-row .input {
  flex: 1;
  min-width: 0;
}

.code-row .btn {
  flex-shrink: 0;
  min-width: 108px;
}

.agree {
  font: var(--font-caption);
  color: var(--muted-foreground);
  text-align: center;
  margin: 6px 0 0;
}
</style>
