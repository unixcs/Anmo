<template>
  <!-- 调整卡次数：步进器 + 快捷按钮，无需手写正负号 -->
  <el-dialog :model-value="modelValue" title="调整次数" width="min(420px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="onOpen">
    <template v-if="card">
      <div class="current">
        <el-button circle size="large" :disabled="delta <= -card.remaining_count" @click="delta--">−</el-button>
        <div class="count-box">
          <div class="now">{{ card.remaining_count + delta }}</div>
          <div class="meta">调整后剩余 / 现有 {{ card.remaining_count }} 次</div>
        </div>
        <el-button circle size="large" @click="delta++">＋</el-button>
      </div>
      <div class="quick">
        <el-button v-for="q in QUICK" :key="q" size="small" round @click="delta = q">{{ q > 0 ? `+${q}` : q }}</el-button>
      </div>
      <el-input v-model="remark" placeholder="调整备注（如：手工调整、活动补偿）" style="margin-top: 14px" />
    </template>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="busy" :disabled="delta === 0" @click="submit">
        {{ delta === 0 ? '请选择调整量' : delta > 0 ? `增加 ${delta} 次` : `扣减 ${-delta} 次` }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { adjustCard, type MemberCard } from '../core/api/admin'

const props = defineProps<{ modelValue: boolean; card: MemberCard | null; onDone: () => void }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const QUICK = [-3, -1, 1, 3]

const delta = ref(0)
const remark = ref('手工调整')
const busy = ref(false)

function onOpen(): void {
  delta.value = 0
  remark.value = '手工调整'
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) onOpen()
  },
)

async function submit(): Promise<void> {
  if (!props.card || delta.value === 0) return
  busy.value = true
  try {
    await adjustCard(props.card.id, delta.value, remark.value || '手工调整')
    ElMessage.success(`已${delta.value > 0 ? '增加' : '扣减'} ${Math.abs(delta.value)} 次`)
    emit('update:modelValue', false)
    props.onDone()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '调整失败')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.current { display: flex; align-items: center; justify-content: center; gap: 22px; padding: 10px 0 4px; }
.count-box { text-align: center; }
.now { font-size: 34px; font-weight: 700; font-variant-numeric: tabular-nums; }
.meta { color: #999; font-size: 12px; margin-top: 2px; }
.quick { display: flex; justify-content: center; gap: 8px; margin-top: 14px; }
</style>
