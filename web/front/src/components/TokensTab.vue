<script setup>
import { ref, onMounted } from 'vue'
import { listTokens, createToken, deleteToken, updateQuota, resetUsage, extendExpiry, setSuspended, renameToken, setSpeedLimit } from '../api'
import { showToast } from '../toast'
import QuotaModal from './QuotaModal.vue'
import ExtendModal from './ExtendModal.vue'
import SpeedModal from './SpeedModal.vue'
import QrModal from './QrModal.vue'

const tokens = ref([])
const loading = ref(false)

const newName = ref('')
const newQuota = ref('')
const newExpires = ref('')
const newSpeed = ref('')
const createErr = ref('')

const quotaTarget = ref(null)
const extendTarget = ref(null)
const speedTarget = ref(null)
const qrTarget = ref(null)

// inline rename state
const editingToken = ref(null)
const editingName = ref('')

async function load() {
  loading.value = true
  try { tokens.value = await listTokens() } finally { loading.value = false }
}

async function doCreate() {
  createErr.value = ''
  if (!newName.value.trim()) { createErr.value = '名称不能为空'; return }
  const body = { name: newName.value.trim() }
  if (newQuota.value) body.quota_gb = parseFloat(newQuota.value)
  if (newExpires.value) body.expires_days = parseInt(newExpires.value)
  if (newSpeed.value) body.speed_limit_kbps = parseInt(newSpeed.value)
  try {
    const data = await createToken(body)
    newName.value = ''; newQuota.value = ''; newExpires.value = ''; newSpeed.value = ''
    copy(data.clash_url || data.sub_url)
    showToast('已创建，订阅链接已复制')
    load()
  } catch (e) { createErr.value = e.message }
}

async function doDelete(token) {
  if (!confirm('删除后订阅立即失效，确认删除？')) return
  try { await deleteToken(token); showToast('已删除'); load() } catch { showToast('删除失败') }
}

async function doResetUsage(token) {
  if (!confirm('重置该 Token 的流量统计？')) return
  try { await resetUsage(token); showToast('流量已重置'); load() } catch { showToast('重置失败') }
}

async function doToggleSuspend(t) {
  const action = t.suspended ? '恢复' : '暂停'
  if (!confirm(`${action}该 Token？`)) return
  try {
    await setSuspended(t.token, !t.suspended)
    showToast(`已${action}`)
    load()
  } catch (e) { showToast(e.message) }
}

function startRename(t) {
  editingToken.value = t.token
  editingName.value = t.name
}

async function saveRename(token) {
  const name = editingName.value.trim()
  if (!name) { editingToken.value = null; return }
  try {
    await renameToken(token, name)
    showToast('已重命名')
    editingToken.value = null
    load()
  } catch { showToast('重命名失败') }
}

function openExtend(t) { extendTarget.value = { token: t.token, expires_at: t.expires_at } }

async function onSaveExtend({ token, days }) {
  try {
    const msg = await extendExpiry(token, days)
    showToast(msg || '续期成功')
    extendTarget.value = null
    load()
  } catch { showToast('续期失败') }
}

function openQuota(t) { quotaTarget.value = { token: t.token, quota_gb: t.quota_gb } }

async function onSaveQuota({ token, quotaGB }) {
  try {
    await updateQuota(token, quotaGB)
    showToast('配额已更新')
    quotaTarget.value = null
    load()
  } catch { showToast('更新失败') }
}

function openSpeed(t) { speedTarget.value = { token: t.token, kbps: t.speed_limit_kbps || 0 } }

async function onSaveSpeed({ token, kbps }) {
  try {
    await setSpeedLimit(token, kbps)
    showToast(kbps > 0 ? `限速已设为 ${kbps} KB/s` : '已取消限速')
    speedTarget.value = null
    load()
  } catch { showToast('设置失败') }
}

function copy(text) {
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(text)
  } else {
    const el = document.createElement('textarea')
    el.value = text
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
  }
}

function fmtBytes(b) {
  if (!b) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB']; let i = 0, v = b
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return v.toFixed(1) + ' ' + u[i]
}
function fmtDate(s) { return s ? s.slice(0, 10) : '—' }

function usedPct(t) {
  return t.quota_gb ? Math.min(100, t.used_bytes / (t.quota_gb * 1e9) * 100) : 0
}

function trafficBarClass(t) {
  const pct = usedPct(t)
  if (pct >= 100) return 'fill-over'
  if (pct >= 80)  return 'fill-warn'
  return ''
}

function daysUntil(dateStr) {
  if (!dateStr) return null
  return Math.ceil((new Date(dateStr) - Date.now()) / 86400000)
}

