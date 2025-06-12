import { createRouter, createWebHistory } from 'vue-router'
import StockList   from '../components/StockList.vue'
import StockDetail from '../components/StockDetail.vue'
import Recommend from '../components/Recommend.vue'

const routes = [
  { path: '/',                 name: 'StockList',   component: StockList },
  { path: '/stocks/:ticker',   name: 'StockDetail', component: StockDetail, props: true },
  {path:'/recommend',          name: 'Recommend', component : Recommend}
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
