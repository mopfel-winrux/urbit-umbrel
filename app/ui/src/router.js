import { createRouter, createWebHistory } from 'vue-router'
import Login from './views/Login.vue'
import Home  from './views/Home.vue'
import { getStatus } from './api'
import { isAuthed } from './util.js'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/:pathMatch(.*)*', component: Home }
  ]
})

router.beforeEach(async (to, _from, next) => {
  if (to.path === '/login') return next()
  if (!isAuthed()) return next('/login')
  try { await getStatus(); next() } catch { next('/login') }
})

export default router
