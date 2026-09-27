<template>
  <el-card shadow="never">
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="姓名 / 手机号" clearable style="width: 220px"
        @keyup.enter="search" />
      <el-button type="primary" @click="search">查询</el-button>
      <el-button @click="createVisible = true">新建会员</el-button>
      <span class="spacer" />
      <span class="total">共 {{ total }} 位</span>
    </div>

    <el-table :data="rows" v-loading="loading" @row-click="openDetail">
      <el-table-column prop="member_no" label="会员号" width="160" />
      <el-table-column prop="name" label="姓名" width="120" />
      <el-table-column prop="phone" label="手机号" width="130" />
      <el-table-column prop="gender" label="性别" width="70" />
      <el-table-column label="最近到店" width="150">
        <template #default="{ row }">{{ fmtTime(row.last_visit_at) }}</template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
      <el-table-column label="操作" width="90" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click.stop="openDetail(row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination class="pager" background layout="prev, pager, next, total" :total="total"
      :page-size="perPage" :current-page="page" @current-change="onPage" />
  </el-card>

  <!-- 新建会员 -->
  <el-dialog v-model="createVisible" title="新建会员" width="440px">
    <el-form label-width="80px">
      <el-form-item label="姓名" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="手机号" required><el-input v-model="form.phone" maxlength="11" /></el-form-item>
      <el-form-item label="性别">
        <el-select v-model="form.gender" style="width: 120px">
          <el-option label="女" value="女" />
          <el-option label="男" value="男" />
        </el-select>
      </el-form-item>
      <el-form-item label="生日"><el-date-picker v-model="form.birthday" type="date"
        value-format="YYYY-MM-DD" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="doCreate">保存</el-button>
    </template>
  </el-dialog>

  <!-- 会员详情抽屉 -->
  <el-drawer v-model="detailVisible" size="640px" :title="detail ? `${detail.member.name}（${detail.member.member_no}）` : ''">
    <template v-if="detail">
      <h4 class="sec">基本资料</h4>
      <el-form label-width="80px" size="small">
        <el-form-item label="手机号"><el-input v-model="detail.member.phone" disabled /></el-form-item>
        <el-form-item label="姓名"><el-input v-model="edit.name" /></el-form-item>
        <el-form-item label="性别">
          <el-select v-model="edit.gender" style="width: 120px">
            <el-option label="女" value="女" /><el-option label="男" value="男" />
          </el-select>
        </el-form-item>
        <el-form-item label="生日"><el-date-picker v-model="edit.birthday" type="date"
          value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="edit.remark" /></el-form-item>
        <el-form-item label="标签">
          <el-select v-model="edit.tagIds" multiple placeholder="选择标签" style="width: 100%">
            <el-option v-for="t in tags" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="small" :loading="saving" @click="saveProfile">保存资料</el-button>
        </el-form-item>
      </el-form>

      <div class="sec-head">
        <h4 class="sec">会员卡</h4>
        <el-button size="small" type="primary" @click="issueVisible = true">开卡</el-button>
      </div>
      <el-table :data="cards" size="small">
        <el-table-column label="模板" min-width="110">
          <template #default="{ row }">{{ templateName(row.card_template_id) }}</template>
        </el-table-column>
        <el-table-column label="剩余/总" width="80">
          <template #default="{ row }">{{ row.remaining_count }}/{{ row.total_count }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">{{ CARD_STATUS_TEXT[row.status] ?? row.status }}</template>
        </el-table-column>
        <el-table-column label="有效期" width="110">
          <template #default="{ row }">{{ row.valid_until ?? '永久' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="doAdjust(row)">调整</el-button>
            <el-button size="small" link type="primary" @click="showTx(row)">流水</el-button>
            <el-button v-if="row.status === 'ACTIVE'" size="small" link type="danger"
              @click="doCancelCard(row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </el-drawer>

  <!-- 开卡 -->
  <el-dialog v-model="issueVisible" title="为该会员开卡" width="420px">
    <el-form label-width="90px">
      <el-form-item label="卡模板" required>
        <el-select v-model="issueTemplateId" style="width: 100%" placeholder="选择已上架模板">
          <el-option v-for="t in templates.filter((x) => x.status === 'ACTIVE')" :key="t.id"
            :label="`${t.name}（${CARD_TYPE_TEXT[t.type]} ${t.total_count}次 ${yuan(t.price)}）`" :value="t.id" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="issueVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="doIssue">确认开卡</el-button>
    </template>
  </el-dialog>

  <!-- 卡流水 -->
  <el-dialog v-model="txVisible" title="次数流水" width="620px">
    <el-table :data="txRows" size="small" max-height="420">
      <el-table-column label="类型" width="90">
        <template #default="{ row }">{{ CARD_TX_TYPE_TEXT[row.type] ?? row.type }}</template>
      </el-table-column>
      <el-table-column prop="quantity" label="±次数" width="70" />
      <el-table-column prop="before_count" label="变前" width="60" />
      <el-table-column prop="after_count" label="变后" width="60" />
      <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
      <el-table-column label="时间" width="150">
        <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  adjustCard,
  cancelCard,
  createMember,
  getMember,
  issueCard,
  listCardTransactions,
  listCardTemplates,
  listMemberCards,
  listMembers,
  listTags,
  setMemberTags,
  updateMember,
  type Member,
  type MemberCard,
  type CardTransaction,
  type CardTemplate,
  type Tag,
} from '../core/api/admin'
import { CARD_STATUS_TEXT, CARD_TX_TYPE_TEXT, CARD_TYPE_TEXT, fmtTime, yuan } from '../core/format'

