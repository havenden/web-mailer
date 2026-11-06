<template>
  <a-layout class="layout">
    <a-layout-sider v-model:collapsed="isCollapse" :trigger="null" collapsible :width="220" class="sider">
      <div class="logo">
        <span class="logo-icon">📮</span>
        <span v-show="!isCollapse" class="logo-text">Web-Mailer</span>
      </div>
      <a-menu
        v-model:selectedKeys="selectedKeys"
        theme="dark"
        :items="menuItems"
        @click="onMenuClick"
      />
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="header">
        <div class="header-left">
          <component :is="isCollapse ? MenuUnfoldOutlined : MenuFoldOutlined" class="collapse-btn" @click="isCollapse = !isCollapse" />
          <a-breadcrumb separator="/">
            <a-breadcrumb-item>Web-Mailer</a-breadcrumb-item>
            <a-breadcrumb-item>{{ $route.name }}</a-breadcrumb-item>
          </a-breadcrumb>
        </div>
        <div class="header-right">
          <a-dropdown>
            <span class="user-info">
              <a-avatar :size="30" style="background:#1677ff">{{ (user.nickname || user.username || 'A').slice(0,1) }}</a-avatar>
              <span class="username">{{ user.nickname || user.username }}</span>
            </span>
            <template #overlay>
              <a-menu @click="onCommand">
                <a-menu-item key="password">修改密码</a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout">退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>

      <a-layout-content class="main">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>

  <a-modal v-model:open="pwdVisible" title="修改密码" @ok="changePwd" :confirm-loading="pwdLoading">
    <a-form :label-col="{ span: 6 }">
      <a-form-item label="原密码">
        <a-input-password v-model:value="pwdForm.old_password" />
      </a-form-item>
      <a-form-item label="新密码">
        <a-input-password v-model:value="pwdForm.new_password" placeholder="至少 6 位" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  MenuUnfoldOutlined, MenuFoldOutlined, DashboardOutlined, UserOutlined,
  LinkOutlined, MessageOutlined, FileTextOutlined, MailOutlined, UnorderedListOutlined
} from '@ant-design/icons-vue'
import api from '../api'

const router = useRouter()
const route = useRoute()
const isCollapse = ref(window.innerWidth < 768)
const user = ref(JSON.parse(localStorage.getItem('user') || '{}'))
const selectedKeys = ref([route.path])

const iconMap = { DashboardOutlined, UserOutlined, LinkOutlined, MessageOutlined, FileTextOutlined, MailOutlined, UnorderedListOutlined }

const menus = [
  { path: '/dashboard', name: '仪表盘', icon: 'DashboardOutlined' },
  { path: '/users', name: '用户管理', icon: 'UserOutlined' },
  { path: '/interfaces', name: '接口管理', icon: 'LinkOutlined' },
  { path: '/mail-configs', name: '邮件配置', icon: 'MessageOutlined' },
  { path: '/push-records', name: '接收记录', icon: 'FileTextOutlined' },
  { path: '/mail-records', name: '邮件记录', icon: 'MailOutlined' },
  { path: '/logs', name: '操作日志', icon: 'UnorderedListOutlined' }
]

const menuItems = computed(() => menus.map(m => ({
  key: m.path,
  label: m.name,
  icon: () => h(iconMap[m.icon])
})))

function onMenuClick({ key }) { router.push(key) }

const onResize = () => { isCollapse.value = window.innerWidth < 768 }
onMounted(() => window.addEventListener('resize', onResize))
onBeforeUnmount(() => window.removeEventListener('resize', onResize))

const pwdVisible = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '' })

function onCommand({ key }) {
  if (key === 'logout') {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    router.push('/login')
  } else if (key === 'password') {
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdVisible.value = true
  }
}

async function changePwd() {
  if (!pwdForm.old_password || pwdForm.new_password.length < 6) {
    message.warning('请填写原密码，新密码至少 6 位')
    return
  }
  pwdLoading.value = true
  try {
    await api.put('/auth/password', pwdForm)
    message.success('密码修改成功，请重新登录')
    pwdVisible.value = false
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    router.push('/login')
  } finally { pwdLoading.value = false }
}
</script>

<style scoped>
.layout { height: 100%; }
.sider { background: #001529; }
.logo { height: 56px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; font-weight: bold; }
.logo-icon { margin-right: 8px; }
.header { background: #fff; display: flex; align-items: center; justify-content: space-between; box-shadow: 0 1px 4px rgba(0,0,0,0.08); height: 56px; padding: 0 24px; }
.header-left { display: flex; align-items: center; gap: 16px; }
.collapse-btn { font-size: 18px; cursor: pointer; }
.user-info { display: flex; align-items: center; gap: 8px; cursor: pointer; }
.username { font-size: 14px; }
.main { background: #f0f2f5; padding: 16px; overflow: auto; }

@media (max-width: 768px) {
  .sider { position: fixed; z-index: 100; height: 100%; }
  .username { display: none; }
}
</style>