function expiryClass(dateStr) {
  const d = daysUntil(dateStr)
  if (d === null) return ''
  if (d <= 0)  return 'expiry-expired'
  if (d <= 7)  return 'expiry-soon'
  return ''
}

function expiryLabel(dateStr) {
  const d = daysUntil(dateStr)
  if (d === null) return '—'
  if (d <= 0)  return `${dateStr.slice(0,10)} (已过期)`
  if (d <= 7)  return `${dateStr.slice(0,10)} (${d}天后)`
  return dateStr.slice(0, 10)
}

function statusInfo(t) {
  if (!t.active)   return { cls: 'badge-inactive', label: 'Deleted' }
  if (t.suspended) return { cls: 'badge-inactive', label: 'Suspended' }
  return { cls: 'badge-active', label: 'Active' }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="card">
      <h2 class="section-title">创建 Token</h2>
      <div class="create-row">
        <div class="field-wrap">
          <label>名称</label>
          <input v-model="newName" placeholder="e.g. iPhone" @keydown.enter="doCreate" />
        </div>
        <div class="field-wrap">
          <label>配额 GB（留空不限）</label>
          <input v-model="newQuota" type="number" min="1" placeholder="不限" />
        </div>
        <div class="field-wrap">
          <label>有效天数（留空永久）</label>
          <input v-model="newExpires" type="number" min="1" placeholder="永久" />
        </div>
        <div class="field-wrap">
          <label>限速 KB/s（留空不限）</label>
          <input v-model="newSpeed" type="number" min="0" placeholder="不限" />
        </div>
        <div class="field-wrap field-btn">
          <label>&nbsp;</label>
          <button class="btn-primary" @click="doCreate">创建</button>
        </div>
      </div>
      <p v-if="createErr" class="err-msg">{{ createErr }}</p>
    </div>

    <div class="card">
      <div class="table-header">
        <h2 class="section-title">Token 列表</h2>
        <button class="btn-outline btn-sm" @click="load">刷新</button>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>状态</th>
              <th>流量</th>
              <th>创建</th>
              <th>到期</th>
              <th>订阅链接</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="7" class="empty-cell">加载中…</td>
            </tr>
            <tr v-else-if="!tokens.length">
              <td colspan="7" class="empty-cell">暂无 Token</td>
            </tr>
            <tr v-for="t in tokens" :key="t.token">
              <td>
                <template v-if="editingToken === t.token">
                  <input
                    class="rename-input"
                    v-model="editingName"
                    @keydown.enter="saveRename(t.token)"
                    @keydown.escape="editingToken = null"
                    @blur="saveRename(t.token)"
                    autofocus
                  />
                </template>
                <template v-else>
                  <strong class="token-name" @dblclick="startRename(t)" title="双击重命名">{{ t.name }}</strong>
                  <span class="token-port">:{{ t.server_port }}</span>
                  <span v-if="t.speed_limit_kbps > 0" class="speed-badge">{{ t.speed_limit_kbps }} KB/s</span>
                </template>
              </td>
              <td><span class="badge" :class="statusInfo(t).cls">{{ statusInfo(t).label }}</span></td>
              <td>
                <div class="traffic-cell">
                  <div class="traffic-top">
                    <span :class="{ 'warn-text': usedPct(t) >= 80 }">
                      {{ fmtBytes(t.used_bytes) }} / {{ t.quota_gb ? t.quota_gb + ' GB' : '∞' }}
                    </span>
                    <button v-if="t.active" class="btn-ghost micro-btn" @click="openQuota(t)">编辑</button>
                    <button v-if="t.active" class="btn-ghost micro-btn" @click="doResetUsage(t.token)">重置</button>
                  </div>
                  <div v-if="t.quota_gb" class="traffic-bar">
                    <div
                      class="traffic-fill"
                      :class="trafficBarClass(t)"
                      :style="{ width: usedPct(t) + '%' }"
                    />
                  </div>
                </div>
              </td>
              <td class="date-cell">{{ fmtDate(t.created_at) }}</td>
              <td class="date-cell" :class="expiryClass(t.expires_at)">
                {{ expiryLabel(t.expires_at) }}
                <button v-if="t.active" class="btn-ghost micro-btn" @click="openExtend(t)">续期</button>
              </td>
              <td>
                <div class="sub-links">
                  <div class="sub-row">
                    <span class="sub-label">Clash</span>
                    <span class="sub-url">{{ t.clash_url || t.sub_url }}</span>
                    <button class="btn-ghost copy-btn" @click="copy(t.clash_url || t.sub_url); showToast('已复制')">复制</button>
                  </div>
                  <div class="sub-row">
                    <span class="sub-label">SS</span>
                    <span class="sub-url">{{ t.ss_url }}</span>
                    <button class="btn-ghost copy-btn" @click="copy(t.ss_url); showToast('已复制')">复制</button>
                    <button class="btn-ghost copy-btn" @click="qrTarget = { url: t.ss_url, label: t.name }">二维码</button>
                  </div>
                </div>
              </td>
              <td>
                <div class="action-col">
                  <button v-if="t.active" class="btn-outline btn-sm" @click="openSpeed(t)">限速</button>
                  <button v-if="t.active" class="btn-outline btn-sm" @click="doToggleSuspend(t)">
                    {{ t.suspended ? '恢复' : '暂停' }}
                  </button>
                  <button class="btn-danger btn-sm" @click="doDelete(t.token)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <QuotaModal
      v-if="quotaTarget"
      :token="quotaTarget.token"
      :current="quotaTarget.quota_gb"
      @save="onSaveQuota"
      @cancel="quotaTarget = null"
    />

    <ExtendModal
      v-if="extendTarget"
      :token="extendTarget.token"
      :expires-at="extendTarget.expires_at"
      @save="onSaveExtend"
      @cancel="extendTarget = null"
    />

    <SpeedModal
      v-if="speedTarget"
      :token="speedTarget.token"
      :current="speedTarget.kbps"
      @save="onSaveSpeed"
      @cancel="speedTarget = null"
    />

    <QrModal
      v-if="qrTarget"
      :url="qrTarget.url"
      :label="qrTarget.label"
      @close="qrTarget = null"
    />
  </div>
</template>

<style scoped>
.section-title { font-size: 14px; font-weight: 600; color: #1e293b; margin-bottom: 14px; }
.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
.table-header .section-title { margin-bottom: 0; }

.create-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr auto;
  gap: 12px;
  align-items: end;
}
@media (max-width: 800px) { .create-row { grid-template-columns: 1fr 1fr; } }
@media (max-width: 500px) { .create-row { grid-template-columns: 1fr; } }

.field-wrap label { display: block; font-size: 12px; font-weight: 500; color: #64748b; margin-bottom: 5px; }
.field-btn button { white-space: nowrap; }
.err-msg { color: #ef4444; font-size: 12.5px; margin-top: 10px; }

.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
th {
  text-align: left; padding: 9px 12px;
  border-bottom: 2px solid #f1f5f9;
  font-size: 12px; font-weight: 600; color: #64748b; text-transform: uppercase; letter-spacing: .04em;
}
td { padding: 12px; border-bottom: 1px solid #f8fafc; vertical-align: middle; }
tr:last-child td { border-bottom: none; }
tr:hover td { background: #fafbff; }

.token-name { font-weight: 600; font-size: 13.5px; display: block; cursor: pointer; }
.token-port { font-family: monospace; font-size: 11.5px; color: #94a3b8; }
.speed-badge {
  display: inline-block; margin-left: 6px;
  background: #f0fdf4; color: #16a34a;
  font-size: 10.5px; font-weight: 600; padding: 1px 6px; border-radius: 999px;
}
.rename-input {
  font-size: 13.5px; font-weight: 600;
  border: 1.5px solid #6366f1; border-radius: 5px;
  padding: 2px 7px; outline: none; width: 120px;
}

.traffic-cell { min-width: 160px; }
.traffic-top { display: flex; align-items: center; gap: 4px; font-size: 12.5px; margin-bottom: 4px; }
.warn-text { color: #d97706; font-weight: 600; }
.micro-btn { padding: 1px 6px; font-size: 11.5px; }
.traffic-bar { height: 5px; background: #e2e8f0; border-radius: 3px; overflow: hidden; }
.traffic-fill { height: 100%; background: #6366f1; border-radius: 3px; transition: width .3s; }
.traffic-fill.fill-warn { background: #f59e0b; }
.traffic-fill.fill-over { background: #ef4444; }

.date-cell { font-size: 12.5px; color: #64748b; white-space: nowrap; }
.expiry-soon { color: #d97706; font-weight: 600; }
.expiry-expired { color: #ef4444; font-weight: 600; }

.sub-links { display: flex; flex-direction: column; gap: 5px; min-width: 260px; }
.sub-row { display: flex; align-items: center; gap: 6px; }
.sub-label { font-size: 11px; font-weight: 600; color: #94a3b8; text-transform: uppercase; min-width: 52px; }
.sub-url { font-family: monospace; font-size: 11.5px; color: #475569; flex: 1; word-break: break-all; line-height: 1.4; }
.copy-btn { padding: 2px 8px; font-size: 11.5px; white-space: nowrap; }

.action-col { display: flex; flex-direction: column; gap: 5px; align-items: flex-start; }
.btn-sm { padding: 4px 12px; font-size: 12.5px; }
.empty-cell { text-align: center; color: #94a3b8; padding: 32px; }
</style>