const keyword = ref('')
const page = ref(1)
const perPage = 20
const total = ref(0)
const rows = ref<Member[]>([])
const loading = ref(false)
const saving = ref(false)

const createVisible = ref(false)
const form = ref({ name: '', phone: '', gender: '女', birthday: '' as string | '', remark: '' })

const detailVisible = ref(false)
const detail = ref<{ member: Member; tags: Tag[] } | null>(null)
const edit = ref({ name: '', gender: '', birthday: '' as string | '', remark: '', tagIds: [] as string[] })
const tags = ref<Tag[]>([])
const cards = ref<MemberCard[]>([])
const templates = ref<CardTemplate[]>([])
const templateMap = ref<Record<string, string>>({})

const issueVisible = ref(false)
const issueTemplateId = ref('')
const txVisible = ref(false)
const txRows = ref<CardTransaction[]>([])

function templateName(id: string): string {
  return templateMap.value[id] ?? id.slice(0, 8)
}

function onPage(p: number) {
  page.value = p
  void load()
}

function search() {
  page.value = 1
  void load()
}

async function load() {
  loading.value = true
  try {
    const res = await listMembers(keyword.value.trim(), page.value, perPage)
    rows.value = res.data
    total.value = res.total
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function doCreate() {
  if (!form.value.name || !form.value.phone) {
    ElMessage.warning('姓名与手机号必填')
    return
  }
  saving.value = true
  try {
    await createMember({
      name: form.value.name,
      phone: form.value.phone,
      gender: form.value.gender,
      birthday: form.value.birthday || null,
      remark: form.value.remark,
    })
    ElMessage.success('已创建')
    createVisible.value = false
    form.value = { name: '', phone: '', gender: '女', birthday: '', remark: '' }
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    saving.value = false
  }
}

async function openDetail(row: Member) {
  try {
    const [d, t, tpl] = await Promise.all([getMember(row.id), listTags(), listCardTemplates()])
    detail.value = d
    tags.value = t ?? []
    templates.value = tpl ?? []
    templateMap.value = Object.fromEntries((tpl ?? []).map((x) => [x.id, x.name]))
    edit.value = {
      name: d.member.name,
      gender: d.member.gender,
      birthday: d.member.birthday ?? '',
      remark: d.member.remark,
      tagIds: (d.tags ?? []).map((x) => x.id),
    }
    cards.value = (await listMemberCards(row.id)) ?? []
    detailVisible.value = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  }
}

async function refreshCards() {
  if (detail.value) cards.value = await listMemberCards(detail.value.member.id)
}

async function saveProfile() {
  if (!detail.value) return
  saving.value = true
  try {
    await Promise.all([
      updateMember(detail.value.member.id, {
        name: edit.value.name,
        gender: edit.value.gender,
        birthday: edit.value.birthday || null,
        remark: edit.value.remark,
      }),
      setMemberTags(detail.value.member.id, edit.value.tagIds),
    ])
    ElMessage.success('已保存')
    detailVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function doIssue() {
  if (!detail.value || !issueTemplateId.value) {
    ElMessage.warning('请选择卡模板')
    return
  }
  saving.value = true
  try {
    await issueCard(detail.value.member.id, issueTemplateId.value)
    ElMessage.success('开卡成功')
    issueVisible.value = false
    issueTemplateId.value = ''
    await refreshCards()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '开卡失败')
  } finally {
    saving.value = false
  }
}

async function doAdjust(card: MemberCard) {
  let value = ''
  try {
    const res = await ElMessageBox.prompt(
      `当前剩余 ${card.remaining_count} 次。输入调整量（正数增加，负数扣减，如 -1 / +2）`,
      '调整次数',
      { inputPattern: /^[+-]?\d+$/, inputErrorMessage: '请输入整数（可带正负号）' },
    )
    value = res.value.trim()
  } catch {
    return
  }
  let remark = ''
  try {
    const res = await ElMessageBox.prompt('调整备注', '调整次数', { inputValue: '手工调整' })
    remark = res.value.trim()
  } catch {
    return
  }
  try {
    await adjustCard(card.id, Number(value), remark)
    ElMessage.success('已调整')
    await refreshCards()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '调整失败')
  }
}

async function doCancelCard(card: MemberCard) {
  try {
    await ElMessageBox.confirm(
      `作废后剩余 ${card.remaining_count} 次清零且不可恢复，确认作废？`,
      '作废会员卡',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await cancelCard(card.id)
    ElMessage.success('已作废')
    await refreshCards()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '作废失败')
  }
}

async function showTx(card: MemberCard) {
  try {
    txRows.value = await listCardTransactions(card.id)
    txVisible.value = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载流水失败')
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
.spacer {
  flex: 1;
}
.total {
  color: #909399;
  font-size: 13px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
.sec {
  margin: 6px 0 10px;
  color: #303133;
}
.sec-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
:deep(.el-drawer__body) {
  padding-top: 6px;
}
</style>
