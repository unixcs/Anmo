import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import { setTokenProvider, setUnauthorizedHandler } from './core/api/http'
import { getToken, clearSession } from './platform/auth'

setTokenProvider(getToken)
setUnauthorizedHandler(() => {
  clearSession()
  if (router.currentRoute.value.path !== '/login') void router.push('/login')
})

createApp(App).use(router).use(ElementPlus, { locale: zhCn }).mount('#app')
