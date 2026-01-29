<template>
  <div class="modal-overlay">
    <div class="modal">
      <h2>Cash Out</h2>
      <p>Current Chips: {{ chips }}</p>
      <p>Are you sure you want to leave the table?</p>
      <div class="actions">
        <button @click="$emit('cancel')">Cancel</button>
        <button @click="$emit('confirm')">Confirm</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  chips: number;
}>();
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 200;
}
.modal {
  background: white;
  padding: 20px;
  border-radius: 8px;
  color: black;
  min-width: 300px;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}
</style>
