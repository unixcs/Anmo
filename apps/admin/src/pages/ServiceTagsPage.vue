<template>
  <el-card shadow="never">
    <p class="page-hint">
      标签用于结算时快速记录「调理部位 / 服务方式」；服务记录保存的是名称快照，
      停用/删除不影响历史记录。被服务记录引用过的标签仅可停用。
    </p>
    <div class="groups">
      <div v-for="g in groups" :key="g.group" class="group">
        <div class="group-head">
          <h4>{{ g.title }}</h4>
          <div class="add">
            <el-input v-model="g.draft" size="small" maxlength="12" :placeholder="`新${g.noun}（≤12字）`"
              @keyup.enter="doAdd(g)" />
            <el-button size="small" type="primary" @click="doAdd(g)">添加</el-button>
          </div>
        </div>
        <el-table :data="g.rows" size="small" v-loading="g.loading" row-key="id"
          :row-class-name="(r: { row: ServiceTag }) => (r.row.status === 'DISABLED' ? 'row-disabled' : '')">
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'ACTIVE' ? 'success' : 'info'">
                {{ row.status === 'ACTIVE' ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="230">
            <template #default="{ row, $index }">
              <el-button size="small" link type="primary" :disabled="$index === 0" @click="doMove(g, row, -1)">上移</el-button>
              <el-button size="small" link type="primary" :disabled="$index === g.rows.length - 1"
                @click="doMove(g, row, 1)">下移</el-button>
              <el-button size="small" link type="primary" @click="doToggleStatus(g, row)">
                {{ row.status === 'ACTIVE' ? '停用' : '启用' }}
              </el-button>
              <el-button size="small" link type="danger" @click="doDelete(g, row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div class="group presets">
      <div class="group-head">
        <h4>技师备注快捷短语</h4>
        <div class="add">
          <el-input v-model="presetDraft" size="small" maxlength="50" placeholder="新短语（≤50字）"
            @keyup.enter="addPreset" />
          <el-button size="small" type="primary" @click="addPreset">添加</el-button>
        </div>
      </div>
      <p class="presets-hint">结算弹窗「技师备注」下方点击即填入；最多 20 条。</p>
      <div class="preset-chips">
        <el-tag v-for="(p, i) in presets" :key="`${p}-${i}`" closable class="preset-chip" @close="removePreset(i)">
          {{ p }}
        </el-tag>
        <span v-if="presets.length === 0" class="presets-none">暂无短语</span>
      </div>
      <el-button type="primary" :loading="savingPresets" @click="savePresets">保存短语</el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
// ServiceTagsPage — 服务标签管理（D28 §3.1）：部位/方式两组 + 技师备注快捷短语。
// 删除被引用的标签返回 409 TAG_IN_USE → 提示仅可停用。
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createServiceTag,
  deleteServiceTag,
  getSettings,
  listServiceTags,
  saveSetting,
  updateServiceTag,
  type ServiceTag,
} from '../core/api/admin'
import { ApiError } from '../core/api/http'

interface TagGroup {
  group: 'BODY_PART' | 'METHOD'
  title: string
  noun: string
  rows: ServiceTag[]
  draft: string
  loading: boolean
}

const groups = reactive<TagGroup[]>([
  { group: 'BODY_PART', title: '调理部位', noun: '部位', rows: [], draft: '', loading: false },
  { group: 'METHOD', title: '服务方式', noun: '方式', rows: [], draft: '', loading: false },
])

const presets = ref<string[]>([])
const presetDraft = ref('')
const savingPresets = ref(false)

async function loadGroup(g: TagGroup): Promise<void> {
  g.loading = true
  try {
    g.rows = ((await listServiceTags(g.group)) ?? []).sort((a, b) => a.sort - b.sort)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载标签失败')
  } finally {
    g.loading = false
  }
}

async function loadPresets(): Promise<void> {
  try {
    const settings = await getSettings()
    const raw = settings['service_note_presets']
    const parsed = raw ? (JSON.parse(raw) as unknown) : []
    presets.value = Array.isArray(parsed) ? parsed.filter((x): x is string => typeof x === 'string') : []
  } catch {
    presets.value = []
  }
}

async function doAdd(g: TagGroup): Promise<void> {
  const name = g.draft.trim()
  if (!name) {
    ElMessage.warning(`请输入${g.noun}名称`)
    return
  }
  try {
    await createServiceTag(g.group, name)
    g.draft = ''
    await loadGroup(g)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '添加失败')
  }
}

async function doMove(g: TagGroup, row: ServiceTag, dir: -1 | 1): Promise<void> {
  const idx = g.rows.findIndex((r) => r.id === row.id)
  const swapIdx = idx + dir
  if (idx < 0 || swapIdx < 0 || swapIdx >= g.rows.length) return
  const other = g.rows[swapIdx]
  try {
    await Promise.all([
      updateServiceTag(row.id, { sort: other.sort }),
      updateServiceTag(other.id, { sort: row.sort }),
    ])
    await loadGroup(g)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '排序失败')
  }
}

async function doToggleStatus(g: TagGroup, row: ServiceTag): Promise<void> {
  try {
    await updateServiceTag(row.id, { status: row.status === 'ACTIVE' ? 'DISABLED' : 'ACTIVE' })
    await loadGroup(g)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

async function doDelete(g: TagGroup, row: ServiceTag): Promise<void> {
  try {
    await deleteServiceTag(row.id)
    ElMessage.success('已删除')
    await loadGroup(g)
  } catch (e) {
    if (e instanceof ApiError && e.code === 'TAG_IN_USE') {
      ElMessage.warning('该标签已被服务记录使用，仅可停用')
    } else {
      ElMessage.error(e instanceof Error ? e.message : '删除失败')
    }
  }
}

function addPreset(): void {
  const v = presetDraft.value.trim()
  if (!v) return
  if (presets.value.length >= 20) {
    ElMessage.warning('最多 20 条')
    return
  }
  presets.value.push(v)
  presetDraft.value = ''
}

function removePreset(i: number): void {
  presets.value.splice(i, 1)
}

async function savePresets(): Promise<void> {
  savingPresets.value = true
  try {
    await saveSetting('service_note_presets', JSON.stringify(presets.value))
    ElMessage.success('已保存')
    await loadPresets()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    savingPresets.value = false
  }
}

onMounted(() => {
  for (const g of groups) void loadGroup(g)
  void loadPresets()
})
</script>

<style scoped>
.page-hint {
  color: #909399;
  font-size: 12px;
  margin: 0 0 12px;
}
.groups {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}
.group {
  flex: 1 1 320px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
}
.group-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  gap: 10px;
}
.group-head h4 {
  margin: 0;
}
.add {
  display: flex;
  gap: 6px;
}
.add .el-input {
  width: 160px;
}
:deep(.row-disabled) {
  color: #b0b3b8;
}
.presets {
  margin-top: 16px;
}
.presets-hint {
  color: #909399;
  font-size: 12px;
  margin: 0 0 8px;
}
.preset-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
}
.preset-chip {
  margin: 0;
}
.presets-none {
  color: #b0b3b8;
  font-size: 12px;
}
</style>
