<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { notify } from '../platform/notify/toast'
import AppStatusBadge from '../components/ui/AppStatusBadge.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'

interface AptItem extends Appointment {
  serviceNames: string
}

const list = ref<AptItem[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const arr = await api.myAppointments('COMPLETED')
    list.value = [...(arr ?? [])]
      .sort((a, b) => b.scheduled_start.localeCompare(a.scheduled_start))
      .map((a) => {
        const item = a as Appointment & { services?: { service_name_snapshot: string }[] }
        return {
          ...a,
          serviceNames: (item.services ?? []).map((s) => s.service_name_snapshot).filter(Boolean).join(' · '),
        }
      })
  } catch (e) {
    notify((e as Error).message)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page history">
    <div class="page-head">
      <h1>历史服务</h1>
      <p class="sub">已完成的服务记录</p>
    </div>

    <AppSkeleton v-if="loading" variant="card" />

    <AppEmpty
      v-else-if="!list.length"
      icon="history"
      main="还没有已完成的服务"
      sub="做完第一次服务后就会出现在这里"
    >
      <RouterLink to="/booking" class="btn secondary sm">去预约</RouterLink>
    </AppEmpty>

    <div v-else class="card plain tx-list">
      <div v-for="a in list" :key="a.id" class="row">
        <div class="row-body">
          <div class="row-top">
            <span class="time num">{{ a.scheduled_start.slice(0, 16).replace('T', ' ') }}</span>
            <AppStatusBadge :status="a.status" />
          </div>
          <div v-if="a.serviceNames" class="svc">{{ a.serviceNames }}</div>
          <div class="no num">{{ a.appointment_no }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tx-list {
  padding: 4px 16px;
}

.row {
  padding: 12px 0;
}

.row + .row {
  border-top: 1px solid var(--border);
}

.row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.time {
  font-weight: 600;
}

.svc {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-top: 4px;
}

.no {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin-top: 2px;
}
</style>
