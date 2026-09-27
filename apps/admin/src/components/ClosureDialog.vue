<template>
  <!-- 闭店设置（D22）：按天闭上午/下午/全天，列出可删 -->
  <el-dialog :model-value="modelValue" title="闭店设置" width="min(560px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="load">
    <el-form inline>
      <el-form-item label="日期">
        <el-date-picker v-model="dateStr" type="date" value-format="YYYY-MM-DD" :clearable="false"
          style="width: 140px" />
      </el-form-item>
      <el-form-item label="范围">
        <el-radio-group v-model="part">
          <el-radio-button value="AM">上午</el-radio-button>
          <el-radio-button value="PM">下午</el-radio-button>
          <el-radio-button value="FULL">全天</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="remark" placeholder="如：外出培训" style="width: 140px" />
      </el-form-item>
      <el-form-item>
        <el-button type="danger" :loading="busy" @click="submit">设置闭店</el-button>
      </el-form-item>
    </el-form>

    <el-table :data="list ?? []" size="small" v-loading="loading">
      <el-table-column label="日期" width="120">
        <template #default="{ row }">{{ row.closure_date }}</template>
      </el-table-column>
      <el-table-column label="范围" width="90">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.day_part === 'AM' ? '上午' : '下午' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="140" />
      <el-table-column label="操作" width="80">
        <template #default="{ row }">
          <el-button size="small" link type="danger" @click="remove(row.id)">恢复营业</el-button>
        </template>
      </el-table-column>
    </el-table>
    <p class="hint">顾客端该半天将不可预约；已有预约不会自动取消，需手动联系顾客改期。</p>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createClosure, deleteClosure, listClosures, type Closure } from '../core/api/admin'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const dateStr = ref('')
const part = ref<'AM' | 'PM' | 'FULL'>('FULL')
const remark = ref('')
const list = ref<Closure[] | null>(null)
const loading = ref(false)
const busy = ref(false)

async function load(): Promise<void> {
  loading.value = true
  try {
    list.value = (await listClosures()) ?? []
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) void load()
  },
)

async function submit(): Promise<void> {
  if (!dateStr.value) {
    ElMessage.warning('请选择日期')
    return
  }
  busy.value = true
  try {
    const res = await createClosure(dateStr.value, part.value, remark.value)
    if (res.conflict_count > 0) {
      ElMessage.warning(`已设置闭店：该时段已有 ${res.conflict_count} 个预约，请手动联系顾客改期`)
    } else {
      ElMessage.success('已设置闭店')
    }
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '设置失败')
  } finally {
    busy.value = false
  }
}

async function remove(id: string): Promise<void> {
  try {
    await deleteClosure(id)
    ElMessage.success('已恢复营业')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
</script>

<style scoped>
.hint { color: #999; font-size: 12px; margin-top: 10px; }
</style>
