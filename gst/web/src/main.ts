import { createApp } from 'vue'
import {
  ElButton,
  ElDialog,
  ElLoading,
  ElPagination,
  ElSkeleton,
  ElTable,
  ElTableColumn,
} from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'

createApp(App)
  .use(ElButton)
  .use(ElDialog)
  .use(ElLoading)
  .use(ElPagination)
  .use(ElSkeleton)
  .use(ElTable)
  .use(ElTableColumn)
  .mount('#app')
