<template>
  <el-dialog :model-value="modelValue" title="改期" width="min(560px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="onOpen">
    <template v-if="aptId">
      <el-form label-width="64px">
        <el-form-item label="日期">
          <el-date-picker v-model="dateStr" type="date" value-format="YYYY-MM-DD"
            :clearable="false" style="width: 100%" @change="loadOptions" />
        </el-form-item>
      </el-form>

      <div v-loading="loading">
        <div v-if="options && !options.open" class="empty">这一天休息，换一天吧</div>
        <template v-else-if="options">
          <p class="sec">上午 / 下午（必选）</p>
          <div class="part-row">
            <button v-if="!options.am.closed" class="part" :class="{ picked: part === 'AM' }"
              @click="part = 'AM'; slotTime = ''">
              <b>上午</b><span>剩 {{ options.am.remaining }} 名额</span>
            </button>
            <button v-if="!options.pm.closed" class="part" :class="{ picked: part === 'PM' }"
              @click="part = 'PM'; slotTime = ''">
              <b>下午</b><span>剩 {{ options.pm.remaining }} 名额</span>
            </button>
          </div>

          <template v-if="part">
            <p class="sec">具体时间（不选 = 模糊预约，由店家安排）</p>
            <div class="slot-grid">
              <button v-for="s in (part === 'AM' ? options.am.slots : options.pm.slots) ?? []" :key="s.time"
                class="slot" :class="{ picked: s.time === slotTime, full: s.remaining < 1 }"
                :disabled="s.remaining < 1" @click="slotTime = s.time">{{ s.time }}</button>
            </div>
          </template>
        </template>
      </div>
    </template>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="busy" :disabled="!part" @click="submit">
        {{ part ? `改到 ${dateStr}${slotTime ? ' ' + slotTime : part === 'AM' ? ' 上午' : ' 下午'}` : '请选择时段' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getBookingOptions, rescheduleAppointment, type BookingOptions } from '../core/api/admin'

const props = defineProps<{ modelValue: boolean; aptId: string; onDone: () => void }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const dateStr = ref('')
const options = ref<BookingOptions | null>(null)
const part = ref<'' | 'AM' | 'PM'>('')
const slotTime = ref('')
const loading = ref(false)
const busy = ref(false)

function today(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

async function loadOptions(): Promise<void> {
  options.value = null
  part.value = ''
  slotTime.value = ''
  if (!dateStr.value) return
  loading.value = true
  try {
    options.value = await getBookingOptions(dateStr.value)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function onOpen(): void {
  dateStr.value = today()
  void loadOptions()
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) onOpen()
  },
)

async function submit(): Promise<void> {
  if (!props.aptId || !part.value) return
  busy.value = true
  const target = slotTime.value
    ? { start_time: `${dateStr.value} ${slotTime.value}` }
    : { date: dateStr.value, day_part: part.value }
  try {
    await rescheduleAppointment(props.aptId, target)
    ElMessage.success('已改期')
    emit('update:modelValue', false)
    props.onDone()
  } catch (e) {
    const err = e as { code?: string; message: string }
    if (err.code === 'APT_SLOT_FULL' || err.code === 'APT_HALFDAY_FULL' || err.code === 'APT_CLOSED') {
      ElMessage.warning(err.message)
      await loadOptions()
    } else {
      ElMessage.error(err.message)
    }
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.sec { font-size: 13px; color: #666; margin: 10px 0 6px; }
.empty { color: #999; text-align: center; padding: 24px 0; }
.part-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.part { border: 1px solid #dcdfe6; background: #fff; border-radius: 8px; padding: 12px 0; display: flex; flex-direction: column; align-items: center; gap: 2px; cursor: pointer; }
.part b { font-size: 15px; }
.part span { font-size: 12px; color: #999; }
.part.picked { border-color: var(--el-color-primary); color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.slot-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.slot { border: 1px solid #dcdfe6; background: #fff; border-radius: 6px; padding: 8px 0; font-size: 13px; cursor: pointer; }
.slot.full { opacity: .4; cursor: not-allowed; }
.slot.picked { border-color: var(--el-color-primary); background: var(--el-color-primary); color: #fff; }
</style>
