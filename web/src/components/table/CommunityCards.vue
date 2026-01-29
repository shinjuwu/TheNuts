<template>
  <div class="community-cards">
    <PlayingCard v-for="(card, i) in cards" :key="i" :card="card" />
    <!-- Placeholders for empty slots? Optional. -->
  </div>
</template>

<script setup lang="ts">
import PlayingCard from '@/components/card/PlayingCard.vue';
import type { Card } from '@/types';

defineProps<{
  cards: Card[];
}>();
</script>

<style scoped>
.community-cards {
  display: flex;
  gap: 8px;
  justify-content: center;
  align-items: center;
  height: 60px; /* placeholder height */
}
</style>
