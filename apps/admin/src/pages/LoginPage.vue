<template>
  <div class="wrap">
    <el-card class="card">
      <h2 class="h">Anmo 商家管理</h2>
      <p class="tip">请使用商家账号登录（老板/店员）</p>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="手机号">
          <el-input v-model="phone" placeholder="商家手机号" maxlength="11" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" show-password placeholder="密码"
            @keyup.enter="submit" />
        </el-form-item>
        <el-button type="primary" class="btn" :loading="loading" @click="submit">登 录</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login } from '../core/api/admin'
import { setSession } from '../platform/auth'

const router = useRouter()
const phone = ref('')
const password = ref('')
const loading = ref(false)

async function submit() {
  if (!phone.value || !password.value) {
    ElMessage.warning('请输入手机号和密码')
    return
  }
  loading.value = true
  try {
    const res = await login(phone.value.trim(), password.value)
    setSession(res.token, res.user)
    ElMessage.success('登录成功')
    void router.push('/dashboard')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #2f4554 100%);
}
.card {
  width: 360px;
  padding: 8px 12px 4px;
}
.h {
  text-align: center;
  margin: 4px 0 2px;
  color: #1f2d3d;
}
.tip {
  text-align: center;
  color: #909399;
  font-size: 13px;
  margin: 0 0 14px;
}
.btn {
  width: 100%;
}
</style>
