import { auth, clearAuth } from './auth'

async function req(path, opts = {}) {
  const res = await fetch(path, {
    ...opts,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + auth.token,
      ...(opts.headers || {}),
    },
  })
  if (res.status === 401) clearAuth()
  return res
}

export async function login(username, password) {
  const res = await fetch('/admin/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) throw new Error('Invalid credentials')
  return res.json()
}

export async function listTokens() {
  const res = await req('/admin/tokens')
  if (!res.ok) throw new Error('Failed to load tokens')
  return res.json()
}

export async function createToken(body) {
  const res = await req('/admin/tokens', { method: 'POST', body: JSON.stringify(body) })
  if (!res.ok) throw new Error('Failed to create token')
  return res.json()
}

export async function deleteToken(token) {
  const res = await req('/admin/tokens/' + token, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete token')
}

export async function updateQuota(token, quotaGB) {
  const res = await req('/admin/tokens/' + token, {
    method: 'PATCH',
    body: JSON.stringify({ quota_gb: quotaGB }),
  })
  if (!res.ok) throw new Error('Failed to update quota')
}

export async function listAdmins() {
  const res = await req('/admin/admins')
  if (!res.ok) throw new Error('Failed to load admins')
  return res.json()
}

export async function createAdmin(username, password) {
  const res = await req('/admin/admins', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) throw new Error('Failed to create admin')
}

export async function deleteAdmin(username) {
  const res = await req('/admin/admins/' + encodeURIComponent(username), { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete admin')
}
