<script setup>
import { ref, onMounted } from 'vue'
import { listAdmins, createAdmin, deleteAdmin } from '../api'
import { showToast } from '../toast'

const admins = ref([])
const loading = ref(false)
const newUser = ref('')
const newPass = ref('')
const createErr = ref('')

async function load() {
  loading.value = true
  try { admins.value = await listAdmins() } finally { loading.value = false }
}

async function doCreate() {
  createErr.value = ''
  if (!newUser.value.trim() || !newPass.value) { createErr.value = '用户名和密码不能为空'; return }
  try {
    await createAdmin(newUser.value.trim(), newPass.value)
    newUser.value = ''; newPass.value = ''
    showToast('管理员已创建')
    load()
  } catch (e) { createErr.value = e.message }
}

async function doDelete(username) {
  if (!confirm(`删除管理员 "${username}"？`)) return
  try { await deleteAdmin(username); showToast('已删除'); load() } catch { showToast('删除失败') }
}

function fmtDate(s) { return s ? s.slice(0, 10) : '—' }
onMounted(load)
</script>

<template>
  <div>
    <div class="card">
      <h2 class="section-title">添加管理员</h2>
      <div class="create-row">
        <div class="field-wrap">
          <label>用户名</label>
          <input v-model="newUser" placeholder="username" />
        </div>
        <div class="field-wrap">
          <label>密码</label>
          <input v-model="newPass" type="password" placeholder="password" />
        </div>
        <div class="field-wrap field-btn">
          <label>&nbsp;</label>
          <button class="btn-primary" @click="doCreate">添加</button>
        </div>
      </div>
      <p v-if="createErr" class="err-msg">{{ createErr }}</p>
    </div>

    <div class="card">
      <div class="table-header">
        <h2 class="section-title">管理员列表</h2>
        <button class="btn-outline btn-sm" @click="load">刷新</button>
      </div>
      <table>
        <thead>
          <tr>
            <th>用户名</th>
            <th>角色</th>
            <th>创建时间</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="4" class="empty-cell">加载中…</td>
          </tr>
          <tr v-else-if="!admins.length">
            <td colspan="4" class="empty-cell">暂无管理员</td>
          </tr>
          <tr v-for="a in admins" :key="a.username">
            <td><strong>{{ a.username }}</strong></td>
            <td>
              <span class="badge" :class="a.is_owner ? 'badge-owner' : 'badge-admin'">
                {{ a.is_owner ? 'Owner' : 'Admin' }}
              </span>
            </td>
            <td class="date-cell">{{ fmtDate(a.created_at) }}</td>
            <td>
              <button v-if="!a.is_owner" class="btn-danger btn-sm" @click="doDelete(a.username)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.section-title { font-size: 14px; font-weight: 600; color: #1e293b; margin-bottom: 14px; }
.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
.table-header .section-title { margin-bottom: 0; }

.create-row { display: grid; grid-template-columns: 1fr 1fr auto; gap: 12px; align-items: end; }
@media (max-width: 600px) { .create-row { grid-template-columns: 1fr; } }
.field-wrap label { display: block; font-size: 12px; font-weight: 500; color: #64748b; margin-bottom: 5px; }
.err-msg { color: #ef4444; font-size: 12.5px; margin-top: 10px; }

table { width: 100%; border-collapse: collapse; }
th {
  text-align: left; padding: 9px 12px;
  border-bottom: 2px solid #f1f5f9;
  font-size: 12px; font-weight: 600; color: #64748b; text-transform: uppercase; letter-spacing: .04em;
}
td { padding: 12px; border-bottom: 1px solid #f8fafc; vertical-align: middle; }
tr:last-child td { border-bottom: none; }
tr:hover td { background: #fafbff; }
.date-cell { font-size: 12.5px; color: #64748b; }
.btn-sm { padding: 4px 12px; font-size: 12.5px; }
.empty-cell { text-align: center; color: #94a3b8; padding: 32px; }
</style>
