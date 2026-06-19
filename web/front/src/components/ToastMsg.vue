<script setup>
import { toast } from '../toast'
</script>

<template>
  <Transition name="toast">
    <div v-if="toast.visible" class="toast">{{ toast.msg }}</div>
  </Transition>
</template>

<style scoped>
.toast {
  position: fixed; bottom: 28px; left: 50%; transform: translateX(-50%);
  background: #1e293b; color: #fff;
  padding: 10px 22px; border-radius: 8px;
  font-size: 13.5px; font-weight: 500;
  box-shadow: 0 4px 16px rgba(0,0,0,.2);
  z-index: 100; pointer-events: none;
  white-space: nowrap;
}
.toast-enter-active, .toast-leave-active { transition: opacity .2s, transform .2s; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateX(-50%) translateY(8px); }
</style>
