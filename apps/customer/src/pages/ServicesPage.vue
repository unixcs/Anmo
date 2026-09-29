<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Catalog } from '../core/api/endpoints'
import { yuan } from '../core/utils/format'
import AppIcon from '../components/ui/AppIcon.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const catalog = ref<Catalog>({ categories: [], services: [] })
const loading = ref(true)

onMounted(async () => {
  try {
    catalog.value = await api.catalog()
  } finally {
    loading.value = false
  }
})

function byCategory(): Record<string, typeof catalog.value.services> {
  const map: Record<string, typeof catalog.value.services> = {}
  for (const c of catalog.value.categories) map[c.id] = []
  map[''] = catalog.value.services.filter((s) => !catalog.value.categories.some((c) => c.id === s.category_id))
  for (const s of catalog.value.services) {
    if (map[s.category_id]) map[s.category_id].push(s)
  }
  return map
}
</script>

<template>
  <div class="page services">
    <div class="page-head">
      <h1>服务项目</h1>
      <p class="sub">选好项目，选个舒服的时间</p>
    </div>

    <AppSkeleton v-if="loading" variant="card" />

    <AppEmpty
      v-else-if="!catalog.services.length"
      icon="inbox"
      main="暂无服务项目"
      sub="商家上架后会显示在这里"
    />

    <template v-else>
      <section v-for="cat in catalog.categories" :key="cat.id" class="group">
        <h2>{{ cat.name }}</h2>
        <div v-for="s in byCategory()[cat.id]" :key="s.id" class="card svc">
          <div class="svc-body">
            <div class="name">{{ s.name }}</div>
            <div v-if="s.description" class="desc">{{ s.description }}</div>
            <div class="meta">
              <span class="num">{{ s.duration_minutes }} 分钟</span>
              <span class="dot-sep">·</span>
              <span class="price money">¥{{ yuan(s.default_price) }}</span>
            </div>
          </div>
          <RouterLink :to="`/booking?service=${s.id}`" class="btn secondary sm pressable">
            预约
            <AppIcon name="arrow-right" :size="14" />
          </RouterLink>
        </div>
      </section>
      <section v-if="byCategory()['']?.length" class="group">
        <h2>其他</h2>
        <div v-for="s in byCategory()['']" :key="s.id" class="card svc">
          <div class="svc-body">
            <div class="name">{{ s.name }}</div>
            <div v-if="s.description" class="desc">{{ s.description }}</div>
            <div class="meta">
              <span class="num">{{ s.duration_minutes }} 分钟</span>
              <span class="dot-sep">·</span>
              <span class="price money">¥{{ yuan(s.default_price) }}</span>
            </div>
          </div>
          <RouterLink :to="`/booking?service=${s.id}`" class="btn secondary sm pressable">
            预约
            <AppIcon name="arrow-right" :size="14" />
          </RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.group h2 {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 20px 0 10px;
}

.group:first-of-type h2 {
  margin-top: 0;
}

.svc {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.svc-body {
  flex: 1;
  min-width: 0;
}

.name {
  font-weight: 600;
}

.desc {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 3px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.meta {
  font: var(--font-sub);
  color: var(--muted-foreground);
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot-sep {
  opacity: 0.6;
}

.price {
  color: var(--primary);
}
</style>
