<template>
  <div class="hand-cards">
    <PlayingCard 
      v-for="(card, index) in myCards" 
      :key="index" 
      :card="card" 
      class="my-card"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useGameStore } from '@/stores/game';
import PlayingCard from '@/components/card/PlayingCard.vue';

const gameStore = useGameStore();
const myCards = computed(() => gameStore.myCards);
</script>

<style scoped>
.hand-cards {
  display: flex;
  gap: 4px;
  justify-content: center;
  padding: 10px;
}

.my-card {
  width: 60px; /* Larger than default */
  height: 84px;
  transform: translateY(0);
  transition: transform 0.2s;
}

.my-card:hover {
  transform: translateY(-10px);
}
</style>
