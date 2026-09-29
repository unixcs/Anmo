<template>
  <!-- 高德地图选点（V2.2 R2）：搜索结果点选回填 / 拖拽标记选点，点「确认选点」才回填 -->
  <el-dialog :model-value="modelValue" title="地图选点" width="min(760px, 96vw)"
    @update:model-value="onVisible" @opened="onOpened" @closed="onClosed">
    <div class="picker-toolbar">
      <el-input v-model="keyword" placeholder="输入门店地址关键词搜索，如：XX市XX区XX路12号" style="flex: 1"
        @keyup.enter="doSearch" />
      <el-button type="primary" :loading="searching" @click="doSearch">搜索</el-button>
      <span v-if="picked" class="picked">{{ picked }}</span>
    </div>
    <div v-if="pois.length" class="poi-list">
      <div v-for="(p, i) in pois" :key="i" class="poi-row" @click="choosePoi(p)">
        <span class="poi-name">{{ p.name }}</span>
        <span class="poi-addr">{{ districtOf(p) }} {{ String(p.address ?? '') }}</span>
      </div>
    </div>
    <div ref="mapEl" class="map-box" />
    <template #footer>
      <el-button @click="onCancel">取消</el-button>
      <el-button type="primary" @click="onConfirm">确认选点</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

// AMap JS API 无官方类型定义，地图交互处统一用 any 收敛
const props = defineProps<{
  modelValue: boolean
  apiKey: string
  jsCode: string
  initLng?: string
  initLat?: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm', p: { lng: string; lat: string }): void
  (e: 'cancel'): void
}>()

interface Poi {
  name: string
  address: string
  district?: string
  pname?: string
  cityname?: string
  adname?: string
  location: { lng: number; lat: number }
}

// ---- AMap 脚本加载器：promise 缓存防重复加载；jsCode 有才设安全密钥 ----
let amapPromise: Promise<any> | null = null

function loadAmap(key: string, jsCode: string): Promise<any> {
  const w = window as any
  if (w.AMap) return Promise.resolve(w.AMap)
  if (amapPromise) return amapPromise
  if (jsCode) w._AMapSecurityConfig = { securityJsCode: jsCode }
  amapPromise = new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = `https://webapi.amap.com/maps?v=2.0&key=${encodeURIComponent(key)}`
    s.onload = () => {
      if (w.AMap) resolve(w.AMap)
      else reject(new Error('AMap 全局对象缺失'))
    }
    s.onerror = () => {
      amapPromise = null // 允许下次重试
      reject(new Error('高德地图脚本加载失败'))
    }
    document.head.appendChild(s)
  })
  return amapPromise
}

const mapEl = ref<HTMLDivElement | null>(null)
const keyword = ref('')
const searching = ref(false)
const pois = ref<Poi[]>([])
const picked = ref('')

let map: any = null
let marker: any = null

// 初始中心：有已存坐标用之，否则默认杭州
function centerOf(): [number, number] {
  const lng = Number(props.initLng)
  const lat = Number(props.initLat)
  if (props.initLng && props.initLat && Number.isFinite(lng) && Number.isFinite(lat)) return [lng, lat]
  return [120.153576, 30.287459]
}

async function onOpened(): Promise<void> {
  keyword.value = ''
  pois.value = []
  picked.value = ''
  try {
    const AMap = await loadAmap(props.apiKey, props.jsCode)
    if (!mapEl.value) return
    map = new AMap.Map(mapEl.value, { zoom: 12, center: centerOf() })
    map.on('click', (e: any) => placeMarker(e.lnglat))
    // 已有坐标时先落一个可拖拽标记；否则等点击/搜索再落
    if (props.initLng && props.initLat) marker = new AMap.Marker({ position: centerOf(), draggable: true, map })
  } catch {
    ElMessage.error('高德地图加载失败，请检查 Key 配置或手动输入经纬度')
  }
}

function onClosed(): void {
  if (map) {
    map.destroy()
    map = null
  }
  marker = null
  pois.value = []
}

function onVisible(v: boolean): void {
  emit('update:modelValue', v)
}

function onCancel(): void {
  emit('cancel')
  onVisible(false)
}

function placeMarker(lnglat: any): void {
  const AMap = (window as any).AMap
  if (!map || !AMap) return
  const pos = lnglat && lnglat.getLng ? lnglat : new AMap.LngLat(lnglat.lng, lnglat.lat)
  if (!marker) marker = new AMap.Marker({ draggable: true, map })
  marker.setPosition(pos)
  picked.value = `${Number(pos.getLng()).toFixed(6)}, ${Number(pos.getLat()).toFixed(6)}`
}

function onConfirm(): void {
  const pos = marker && marker.getPosition()
  if (!pos) {
    ElMessage.warning('请先点击地图或选择搜索结果选点')
    return
  }
  emit('confirm', { lng: Number(pos.getLng()).toFixed(6), lat: Number(pos.getLat()).toFixed(6) })
  onVisible(false)
}

function doSearch(): void {
  const kw = keyword.value.trim()
  if (!kw) {
    ElMessage.warning('请输入地址关键词')
    return
  }
  const AMap = (window as any).AMap
  if (!AMap || !map) return
  searching.value = true
  // 插件加载失败时 AMap.plugin 回调不会触发：看门狗兜底解锁按钮并提示拖拽选点
  let settled = false
  const watchdog = window.setTimeout(() => {
    if (settled) return
    settled = true
    searching.value = false
    ElMessage.warning('搜索服务暂不可用，请拖拽地图手动选点')
  }, 10000)
  AMap.plugin('AMap.PlaceSearch', () => {
    if (settled) return
    settled = true
    window.clearTimeout(watchdog)
    searching.value = false
    const ps = new AMap.PlaceSearch({ city: '全国', extensions: 'base' })
    ps.search(kw, (status: string, result: any) => {
      if (status === 'complete' && result?.poiList?.pois?.length) {
        pois.value = result.poiList.pois as Poi[]
      } else {
        pois.value = []
        ElMessage.warning('未找到匹配地址，请拖拽地图手动选点')
      }
    })
  })
}

function choosePoi(p: Poi): void {
  if (!map || !p.location) return
  map.setCenter(p.location)
  placeMarker(p.location)
}

function districtOf(p: Poi): string {
  if (p.district) return p.district
  return [p.pname, p.cityname, p.adname].filter(Boolean).join('')
}
</script>

<style scoped>
.picker-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.picked {
  color: #67c23a;
  font-size: 12px;
  white-space: nowrap;
}
.poi-list {
  max-height: 160px;
  overflow-y: auto;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  margin-bottom: 10px;
}
.poi-row {
  display: flex;
  gap: 10px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
}
.poi-row:hover {
  background: #f5f7fa;
}
.poi-name {
  font-weight: 600;
  white-space: nowrap;
}
.poi-addr {
  color: #909399;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.map-box {
  height: 380px;
  width: 100%;
}
</style>
