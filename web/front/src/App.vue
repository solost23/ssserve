<script setup>
import { auth, clearAuth } from './auth'
import LoginPage from './components/LoginPage.vue'
import TokensTab from './components/TokensTab.vue'
import AdminsTab from './components/AdminsTab.vue'
import ToastMsg from './components/ToastMsg.vue'
import { ref } from 'vue'

const tab = ref('tokens')
</script>

<template>
  <LoginPage v-if="!auth.loggedIn" />

  <template v-else>
    <header class="app-header">
      <div class="header-inner">
        <div class="header-brand">
          <span class="brand-icon">⚡</span>
          <span class="brand-title">Sub Manager</span>
        </div>
        <div class="header-user">
          <span class="user-chip">{{ auth.user }}</span>
          <button class="btn-outline btn-sm" @click="clearAuth">退出</button>
        </div>
      </div>
    </header>

    <main class="app-main">
      <div class="container">
        <nav class="tab-nav">
          <button
            v-for="t in (auth.owner ? ['tokens','admins'] : ['tokens'])"
            :key="t"
            class="tab-btn"
            :class="{ active: tab === t }"
            @click="tab = t"
          >{{ t === 'tokens' ? 'Tokens' : 'Admins' }}</button>
        </nav>

        <TokensTab v-show="tab === 'tokens'" />
        <AdminsTab v-if="auth.owner" v-show="tab === 'admins'" />
      </div>
    </main>
  </template>

  <ToastMsg />
</template>

<style scoped>
.app-header {
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  position: sticky;
  top: 0;
  z-index: 10;
}
.header-inner {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-brand { display: flex; align-items: center; gap: 8px; }
.brand-icon { font-size: 18px; }
.brand-title { font-weight: 700; font-size: 15px; color: #1e293b; }
.header-user { display: flex; align-items: center; gap: 10px; }
.user-chip {
  background: #f1f5f9; color: #475569;
  padding: 3px 10px; border-radius: 999px; font-size: 12.5px; font-weight: 500;
}
.btn-sm { padding: 4px 12px; font-size: 12.5px; }

.app-main { padding: 28px 0 60px; }
.container { max-width: 1100px; margin: 0 auto; padding: 0 24px; }

.tab-nav { display: flex; gap: 2px; margin-bottom: 20px; }
.tab-btn {
  background: transparent; color: #64748b;
  padding: 7px 16px; border-radius: 6px; font-size: 13.5px; font-weight: 500;
}
.tab-btn.active { background: #eef2ff; color: #4f46e5; }
.tab-btn:hover:not(.active) { background: #f8fafc; }
</style>
