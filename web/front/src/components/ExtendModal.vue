<script setup>
import { ref } from 'vue'

const props = defineProps({ token: String, expiresAt: String })
const emit = defineEmits(['save', 'cancel'])

const days = ref(30)

function save() {
  const d = parseInt(days.value)
  if (!d || d <= 0) return
  emit('save', { token: props.token, days: d })
}
</script>

<template>
  <div class="overlay" @click.self="emit('cancel')">
    <div class="modal">
      <h3>续期</h3>
      <p class="desc">
        当前到期：<strong>{{ expiresAt ? expiresAt.slice(0,10) : '永不过期' }}</strong>
      </p>
      <div class="field">
        <label>延长天数</label>
        <input v-model="days" type="number" min="1" @keydown.enter="save" @keydown.escape="emit('cancel')" autofocus />
      </div>
      <div class="actions">
        <button class="btn-outline" @click="emit('cancel')">取消</button>
        <button class="btn-primary" @click="save">确认续期</button>
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
.field label { display: block; font-size: 12px; font-weight: 500; color: #64748b; margin-bottom: 5px; }
.actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
</style>
