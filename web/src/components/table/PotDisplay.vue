<template>
  <div class="pot-display">
    <div class="pot-label">Pot: {{ total }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  total: number;
}>();
</script>

<style scoped>
.pot-display {
  background: rgba(0, 0, 0, 0.5);
  padding: 4px 12px;
  border-radius: 16px;
  color: white;
  margin-top: 8px;
  font-weight: bold;
}
</style>
