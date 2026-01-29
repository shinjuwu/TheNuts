<template>
  <div class="modal-overlay">
    <div class="modal">
      <h2>Buy In</h2>
      <p>Your Balance: {{ balance }}</p>
      <div class="field">
        <label>Amount:</label>
        <input type="number" v-model.number="amount" :max="balance" />
      </div>
      <div class="actions">
        <button @click="$emit('cancel')">Cancel</button>
        <button @click="$emit('confirm', amount)" :disabled="amount <= 0 || amount > balance">Buy In</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

defineProps<{
  balance: number;
}>();

const amount = ref(1000); // Default buyin
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
.field {
  margin: 15px 0;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
