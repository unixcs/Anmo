<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../core/api/endpoints'
import type { Member } from '../core/models/models'
import { notify } from '../platform/notify/toast'
import AppButton from '../components/ui/AppButton.vue'

const router = useRouter()

const member = ref<Member | null>(null)
const name = ref('')
const gender = ref('')
const birthday = ref('')
const busy = ref(false)

onMounted(async () => {
  try {
    const res = await api.myProfile()
    member.value = res.member
    name.value = res.member.name
    gender.value = res.member.gender
    birthday.value = res.member.birthday ?? ''
  } catch (e) {
    notify((e as Error).message)
  }
})

async function save(): Promise<void> {
  busy.value = true
  try {
    const patch: Record<string, string> = {}
    if (name.value) patch.name = name.value
    if (gender.value) patch.gender = gender.value
    if (birthday.value) patch.birthday = birthday.value
    const res = await api.updateProfile(patch)
    member.value = res.member
    notify('已保存')
    // 从预约弹层「去完善资料」进来时：保存后直接回预约页，草稿由 BookingPage onMounted takeDraft 恢复
    if (sessionStorage.getItem('anmo.booking.draft')) {
      router.push('/booking')
    }
  } catch (e) {
    notify((e as Error).message)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page profile">
    <div class="page-head">
      <h1>个人资料</h1>
      <p class="sub">方便店主称呼您、为您安排偏好</p>
    </div>

    <div class="card form">
      <div class="field">
        <label>手机号</label>
        <input class="input num" :value="member?.phone" disabled />
      </div>
      <div class="field">
        <label>会员号</label>
        <input class="input num" :value="member?.member_no" disabled />
      </div>
      <div class="field">
        <label for="pname">称呼</label>
        <input id="pname" v-model="name" class="input" placeholder="怎么称呼您" />
      </div>
      <div class="field">
        <label for="pgender">性别</label>
        <select id="pgender" v-model="gender" class="input">
          <option value="">不透露</option>
          <option value="女">女</option>
          <option value="男">男</option>
        </select>
      </div>
      <div class="field">
        <label for="pbday">生日</label>
        <input id="pbday" v-model="birthday" class="input num" type="date" />
      </div>
      <AppButton block size="lg" :loading="busy" @click="save">保存</AppButton>
    </div>
  </div>
</template>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.input:disabled {
  background: var(--muted);
  color: var(--muted-foreground);
}
</style>
