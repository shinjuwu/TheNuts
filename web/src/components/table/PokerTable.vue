<template>
  <div class="poker-table-container">
    <div class="table-felt">
      <div class="center-area">
        <PotDisplay v-if="tableState" :total="tableState.potTotal" />
        <CommunityCards v-if="tableState" :cards="tableState.communityCards" />
      </div>
    </div>

    <!-- Seats -->
    <!-- Loop 0 to 8 -->
    <div 
      v-for="i in 9" 
      :key="i-1" 
      class="seat-container-abs" 
      :style="getSeatStyle(i-1)"
    >
      <Seat 
        :player="getPlayerAt(i-1)" 
        :seat-index="i-1"
        :is-current-turn="tableState?.currentPos === (i-1)"
        @sit="handleSit"
      >
        <template #cards>
           <!-- Show opponent cards if playing -->
           <div v-if="shouldShowOpponentCards(i-1)" class="opponent-cards">
             <PlayingCard :back="true" style="transform: rotate(-10deg); margin-right: -20px;" />
             <PlayingCard :back="true" style="transform: rotate(10deg);" />
           </div>
        </template>
      </Seat>
      
      <DealerButton 
        v-if="tableState?.dealerPos === (i-1)" 
        class="dealer-btn"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useGameStore } from '@/stores/game';
import Seat from './Seat.vue';
import CommunityCards from './CommunityCards.vue';
import PotDisplay from './PotDisplay.vue';
import DealerButton from './DealerButton.vue';
import PlayingCard from '@/components/card/PlayingCard.vue';
import { PlayerStatus } from '@/types';
const gameStore = useGameStore();
// const { send } = useWebSocket(); // Not used anymore

const tableState = computed(() => gameStore.tableState);

// Fixed positions for 9 seats (Percentages of container)
// 0 is bottom center (User's view relative)
const basePositions = [
  { left: 50, top: 88 }, // Bottom
  { left: 20, top: 80 }, // Bottom-Left
  { left: 5,  top: 50 }, // Left
  { left: 20, top: 20 }, // Top-Left
  { left: 40, top: 12 }, // Top-Left-Center
  { left: 60, top: 12 }, // Top-Right-Center
  { left: 80, top: 20 }, // Top-Right
  { left: 95, top: 50 }, // Right
  { left: 80, top: 80 }, // Bottom-Right
];

// If I am sitting at seat index S, I want S to be at visual index 0.
// visualIndex = (seatIndex - mySeat + 9) % 9
const rotationOffset = computed(() => {
  const mySeat = gameStore.mySeatIdx;
  return mySeat !== -1 ? mySeat : 0;
});

function getSeatStyle(serverSeatIdx: number) {
  // Map server seat to visual seat
  const visualIndex = (serverSeatIdx - rotationOffset.value + 9) % 9;
  const pos = basePositions[visualIndex] || { left: 50, top: 50 };
  return {
    left: `${pos.left}%`,
    top: `${pos.top}%`,
    transform: 'translate(-50%, -50%)'
  };
}

function getPlayerAt(seatIdx: number) {
  return tableState.value?.players[seatIdx] || null;
}

// Define emits in script setup top-level
const emit = defineEmits<{
  (e: 'sit-request', seatIdx: number): void
}>();

function handleSit(seatIdx: number) {
  if (gameStore.mySeatIdx !== -1) {
    alert("You are already sitting!");
    return;
  }
  emit('sit-request', seatIdx);
}

function shouldShowOpponentCards(seatIdx: number) {
  if (!tableState.value) return false;
  if (seatIdx === gameStore.mySeatIdx) return false; // HandCards component handles my cards
  const p = getPlayerAt(seatIdx);
  if (!p) return false;
  return p.status === PlayerStatus.Playing || p.status === PlayerStatus.AllIn;
}
</script>

<style scoped>
.poker-table-container {
  width: 100%;
  max-width: 1000px;
  aspect-ratio: 2/1; /* Ellipse aspect */
  position: relative;
  margin: 0 auto;
}

.table-felt {
  width: 100%;
  height: 100%;
  background: radial-gradient(ellipse at center, #27ae60 0%, #219150 100%);
  border-radius: 50% / 50%;
  border: 15px solid #5d4037;
  box-shadow: inset 0 0 50px rgba(0,0,0,0.5), 0 10px 20px rgba(0,0,0,0.3);
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
}

.center-area {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.seat-container-abs {
  position: absolute;
  /* left/top set by inline style */
}

.dealer-btn {
  position: absolute;
  top: 70%; 
  right: -10px;
  z-index: 10;
}

.opponent-cards {
  display: flex;
  position: absolute;
  top: 30%; 
  left: 60%;
  z-index: 5;
}
</style>
