<template>
  <el-dialog :model-value="modelValue" title="账号设置" width="min(440px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)">
    <p class="hint">修改后下次登录生效；当前会话保持有效。手机号用于商家后台登录。</p>
    <el-form label-width="90px">
      <el-form-item label="当前密码" required>
        <el-input v-model="form.currentPassword" type="password" show-password placeholder="验证身份" />
      </el-form-item>
      <el-form-item label="新手机号">
        <el-input v-model="form.newPhone" maxlength="11" inputmode="numeric"
          placeholder="留空则不修改手机号" />
      </el-form-item>
      <el-form-item label="新密码">
        <el-input v-model="form.newPassword" type="password" show-password placeholder="至少 6 位，留空则不修改" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { updateCredentials } from '../core/api/admin'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const saving = ref(false)
const form = ref({ currentPassword: '', newPhone: '', newPassword: '' })

watch(
  () => props.modelValue,
  (v) => {
    if (v) form.value = { currentPassword: '', newPhone: '', newPassword: '' }
  },
)

async function save(): Promise<void> {
  if (!form.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (!form.value.newPhone && !form.value.newPassword) {
    ElMessage.warning('新手机号和新密码至少填一项')
    return
  }
  if (form.value.newPhone && !/^1\d{10}$/.test(form.value.newPhone)) {
    ElMessage.warning('新手机号格式不正确')
    return
  }
  if (form.value.newPassword && form.value.newPassword.length < 6) {
    ElMessage.warning('新密码至少 6 位')
    return
  }
  saving.value = true
  try {
    await updateCredentials({
      current_password: form.value.currentPassword,
      new_phone: form.value.newPhone || undefined,
      new_password: form.value.newPassword || undefined,
    })
    ElMessage.success('已保存，下次登录请使用新手机号/密码')
    emit('update:modelValue', false)  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.hint {
  color: #909399;
  font-size: 12px;
  margin: 0 0 12px;
  line-height: 1.6;
}
</style>
