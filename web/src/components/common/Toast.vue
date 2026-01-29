<template>
  <div class="toast-container">
    <transition-group name="fade">
      <div v-for="toast in toasts" :key="toast.id" class="toast" :class="toast.type">
        {{ toast.message }}
      </div>
    </transition-group>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

interface Toast {
  id: number;
  message: string;
  type: 'info' | 'error' | 'success';
}

const toasts = ref<Toast[]>([]);
let nextId = 0;

function add(message: string, type: 'info' | 'error' | 'success' = 'info') {
  const id = nextId++;
  toasts.value.push({ id, message, type });
  setTimeout(() => {
    remove(id);
  }, 3000);
}

function remove(id: number) {
  const idx = toasts.value.findIndex(t => t.id === id);
  if (idx !== -1) toasts.value.splice(idx, 1);
}

defineExpose({ add });
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.toast {
  padding: 10px 20px;
  border-radius: 4px;
  color: white;
  box-shadow: 0 2px 5px rgba(0,0,0,0.2);
  min-width: 200px;
}

.toast.info { background: #3498db; }
.toast.error { background: #e74c3c; }
.toast.success { background: #2ecc71; }

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
