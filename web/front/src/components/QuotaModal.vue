<script setup>
import { ref } from 'vue'

const props = defineProps({ token: String, current: Number })
const emit = defineEmits(['save', 'cancel'])

const val = ref(props.current != null ? String(props.current) : '')

function save() {
  const quotaGB = val.value.trim() === '' ? null : parseFloat(val.value)
  if (quotaGB !== null && (isNaN(quotaGB) || quotaGB <= 0)) return
  emit('save', { token: props.token, quotaGB })
}
</script>

<template>
  <div class="overlay" @click.self="emit('cancel')">
    <div class="modal">
      <h3>编辑配额</h3>
      <p class="modal-desc">输入 GB 数（留空为不限流量）</p>
      <input
        v-model="val"
        type="number"
        min="0"
        step="0.5"
        placeholder="留空不限"
        @keydown.enter="save"
        @keydown.escape="emit('cancel')"
        autofocus
      />
      <div class="modal-actions">
        <button class="btn-outline" @click="emit('cancel')">取消</button>
        <button class="btn-primary" @click="save">保存</button>
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
  background: #fff;
  border-radius: 12px;
  padding: 28px 28px 24px;
  width: 340px;
  box-shadow: 0 8px 32px rgba(0,0,0,.12);
}
h3 { font-size: 15px; font-weight: 700; margin-bottom: 6px; }
.modal-desc { font-size: 12.5px; color: #64748b; margin-bottom: 16px; }
.modal-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 18px; }
</style>
