<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../core/api/endpoints'
import { signIn } from '../platform/auth/session'
import { notify } from '../platform/notify/toast'
import AppButton from '../components/ui/AppButton.vue'

// V2.2: 手机号+密码 登录 ⇄ 注册 切换（短信验证码登录已下线）。
// 错误文案一律以服务端 msg 为准（http 已解包并带 code/msg）。
const router = useRouter()
const route = useRoute()
const mode = ref<'login' | 'register'>('login')
const phone = ref('')
const password = ref('')
const confirm = ref('')
const busy = ref(false)

function toggleMode(): void {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  password.value = ''
  confirm.value = ''
}

function forgotPassword(): void {
  notify('请联系商家后台重置密码')
}

async function submit(): Promise<void> {
  if (!/^1\d{10}$/.test(phone.value)) {
    notify('请输入正确的手机号')
    return
  }
  if (password.value.length < 6) {
    notify('密码需 6~64 位')
    return
  }
  if (mode.value === 'register' && password.value !== confirm.value) {
    notify('两次输入的密码不一致')
    return
  }
  busy.value = true
  try {
    const res =
      mode.value === 'login'
        ? await api.login(phone.value, password.value)
        : await api.register(phone.value, password.value)
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
  <div class="login">
    <div class="brand">
      <span class="seal">安摩</span>
      <h1>{{ mode === 'login' ? '欢迎回来' : '创建账号' }}</h1>
      <p>{{ mode === 'login' ? '手机号 + 密码登录' : '注册后即可预约到店服务' }}</p>
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
        <label for="password">密码</label>
        <input
          id="password"
          v-model="password"
          class="input"
          type="password"
          maxlength="64"
          placeholder="6~64 位密码"
          :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
          @keyup.enter="submit"
        />
      </div>
      <div v-if="mode === 'register'" class="field">
        <label for="confirm">确认密码</label>
        <input
          id="confirm"
          v-model="confirm"
          class="input"
          type="password"
          maxlength="64"
          placeholder="再次输入密码"
          autocomplete="new-password"
          @keyup.enter="submit"
        />
      </div>

      <AppButton variant="primary" size="lg" block :loading="busy" @click="submit">
        {{ mode === 'login' ? '登 录' : '注 册' }}
      </AppButton>

      <div class="links">
        <button type="button" class="link" @click="toggleMode">
          {{ mode === 'login' ? '没有账号？注册' : '已有账号？登录' }}
        </button>
        <button v-if="mode === 'login'" type="button" class="link" @click="forgotPassword">
          忘记密码？
        </button>
      </div>
      <p class="agree">登录即代表同意到店服务相关约定</p>
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

.links {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.link {
  background: none;
  border: none;
  padding: 0;
  font: var(--font-sub);
  color: var(--primary);
  cursor: pointer;
}

.agree {
  font: var(--font-caption);
  color: var(--muted-foreground);
  text-align: center;
  margin: 6px 0 0;
}
</style>
