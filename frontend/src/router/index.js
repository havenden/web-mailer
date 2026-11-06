import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', component: () => import('../views/Login.vue') },
  {
    path: '/',
    component: () => import('../layout/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: '仪表盘', component: () => import('../views/Dashboard.vue'), meta: { icon: 'Odometer' } },
      { path: 'users', name: '用户管理', component: () => import('../views/Users.vue'), meta: { icon: 'User' } },
      { path: 'interfaces', name: '接口管理', component: () => import('../views/Interfaces.vue'), meta: { icon: 'Link' } },
      { path: 'mail-configs', name: '邮件配置', component: () => import('../views/MailConfigs.vue'), meta: { icon: 'Message' } },
      { path: 'push-records', name: '接收记录', component: () => import('../views/PushRecords.vue'), meta: { icon: 'Document' } },
      { path: 'mail-records', name: '邮件记录', component: () => import('../views/MailRecords.vue'), meta: { icon: 'Promotion' } },
      { path: 'logs', name: '操作日志', component: () => import('../views/Logs.vue'), meta: { icon: 'List' } }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.path !== '/login' && !token) return '/login'
  if (to.path === '/login' && token) return '/dashboard'
  return true
})

export default router
