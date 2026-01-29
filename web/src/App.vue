<script setup lang="ts">
import { RouterView } from 'vue-router'
import { ref, watch } from 'vue';
import Toast from '@/components/common/Toast.vue';
import { useGameStore } from '@/stores/game';

const gameStore = useGameStore();
const toastRef = ref<InstanceType<typeof Toast> | null>(null);

watch(() => gameStore.lastError, (newErr) => {
  if (newErr && toastRef.value) {
    toastRef.value.add(newErr, 'error');
    // Clear error so we can show same error again if happens (or store handles it)
    gameStore.lastError = ''; 
  }
});
</script>

<template>
  <RouterView />
  <Toast ref="toastRef" />
</template>


<style>
/* Global styles can go here or in style.css */
</style>
