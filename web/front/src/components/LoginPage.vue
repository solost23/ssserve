<script setup>
import { ref } from 'vue'
import { login } from '../api'
import { setAuth } from '../auth'

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  if (!username.value || !password.value) { error.value = '请输入用户名和密码'; return }
  loading.value = true
  try {
    const data = await login(username.value, password.value)
    setAuth(data.token, data.username, data.is_owner)
  } catch {
    error.value = '用户名或密码错误'
  } finally {
    loading.value = false
  }
}

function onKey(e) { if (e.key === 'Enter') submit() }
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">⚡</div>
      <h1 class="login-title">Sub Manager</h1>
      <p class="login-sub">订阅管理后台</p>

      <div class="field">
        <label>用户名</label>
        <input v-model="username" type="text" placeholder="admin" autocomplete="username" @keydown="onKey" />
      </div>
      <div class="field">
        <label>密码</label>
        <input v-model="password" type="password" placeholder="••••••••" autocomplete="current-password" @keydown="onKey" />
      </div>

      <p v-if="error" class="login-error">{{ error }}</p>

      <button class="btn-primary login-btn" :disabled="loading" @click="submit">
        {{ loading ? '登录中…' : '登录' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef2ff 0%, #f1f5f9 100%);
}
.login-card {
  background: #fff;
  border-radius: 14px;
  padding: 40px 36px;
  width: 360px;
  box-shadow: 0 4px 24px rgba(0,0,0,.08);
  text-align: center;
}
.login-logo { font-size: 32px; margin-bottom: 8px; }
.login-title { font-size: 20px; font-weight: 700; color: #1e293b; margin-bottom: 4px; }
.login-sub { font-size: 13px; color: #94a3b8; margin-bottom: 28px; }

.field { text-align: left; margin-bottom: 14px; }
.field label { display: block; font-size: 12.5px; font-weight: 500; color: #475569; margin-bottom: 5px; }

.login-error { color: #ef4444; font-size: 12.5px; margin: -4px 0 12px; }
.login-btn { width: 100%; padding: 10px; font-size: 14px; margin-top: 4px; }
</style>
