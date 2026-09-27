<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Catalog } from '../core/api/endpoints'
import { yuan } from '../core/utils/format'

const catalog = ref<Catalog>({ categories: [], services: [] })

onMounted(async () => {
  catalog.value = await api.catalog()
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
    <h1>服务项目</h1>
    <div v-for="cat in catalog.categories" :key="cat.id" class="group">
      <h2>{{ cat.name }}</h2>
      <div v-for="s in byCategory()[cat.id]" :key="s.id" class="item">
        <div>
          <div class="name">{{ s.name }}</div>
          <div class="desc">{{ s.description }}</div>
          <div class="meta">{{ s.duration_minutes }} 分钟 · ¥{{ yuan(s.default_price) }}</div>
        </div>
        <RouterLink :to="`/booking?service=${s.id}`" class="book">预约</RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.services { padding: 20px 16px; }
h1 { font-size: 20px; }
.group h2 { font-size: 15px; color: var(--muted-foreground); margin: 18px 0 8px; }
.item { background: var(--card); border-radius: 12px; padding: 14px; display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.name { font-weight: 600; }
.desc { color: var(--muted-foreground); font-size: 13px; margin: 4px 0; }
.meta { color: var(--primary); font-size: 14px; }
.book { background: var(--primary); color: var(--card); text-decoration: none; padding: 8px 16px; border-radius: 8px; font-size: 14px; }
</style>
