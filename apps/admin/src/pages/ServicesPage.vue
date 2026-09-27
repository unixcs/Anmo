<template>
  <div>
    <el-card shadow="never" class="block">
      <template #header>
        <div class="head">
          <span>服务分类（用户端按此分组展示；下架后该分类在用户端隐藏）</span>
          <el-button size="small" type="primary" @click="catVisible = true">新建分类</el-button>
        </div>
      </template>
      <template v-if="isMobile">
        <div v-for="row in categories" :key="row.id" class="sv-card">
          <div class="sv-top"><b>{{ row.name }}</b>
            <el-switch :model-value="row.status === 'ACTIVE'"
              @change="(v: string | number | boolean) => toggleCat(row, Boolean(v))" />
          </div>
          <div class="sv-meta">排序 {{ row.sort }}</div>
        </div>
      </template>
      <el-table v-else :data="categories" size="default" v-loading="loading">
        <el-table-column prop="name" label="分类名" min-width="140" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 'ACTIVE'"
              @change="(v: string | number | boolean) => toggleCat(row, Boolean(v))" />
            <span class="st">{{ row.status === 'ACTIVE' ? '上架' : '下架' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="head">
          <span>服务项目（价格/时长/上架状态直接作用于用户端）</span>
          <el-button size="small" type="primary" @click="openCreate">新建项目</el-button>
        </div>
      </template>
      <template v-if="isMobile">
        <div v-for="row in items" :key="row.id" class="sv-card">
          <div class="sv-top"><b>{{ row.name }}</b>
            <el-switch :model-value="row.status === 'ACTIVE'"
              @change="(v: string | number | boolean) => toggleItem(row, Boolean(v))" />
          </div>
          <div class="sv-meta">{{ catName(row.category_id) }} · {{ row.duration_minutes }} 分钟 · {{ yuan(row.default_price) }}</div>
          <div v-if="row.description" class="sv-meta">描述：{{ row.description }}</div>
          <div class="sv-btns"><el-button size="small" @click="openEdit(row)">编辑</el-button></div>
        </div>
      </template>
      <el-table v-else :data="items" size="default" v-loading="loading">
        <el-table-column label="分类" width="120">
          <template #default="{ row }">{{ catName(row.category_id) }}</template>
        </el-table-column>
        <el-table-column prop="name" label="项目名" min-width="130" />
        <el-table-column label="时长" width="90">
          <template #default="{ row }">{{ row.duration_minutes }} 分钟</template>
        </el-table-column>
        <el-table-column label="价格" width="100">
          <template #default="{ row }">{{ yuan(row.default_price) }}</template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="上架" width="120">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 'ACTIVE'"
              @change="(v: string | number | boolean) => toggleItem(row, Boolean(v))" />
            <span class="st">{{ row.status === 'ACTIVE' ? '上架' : '下架' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="catVisible" title="新建分类" width="min(380px, 94vw)">
      <el-form label-width="80px">
        <el-form-item label="分类名" required><el-input v-model="catForm.name" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="catForm.sort" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="catVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreateCat">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="itemVisible" :title="editing ? '编辑项目' : '新建项目'" width="min(500px, 94vw)">
      <el-form label-width="100px">
        <el-form-item label="分类" required>
          <el-select v-model="itemForm.category_id" style="width: 100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="项目名" required><el-input v-model="itemForm.name" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="itemForm.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="时长（分钟）" required>
          <el-input-number v-model="itemForm.duration_minutes" :min="5" :step="5" />
        </el-form-item>
        <el-form-item label="价格（元）" required>
          <el-input-number v-model="itemForm.priceYuan" :min="0" :precision="2" :step="10" />
        </el-form-item>
        <el-form-item label="封面图 URL"><el-input v-model="itemForm.cover_image" placeholder="选填" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="itemForm.sort" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doSaveItem">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createCategory,
  createService,
  listCategories,
  listServices,
  setCategoryStatus,
  setServiceStatus,
  updateService,
  type ServiceCategory,
  type ServiceItem,
} from '../core/api/admin'
import { toFen, yuan } from '../core/format'
import { useIsMobile } from '../core/useMedia'

const categories = ref<ServiceCategory[]>([])
const items = ref<ServiceItem[]>([])
const loading = ref(false)
const isMobile = useIsMobile()
const saving = ref(false)

const catVisible = ref(false)
const catForm = ref({ name: '', sort: 0 })

const itemVisible = ref(false)
const editing = ref<ServiceItem | null>(null)
const itemForm = ref({
  category_id: '',
  name: '',
  description: '',
  duration_minutes: 60,
  priceYuan: 128,
  cover_image: '',
  sort: 0,
})

function catName(id: string): string {
  return categories.value.find((c) => c.id === id)?.name ?? '-'
}

async function load() {
  loading.value = true
  try {
    const [c, s] = await Promise.all([listCategories(), listServices()])
    categories.value = c ?? []
    items.value = s ?? []
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function doCreateCat() {
  if (!catForm.value.name) {
    ElMessage.warning('请输入分类名')
    return
  }
  saving.value = true
  try {
    await createCategory(catForm.value.name, catForm.value.sort)
    ElMessage.success('已创建')
    catVisible.value = false
    catForm.value = { name: '', sort: 0 }
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    saving.value = false
  }
}

async function toggleCat(row: ServiceCategory, on: boolean) {
  try {
    await setCategoryStatus(row.id, on ? 'ACTIVE' : 'INACTIVE')
    row.status = on ? 'ACTIVE' : 'INACTIVE'
    ElMessage.success(on ? '分类已上架' : '分类已下架')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function openCreate() {
  editing.value = null
  itemForm.value = {
    category_id: categories.value[0]?.id ?? '',
    name: '', description: '', duration_minutes: 60,
    priceYuan: 128, cover_image: '', sort: 0,
  }
  itemVisible.value = true
}

function openEdit(row: ServiceItem) {
  editing.value = row
  itemForm.value = {
    category_id: row.category_id,
    name: row.name,
    description: row.description,
    duration_minutes: row.duration_minutes,
    priceYuan: row.default_price / 100,
    cover_image: row.cover_image,
    sort: row.sort,
  }
  itemVisible.value = true
}

async function doSaveItem() {
  if (!itemForm.value.name || !itemForm.value.category_id) {
    ElMessage.warning('请填写分类与项目名')
    return
  }
  saving.value = true
  const body = {
    category_id: itemForm.value.category_id,
    name: itemForm.value.name,
    description: itemForm.value.description,
    duration_minutes: itemForm.value.duration_minutes,
    default_price: toFen(String(itemForm.value.priceYuan)),
    cover_image: itemForm.value.cover_image,
    sort: itemForm.value.sort,
  }
  try {
    if (editing.value) await updateService(editing.value.id, body)
    else await createService(body)
    ElMessage.success('已保存')
    itemVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleItem(row: ServiceItem, on: boolean) {
  try {
    await setServiceStatus(row.id, on ? 'ACTIVE' : 'INACTIVE')
    row.status = on ? 'ACTIVE' : 'INACTIVE'
    ElMessage.success(on ? '已上架，用户端可见' : '已下架，用户端隐藏')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

onMounted(load)
</script>

<style scoped>
.block {
  margin-bottom: 14px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.st {
  margin-left: 6px;
  color: #909399;
  font-size: 12px;
}
</style>
<style scoped>
.sv-card { border: 1px solid #ebeef5; border-radius: 10px; padding: 12px; margin-bottom: 10px; background: #fff; }
.sv-top { display: flex; justify-content: space-between; align-items: center; }
.sv-meta { color: #606266; font-size: 13px; margin-top: 4px; }
.sv-btns { margin-top: 8px; display: flex; gap: 8px; }
</style>
