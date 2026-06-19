<script setup>
import { ref } from 'vue'

const props = defineProps({ token: String, current: { type: Number, default: 0 } })
const emit = defineEmits(['save', 'cancel'])

const kbps = ref(props.current || '')

function save() {
  const v = kbps.value === '' ? 0 : parseInt(kbps.value)
  if (isNaN(v) || v < 0) return
  emit('save', { token: props.token, kbps: v })
}
</script>

<template>
  <div class="overlay" @click.self="emit('cancel')">
    <div class="modal">
      <h3>限速设置</h3>
      <p class="desc">留空或填 0 表示不限速</p>
      <div class="field">
        <label>速度上限 (KB/s)</label>
        <input v-model="kbps" type="number" min="0" placeholder="不限速"
               @keydown.enter="save" @keydown.escape="emit('cancel')" autofocus />
      </div>
      <p v-if="current > 0" class="hint">当前限速：{{ current }} KB/s（≈ {{ (current / 128).toFixed(1) }} Mbps）</p>
      <div class="actions">
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
  background: #fff; border-radius: 12px;
  padding: 28px 28px 24px; width: 340px;
  box-shadow: 0 8px 32px rgba(0,0,0,.12);
}
h3 { font-size: 15px; font-weight: 700; margin-bottom: 6px; }
.desc { font-size: 12.5px; color: #64748b; margin-bottom: 16px; }
.hint { font-size: 12px; color: #94a3b8; margin-top: 8px; }
.field label { display: block; font-size: 12px; font-weight: 500; color: #64748b; margin-bottom: 5px; }
.actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
</style>
