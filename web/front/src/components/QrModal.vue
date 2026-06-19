<script setup>
import { ref, onMounted } from 'vue'
import QRCode from 'qrcode'

const props = defineProps({ url: String, label: String })
const emit = defineEmits(['close'])

const dataUrl = ref('')

onMounted(async () => {
  dataUrl.value = await QRCode.toDataURL(props.url, { width: 260, margin: 2 })
})
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="modal">
      <h3>{{ label }}</h3>
      <p class="url">{{ url }}</p>
      <div class="qr-wrap">
        <img v-if="dataUrl" :src="dataUrl" alt="QR Code" />
        <div v-else class="qr-placeholder">生成中…</div>
      </div>
      <div class="actions">
        <button class="btn-primary" @click="emit('close')">关闭</button>
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
  padding: 28px 28px 24px; width: 320px;
  box-shadow: 0 8px 32px rgba(0,0,0,.12);
  text-align: center;
}
h3 { font-size: 15px; font-weight: 700; margin-bottom: 8px; }
.url {
  font-family: monospace; font-size: 11px; color: #94a3b8;
  word-break: break-all; margin-bottom: 16px; line-height: 1.5;
}
.qr-wrap { display: flex; justify-content: center; margin-bottom: 16px; }
.qr-wrap img { border-radius: 8px; }
.qr-placeholder { width: 260px; height: 260px; background: #f1f5f9; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #94a3b8; font-size: 13px; }
.actions { display: flex; justify-content: center; }
</style>
