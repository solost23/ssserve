import { reactive } from 'vue'

const stored = () => ({
  token: localStorage.getItem('jwt') || '',
  user: localStorage.getItem('jwt_user') || '',
  owner: localStorage.getItem('jwt_owner') === '1',
})

export const auth = reactive({ ...stored(), loggedIn: !!localStorage.getItem('jwt') })

export function setAuth(token, username, isOwner) {
  localStorage.setItem('jwt', token)
  localStorage.setItem('jwt_user', username)
  localStorage.setItem('jwt_owner', isOwner ? '1' : '0')
  auth.token = token
  auth.user = username
  auth.owner = isOwner
  auth.loggedIn = true
}

export function clearAuth() {
  localStorage.removeItem('jwt')
  localStorage.removeItem('jwt_user')
  localStorage.removeItem('jwt_owner')
  auth.token = ''
  auth.user = ''
  auth.owner = false
  auth.loggedIn = false
}
