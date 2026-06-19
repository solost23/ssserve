<script setup>
import { ref } from 'vue'
import { changePassword } from '../api'
import { showToast } from '../toast'

const emit = defineEmits(['close'])

const current = ref('')
const next = ref('')
const confirm = ref('')
const err = ref('')
const loading = ref(false)

async function submit() {
  err.value = ''
  if (!current.value || !next.value) { err.value = '请填写所有字段'; return }
  if (next.value !== confirm.value) { err.value = '两次新密码不一致'; return }
  if (next.value.length < 6) { err.value = '新密码至少 6 位'; return }
  loading.value = true
  try {
    await changePassword(current.value, next.value)
    showToast('密码已修改')
    emit('close')
  } catch (e) {
    err.value = e.message.includes('incorrect') ? '当前密码错误' : e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="modal">
      <h3>修改密码</h3>
      <div class="field">
        <label>当前密码</label>
        <input v-model="current" type="password" placeholder="••••••••" @keydown.escape="emit('close')" />
      </div>
      <div class="field">
        <label>新密码</label>
        <input v-model="next" type="password" placeholder="••••••••" />
      </div>
      <div class="field">
        <label>确认新密码</label>
        <input v-model="confirm" type="password" placeholder="••••••••" @keydown.enter="submit" />
      </div>
      <p v-if="err" class="err">{{ err }}</p>
      <div class="actions">
        <button class="btn-outline" @click="emit('close')">取消</button>
        <button class="btn-primary" :disabled="loading" @click="submit">
          {{ loading ? '保存中…' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,.35);
  display: flex; align-items: center; justify-content: center;
  z-index: 50;
}
.modal {
  background: #fff; border-radius: 12px;
  padding: 28px 28px 24px; width: 360px;
  box-shadow: 0 8px 32px rgba(0,0,0,.12);
}
h3 { font-size: 15px; font-weight: 700; margin-bottom: 18px; }
.field { margin-bottom: 13px; }
.field label { display: block; font-size: 12px; font-weight: 500; color: #64748b; margin-bottom: 5px; }
.err { color: #ef4444; font-size: 12.5px; margin: -4px 0 10px; }
.actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
</style>
