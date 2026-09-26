<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Member } from '../core/models/models'
import { notify } from '../platform/notify/toast'

const member = ref<Member | null>(null)
const name = ref('')
const gender = ref('')
const birthday = ref('')

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
  try {
    const patch: Record<string, string> = {}
    if (name.value) patch.name = name.value
    if (gender.value) patch.gender = gender.value
    if (birthday.value) patch.birthday = birthday.value
    const res = await api.updateProfile(patch)
    member.value = res.member
    notify('已保存')
  } catch (e) {
    notify((e as Error).message)
  }
}
</script>

<template>
  <div class="page profile">
    <h1>个人资料</h1>
    <div class="form">
      <label>手机号</label>
      <input :value="member?.phone" disabled />
      <label>会员号</label>
      <input :value="member?.member_no" disabled />
      <label>称呼</label>
      <input v-model="name" placeholder="怎么称呼您" />
      <label>性别</label>
      <select v-model="gender">
        <option value="">不透露</option>
        <option value="女">女</option>
        <option value="男">男</option>
      </select>
      <label>生日</label>
      <input v-model="birthday" type="date" />
      <button class="primary" @click="save">保存</button>
    </div>
  </div>
</template>

<style scoped>
.profile { padding: 20px 16px; }
h1 { font-size: 20px; }
.form { display: flex; flex-direction: column; gap: 8px; background: #fff; border-radius: 12px; padding: 16px; }
label { color: #999; font-size: 13px; }
input, select { height: 42px; border: 1px solid #ddd; border-radius: 8px; padding: 0 10px; font-size: 15px; }
input:disabled { background: #f7f7f7; color: #999; }
.primary { margin-top: 12px; height: 44px; background: #c85f5f; color: #fff; border: none; border-radius: 10px; font-size: 16px; }
</style>
