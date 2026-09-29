<template>
  <div class="rec-fields" :class="{ disabled: props.disabled }">
    <div class="rf-row">
      <span class="rf-label">调理部位</span>
      <div class="rf-chips">
        <el-check-tag v-for="p in parts" :key="p" :checked="isPart(p)" :disabled="!isPart(p) && full"
          @change="togglePart(p)">{{ p }}</el-check-tag>
        <span v-if="parts.length === 0" class="rf-none">暂无标签（可在「服务标签管理」添加）</span>
      </div>
    </div>
    <div class="rf-row">
      <span class="rf-label">服务方式</span>
      <div class="rf-chips">
        <el-check-tag v-for="m in methods" :key="m" :checked="model.service_method === m"
          @change="toggleMethod(m)">{{ m }}</el-check-tag>
        <span v-if="methods.length === 0" class="rf-none">暂无标签（可在「服务标签管理」添加）</span>
      </div>
    </div>
    <div class="rf-row">
      <span class="rf-label">技师备注</span>
      <div class="rf-note">
        <el-input v-model="model.tech_note" type="textarea" :rows="2" maxlength="200" show-word-limit
          placeholder="本次服务记录（顾客不可见，选填，≤200 字）" />
        <div v-if="presets.length" class="rf-chips rf-presets">
          <el-tag v-for="preset in presets" :key="preset" class="preset-chip" @click="applyPreset(preset)">
            {{ preset }}
          </el-tag>
        </div>
      </div>
    </div>
    <el-checkbox v-model="model.communicated" class="rf-confirm">
      服务前已完成沟通（必勾才能提交）
    </el-checkbox>
  </div>
</template>

<script setup lang="ts">
// ServiceRecordFields — 结算弹窗共用的服务记录录入区块（D28 §3.2）：
// 部位多选 chips（≤3，超出禁选）、方式单选 chips、技师备注 + 快捷短语、
// 底部必勾「服务前已完成沟通」。字段名与后端 json tag 逐字一致。
import { computed, onMounted, ref } from 'vue'
import { getSettings, listServiceTags, type RecordPayload } from '../core/api/admin'

const model = defineModel<RecordPayload>({ required: true })

const props = defineProps<{
  /** 散客未录手机号时整体禁用（仅记账不生成记录） */
  disabled?: boolean
}>()

const parts = ref<string[]>([])
const methods = ref<string[]>([])
const presets = ref<string[]>([])

const full = computed(() => (model.value.body_parts ?? []).length >= 3)

function isPart(p: string): boolean {
  return (model.value.body_parts ?? []).includes(p)
}

function togglePart(p: string): void {
  if (props.disabled) return
  const cur = model.value.body_parts ?? []
  model.value.body_parts = isPart(p) ? cur.filter((x) => x !== p) : [...cur, p]
}

function toggleMethod(m: string): void {
  if (props.disabled) return
  model.value.service_method = model.value.service_method === m ? '' : m
}

function applyPreset(preset: string): void {
  if (props.disabled) return
  const cur = model.value.tech_note ?? ''
  model.value.tech_note = cur ? `${cur}${preset}` : preset
}

onMounted(async () => {
  try {
    const [tags, settings] = await Promise.all([
      listServiceTags(),
      getSettings().catch(() => ({}) as Record<string, string>),
    ])
    parts.value = (tags ?? []).filter((t) => t.tag_group === 'BODY_PART' && t.status === 'ACTIVE').map((t) => t.name)
    methods.value = (tags ?? []).filter((t) => t.tag_group === 'METHOD' && t.status === 'ACTIVE').map((t) => t.name)
    try {
      const raw = (settings as Record<string, string>)['service_note_presets']
      const parsed = raw ? (JSON.parse(raw) as unknown) : []
      if (Array.isArray(parsed)) presets.value = parsed.filter((x): x is string => typeof x === 'string')
    } catch {
      /* 脏值后端已清洗为空，忽略 */
    }
  } catch {
    /* 标签加载失败不阻塞结算 */
  }
})
</script>

<style scoped>
.rec-fields {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 10px 12px;
  margin: 10px 0;
}
.rec-fields.disabled {
  opacity: 0.55;
}
.rf-row {
  display: flex;
  gap: 10px;
  margin-bottom: 8px;
}
.rf-label {
  flex: 0 0 60px;
  color: #606266;
  font-size: 13px;
  padding-top: 4px;
}
.rf-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.rf-none {
  color: #b0b3b8;
  font-size: 12px;
}
.rf-note {
  flex: 1;
}
.rf-presets {
  margin-top: 6px;
}
.preset-chip {
  cursor: pointer;
}
.rf-confirm {
  margin-bottom: 0;
}
</style>
