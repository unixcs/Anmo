<template>
  <el-card shadow="never">
    <el-tabs v-model="tab" @tab-change="load">
      <!-- 轮播图 -->
      <el-tab-pane label="轮播图" name="banners">
        <div class="tab-head">
          <span class="hint">用户端首页轮播展示；下架即隐藏。</span>
          <el-button size="small" type="primary" @click="openBanner(null)">新建轮播图</el-button>
        </div>
        <el-table :data="banners" size="default" v-loading="loading">
          <el-table-column prop="title" label="标题" min-width="120" />
          <el-table-column prop="image" label="图片" min-width="180" show-overflow-tooltip />
          <el-table-column prop="link" label="跳转" min-width="120" show-overflow-tooltip />
          <el-table-column prop="sort" label="排序" width="70" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-switch :model-value="row.status === 'ACTIVE'"
                @change="(v: string | number | boolean) => toggleBanner(row, Boolean(v))" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openBanner(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 公告 -->
      <el-tab-pane label="公告" name="announcements">
        <div class="tab-head">
          <span class="hint">用户端公告栏展示。</span>
          <el-button size="small" type="primary" @click="openAnn(null)">新建公告</el-button>
        </div>
        <el-table :data="announcements" size="default" v-loading="loading">
          <el-table-column prop="title" label="标题" min-width="140" />
          <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
          <el-table-column prop="sort" label="排序" width="70" />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-switch :model-value="row.status === 'ACTIVE'"
                @change="(v: string | number | boolean) => toggleAnn(row, Boolean(v))" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openAnn(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 首页布局 -->
      <el-tab-pane label="首页布局" name="page">
        <div class="tab-head">
          <span class="hint">控制用户端首页内容块顺序（banner/announcement/service_list/activity/richtext）。</span>
          <div>
            <el-button size="small" @click="addBlock">添加内容块</el-button>
            <el-button size="small" type="primary" :loading="saving" @click="saveBlocks">保存布局</el-button>
          </div>
        </div>
        <el-empty v-if="blocks.length === 0" description="暂无内容块，点击右上角添加" :image-size="70" />
        <div v-for="(b, i) in blocks" :key="i" class="block-row">
          <span class="idx">{{ i + 1 }}</span>
          <el-select v-model="b.type" style="width: 150px">
            <el-option v-for="t in BLOCK_TYPES" :key="t" :label="BLOCK_TYPE_TEXT[t]" :value="t" />
          </el-select>
          <el-input v-model="b.dataJson" type="textarea" :rows="2" class="data-input" placeholder='内容数据（JSON），如 {"title":"活动"}' />
          <el-button size="small" type="danger" link @click="blocks.splice(i, 1)">删除</el-button>
        </div>
      </el-tab-pane>

      <!-- 系统设置 -->
      <el-tab-pane label="系统设置" name="settings">
        <div class="tab-head">
          <span class="hint">键值设置（如门店电话、地址等，供业务读取）。</span>
        </div>
        <div class="setting-form">
          <el-input v-model="settingKey" placeholder="设置键，如 store_phone" style="width: 220px" />
          <el-input v-model="settingValue" placeholder="设置值" style="width: 260px" />
          <el-button type="primary" :loading="saving" @click="doSaveSetting">保存</el-button>
        </div>
        <el-table :data="settingRows" size="default" v-loading="loading">
          <el-table-column prop="key" label="键" width="220" />
          <el-table-column prop="value" label="值" min-width="240" />
          <el-table-column label="操作" width="80">
            <template #default="{ row }">
              <el-button size="small" @click="settingKey = row.key; settingValue = row.value">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>

  <el-dialog v-model="bannerVisible" :title="bannerForm.id ? '编辑轮播图' : '新建轮播图'" width="460px">
    <el-form label-width="80px">
      <el-form-item label="标题" required><el-input v-model="bannerForm.title" /></el-form-item>
      <el-form-item label="图片 URL" required><el-input v-model="bannerForm.image" /></el-form-item>
      <el-form-item label="跳转"><el-input v-model="bannerForm.link" /></el-form-item>
      <el-form-item label="排序"><el-input-number v-model="bannerForm.sort" :min="0" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="bannerVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveBannerForm">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="annVisible" :title="annForm.id ? '编辑公告' : '新建公告'" width="460px">
    <el-form label-width="80px">
      <el-form-item label="标题" required><el-input v-model="annForm.title" /></el-form-item>
      <el-form-item label="内容" required><el-input v-model="annForm.content" type="textarea" :rows="3" /></el-form-item>
      <el-form-item label="排序"><el-input-number v-model="annForm.sort" :min="0" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="annVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveAnnForm">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getSettings,
  getPageConfig,
  listAnnouncements,
  listBanners,
  saveAnnouncement,
  saveBanner,
  savePageConfig,
  saveSetting,
  setAnnouncementStatus,
  setBannerStatus,
  type Announcement,
  type Banner,
} from '../core/api/admin'

const BLOCK_TYPES = ['banner', 'announcement', 'service_list', 'activity', 'richtext'] as const
const BLOCK_TYPE_TEXT: Record<string, string> = {
  banner: '轮播图',
  announcement: '公告',
  service_list: '服务列表',
  activity: '活动',
  richtext: '富文本',
}

const tab = ref('banners')
const loading = ref(false)
const saving = ref(false)

const banners = ref<Banner[]>([])
const bannerVisible = ref(false)
const bannerForm = ref({ id: '', title: '', image: '', link: '', sort: 0 })

const announcements = ref<Announcement[]>([])
const annVisible = ref(false)
const annForm = ref({ id: '', title: '', content: '', sort: 0 })

interface EditableBlock {
  type: string
  dataJson: string
}
const blocks = ref<EditableBlock[]>([])

const settingRows = ref<{ key: string; value: string }[]>([])
const settingKey = ref('')
const settingValue = ref('')

async function load() {
  loading.value = true
  try {
    if (tab.value === 'banners') banners.value = (await listBanners()) ?? []
    else if (tab.value === 'announcements') announcements.value = (await listAnnouncements()) ?? []
    else if (tab.value === 'page') {
      const raw = await getPageConfig('home')
      blocks.value = (raw ?? []).map((b) => ({ type: b.type, dataJson: JSON.stringify(b.data) }))
    } else if (tab.value === 'settings') {
      const s = await getSettings()
      settingRows.value = Object.entries(s ?? {}).map(([key, value]) => ({ key, value }))
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

// ---------- banners ----------
function openBanner(row: Banner | null) {
  bannerForm.value = row
    ? { id: row.id, title: row.title, image: row.image, link: row.link, sort: row.sort }
    : { id: '', title: '', image: '', link: '', sort: 0 }
  bannerVisible.value = true
}

async function saveBannerForm() {
  if (!bannerForm.value.title || !bannerForm.value.image) {
    ElMessage.warning('标题与图片必填')
    return
  }
  saving.value = true
  try {
    await saveBanner({
      id: bannerForm.value.id || undefined,
      title: bannerForm.value.title,
      image: bannerForm.value.image,
      link: bannerForm.value.link,
      sort: bannerForm.value.sort,
    })
    ElMessage.success('已保存')
    bannerVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleBanner(row: Banner, on: boolean) {
  try {
    await setBannerStatus(row.id, on ? 'ACTIVE' : 'INACTIVE')
    row.status = on ? 'ACTIVE' : 'INACTIVE'
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

// ---------- announcements ----------
function openAnn(row: Announcement | null) {
  annForm.value = row
    ? { id: row.id, title: row.title, content: row.content, sort: row.sort }
    : { id: '', title: '', content: '', sort: 0 }
  annVisible.value = true
}

async function saveAnnForm() {
  if (!annForm.value.title || !annForm.value.content) {
    ElMessage.warning('标题与内容必填')
    return
  }
  saving.value = true
  try {
    await saveAnnouncement({
      id: annForm.value.id || undefined,
      title: annForm.value.title,
      content: annForm.value.content,
      sort: annForm.value.sort,
    })
    ElMessage.success('已保存')
    annVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleAnn(row: Announcement, on: boolean) {
  try {
    await setAnnouncementStatus(row.id, on ? 'ACTIVE' : 'INACTIVE')
    row.status = on ? 'ACTIVE' : 'INACTIVE'
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

// ---------- page blocks ----------
function addBlock() {
  blocks.value.push({ type: 'richtext', dataJson: '{}' })
}

async function saveBlocks() {
  const parsed: { type: string; data: Record<string, unknown> }[] = []
  for (const b of blocks.value) {
    try {
      parsed.push({ type: b.type, data: JSON.parse(b.dataJson || '{}') })
    } catch {
      ElMessage.error('内容块 JSON 格式有误，请检查')
      return
    }
  }
  saving.value = true
  try {
    await savePageConfig('home', parsed)
    ElMessage.success('布局已保存，用户端首页生效')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

// ---------- settings ----------
async function doSaveSetting() {
  if (!settingKey.value.trim()) {
    ElMessage.warning('请输入设置键')
    return
  }
  saving.value = true
  try {
    await saveSetting(settingKey.value.trim(), settingValue.value)
    ElMessage.success('已保存')
    settingKey.value = ''
    settingValue.value = ''
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.tab-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.hint {
  color: #909399;
  font-size: 12px;
}
.block-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 8px 0;
  border-bottom: 1px dashed #ebeef5;
}
.idx {
  width: 20px;
  color: #909399;
  line-height: 32px;
}
.data-input {
  flex: 1;
}
.setting-form {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}
</style>
