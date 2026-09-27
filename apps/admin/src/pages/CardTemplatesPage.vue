<template>
  <el-card shadow="never">
    <div class="toolbar">
      <span class="hint">卡模板定义卡的类型/次数/有效期/售价；"可核销服务"决定这张卡能约哪些项目。</span>
      <span class="spacer" />
      <el-button type="primary" @click="openCreate">新建卡模板</el-button>
    </div>

    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="name" label="名称" min-width="130" />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">{{ CARD_TYPE_TEXT[row.type] }}</template>
      </el-table-column>
      <el-table-column prop="total_count" label="总次数" width="80" />
      <el-table-column label="有效期" width="180">
        <template #default="{ row }">
          {{ row.validity_type === 'PERMANENT' ? '永久' : `${row.valid_from ?? ''} ~ ${row.valid_until ?? ''}` }}
        </template>
      </el-table-column>
      <el-table-column label="售价" width="90">
        <template #default="{ row }">{{ yuan(row.price) }}</template>
      </el-table-column>
      <el-table-column label="可核销服务" min-width="160">
        <template #default="{ row }">{{ rulesText(row.id) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 'ACTIVE'"
            @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openRules(row)">可核销服务</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="createVisible" title="新建卡模板" width="min(480px, 94vw)">
    <el-form label-width="100px">
      <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="类型">
        <el-radio-group v-model="form.type">
          <el-radio value="COUNT">次卡</el-radio>
          <el-radio value="ACTIVITY">活动卡</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="总次数" required>
        <el-input-number v-model="form.total_count" :min="1" :max="999" />
      </el-form-item>
      <el-form-item label="有效期">
        <el-radio-group v-model="form.validity_type">
          <el-radio value="PERMANENT">永久</el-radio>
          <el-radio value="FIXED">固定期限</el-radio>
        </el-radio-group>
      </el-form-item>
      <template v-if="form.validity_type === 'FIXED'">
        <el-form-item label="生效日期">
          <el-date-picker v-model="form.valid_from" type="date" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="失效日期">
          <el-date-picker v-model="form.valid_until" type="date" value-format="YYYY-MM-DD" />
        </el-form-item>
      </template>
      <el-form-item label="售价（元）" required>
        <el-input-number v-model="form.priceYuan" :min="0" :precision="2" :step="100" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="doCreate">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="rulesVisible" title="设置可核销服务" width="min(460px, 94vw)">
    <el-select v-model="rulesIds" multiple style="width: 100%" placeholder="不选=该卡不能核销任何项目">
      <el-option v-for="s in services.filter((x) => x.status === 'ACTIVE')" :key="s.id" :label="s.name" :value="s.id" />
    </el-select>
    <template #footer>
      <el-button @click="rulesVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="doSaveRules">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createCardTemplate,
  listCardTemplates,
  listServices,
  setCardTemplateRules,
  setCardTemplateStatus,
  type CardTemplate,
  type ServiceItem,
} from '../core/api/admin'
import { CARD_TYPE_TEXT, toFen, yuan } from '../core/format'

const rows = ref<CardTemplate[]>([])
const services = ref<ServiceItem[]>([])
const rulesMap = ref<Record<string, string[]>>({})
const loading = ref(false)
const saving = ref(false)

const createVisible = ref(false)
const form = ref({
  name: '',
  type: 'COUNT' as 'COUNT' | 'ACTIVITY',
  total_count: 10,
  validity_type: 'PERMANENT' as 'PERMANENT' | 'FIXED',
  valid_from: '' as string | '',
  valid_until: '' as string | '',
  priceYuan: 1000,
})

const rulesVisible = ref(false)
const rulesIds = ref<string[]>([])
const rulesTarget = ref<CardTemplate | null>(null)

function rulesText(id: string): string {
  const ids = rulesMap.value[id] ?? []
  if (ids.length === 0) return '未配置'
  const names = ids.map((x) => services.value.find((s) => s.id === x)?.name ?? x.slice(0, 6))
  return names.join('、')
}

async function load() {
  loading.value = true
  try {
    const [t, s] = await Promise.all([listCardTemplates(), listServices()])
    rows.value = t ?? []
    services.value = s ?? []
    rulesMap.value = Object.fromEntries(rows.value.map((x) => [x.id, x.service_ids ?? []]))
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = {
    name: '', type: 'COUNT', total_count: 10, validity_type: 'PERMANENT',
    valid_from: '', valid_until: '', priceYuan: 1000,
  }
  createVisible.value = true
}

async function doCreate() {
  if (!form.value.name) {
    ElMessage.warning('请输入名称')
    return
  }
  if (form.value.validity_type === 'FIXED' && (!form.value.valid_from || !form.value.valid_until)) {
    ElMessage.warning('固定期限必须选择起止日期')
    return
  }
  saving.value = true
  try {
    await createCardTemplate({
      name: form.value.name,
      type: form.value.type,
      total_count: form.value.total_count,
      validity_type: form.value.validity_type,
      valid_from: form.value.valid_from || null,
      valid_until: form.value.valid_until || null,
      price: toFen(String(form.value.priceYuan)),
    })
    ElMessage.success('已创建')
    createVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    saving.value = false
  }
}

async function toggle(row: CardTemplate, on: boolean) {
  try {
    await setCardTemplateStatus(row.id, on ? 'ACTIVE' : 'INACTIVE')
    row.status = on ? 'ACTIVE' : 'INACTIVE'
    ElMessage.success(on ? '已上架（可开卡）' : '已下架（不可开卡）')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function openRules(row: CardTemplate) {
  rulesTarget.value = row
  rulesIds.value = [...(rulesMap.value[row.id] ?? [])]
  rulesVisible.value = true
}

async function doSaveRules() {
  if (!rulesTarget.value) return
  saving.value = true
  try {
    await setCardTemplateRules(rulesTarget.value.id, rulesIds.value)
    rulesMap.value[rulesTarget.value.id] = [...rulesIds.value]
    ElMessage.success('已保存')
    rulesVisible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  align-items: center;
}
.hint {
  color: #909399;
  font-size: 12px;
}
.spacer {
  flex: 1;
}
</style>
