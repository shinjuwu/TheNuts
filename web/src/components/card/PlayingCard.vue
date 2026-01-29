<template>
  <div class="playing-card" :class="[suitClass, { back: isBack }]">
    <template v-if="!isBack">
      <div class="rank">{{ rank }}</div>
      <div class="suit">{{ suitSymbol }}</div>
    </template>
    <template v-else>
      <div class="pattern"></div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  card?: string; // "AS", "TH", etc.
  back?: boolean;
}>();

const isBack = computed(() => props.back || !props.card);

const rank = computed(() => {
  if (isBack.value || !props.card) return '';
  return props.card[0] === 'T' ? '10' : props.card[0];
});

const suit = computed(() => {
  if (isBack.value || !props.card) return '';
  return props.card[1];
});

const suitSymbol = computed(() => {
  switch (suit.value) {
    case 'S': return '♠';
    case 'H': return '♥';
    case 'D': return '♦';
    case 'C': return '♣';
    default: return '';
  }
});

const suitClass = computed(() => {
  if (isBack.value) return 'back';
  switch (suit.value) {
    case 'H': return 'hearts';
    case 'D': return 'diamonds';
    case 'C': return 'clubs';
    case 'S': return 'spades';
    default: return '';
  }
});
</script>

<style scoped>
.playing-card {
  width: 40px;
  height: 56px;
  background: white;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  font-weight: bold;
  box-shadow: 1px 1px 3px rgba(0,0,0,0.3);
  user-select: none;
  position: relative;
}

.playing-card.back {
  background: #34495e;
  border: 2px solid #fff;
}

.playing-card.hearts, .playing-card.diamonds {
  color: #e74c3c;
}

.playing-card.spades, .playing-card.clubs {
  color: #2c3e50;
}

.rank {
  font-size: 14px;
  line-height: 1;
}

.suit {
  font-size: 20px;
  line-height: 1;
}
</style>
