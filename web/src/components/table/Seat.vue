<template>
  <div class="seat-wrapper" :class="{ 'is-active': isCurrentTurn }">
    <div class="seat" :class="{ 'has-player': !!player, 'folded': isFolded }">
      <template v-if="player">
        <div class="avatar-placeholder">{{ player.name.charAt(0).toUpperCase() }}</div>
        <div class="player-details">
          <div class="name" :title="player.name">{{ player.name }}</div>
          <div class="chips">💰{{ player.chips }}</div>
        </div>
        
        <!-- Status Overlay -->
        <div v-if="player.status !== PlayerStatus.Playing && player.status !== PlayerStatus.AllIn" class="status-badge">
          {{ player.status }}
        </div>
        <div v-if="player.status === PlayerStatus.AllIn" class="status-badge all-in">ALL IN</div>
        
        <!-- Current Bet Bubble (positioned absolute usually, but here flex for now) -->
        <div v-if="player.currentBet > 0" class="bet-chip">
          {{ player.currentBet }}
        </div>
      </template>
      
      <template v-else>
        <button class="sit-btn" @click="$emit('sit', seatIndex)">
          +
        </button>
      </template>
    </div>
    
    <!-- Optional slot for cards (opponent cards or showdown cards) -->
    <div class="seat-cards">
      <slot name="cards"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { type Player, PlayerStatus } from '@/types';

const props = defineProps<{
  player: Player | null;
  seatIndex: number;
  isCurrentTurn?: boolean;
}>();

defineEmits<{
  (e: 'sit', index: number): void;
}>();

const isFolded = computed(() => props.player?.status === PlayerStatus.Folded);
</script>

<style scoped>
.seat-wrapper {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100px;
}

.seat {
  width: 80px;
  height: 80px;
  background: rgba(0, 0, 0, 0.4);
  border: 2px solid transparent;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: white;
  position: relative;
  transition: all 0.3s;
}

.seat.has-player {
  background: #2c3e50;
  border-color: #34495e;
}

.seat-wrapper.is-active .seat {
  border-color: #f1c40f;
  box-shadow: 0 0 15px rgba(241, 196, 15, 0.6);
}

.seat.folded {
  opacity: 0.6;
}

.avatar-placeholder {
  font-size: 24px;
  font-weight: bold;
}

.player-details {
  font-size: 12px;
  text-align: center;
}

.name {
  max-width: 70px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chips {
  color: #2ecc71;
}

.sit-btn {
  background: transparent;
  border: 2px dashed rgba(255,255,255,0.3);
  color: rgba(255,255,255,0.5);
  border-radius: 50%;
  width: 60px;
  height: 60px;
  cursor: pointer;
  font-size: 24px;
}

.sit-btn:hover {
  border-color: rgba(255,255,255,0.8);
  color: white;
}

.status-badge {
  position: absolute;
  top: -10px;
  background: #7f8c8d;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;
  text-transform: uppercase;
}

.status-badge.all-in {
  background: #e74c3c;
  color: white;
  font-weight: bold;
}

.bet-chip {
  position: absolute;
  bottom: -15px;
  background: rgba(255, 215, 0, 0.9);
  color: black;
  border-radius: 12px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: bold;
  border: 1px solid #d4af37;
}

.seat-cards {
  position: absolute;
  top: 50%; /* Adjust based on where cards should go */
  left: 50%;
  transform: translate(-50%, -50%); /* Start centered */
  /* This needs layouting in PokerTable to positioning cards relative to seat */
  /* Actually, usually cards are slightly overlapping the avatar or next to it */
  pointer-events: none;
}
</style>
