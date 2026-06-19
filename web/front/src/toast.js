import { reactive } from 'vue'

export const toast = reactive({ msg: '', visible: false })
let timer

export function showToast(msg) {
  toast.msg = msg
  toast.visible = true
  clearTimeout(timer)
  timer = setTimeout(() => { toast.visible = false }, 2800)
}
