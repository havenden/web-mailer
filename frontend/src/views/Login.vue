<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">📮 Web-Mailer</div>
      <div class="login-sub">接口数据接收与邮件推送管理系统</div>
      <a-form @finish="onLogin">
        <a-form-item>
          <a-input v-model:value="form.username" placeholder="用户名" size="large">
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-input-password v-model:value="form.password" placeholder="密码" size="large" @keyup.enter="onLogin">
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" size="large" block html-type="submit" :loading="loading">登 录</a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined } from '@ant-design/icons-vue'
import api from '../api'

const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function onLogin() {
  if (!form.username || !form.password) {
    message.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const data = await api.post('/auth/login', form)
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    router.push('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page { height: 100%; display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #1f4e79, #2d6da3); }
.login-card { width: 380px; background: #fff; border-radius: 12px; padding: 40px 36px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
.login-logo { text-align: center; font-size: 26px; font-weight: bold; color: #1f4e79; }
.login-sub { text-align: center; color: #999; font-size: 13px; margin: 8px 0 28px; }
@media (max-width: 480px) {
  .login-card { width: calc(100% - 32px); }
}
</style>
