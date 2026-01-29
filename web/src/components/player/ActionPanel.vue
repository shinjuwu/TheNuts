<template>
  <div class="action-panel" v-if="gameStore.isMyTurn">
    <div class="timer-bar" :style="{ width: timerWidth + '%' }"></div>
    
    <div class="controls">
      <button v-if="canFold" class="btn fold" @click="doAction(GameActionType.Fold)">Fold</button>
      
      <button v-if="canCheck" class="btn check" @click="doAction(GameActionType.Check)">Check</button>
      
      <button v-if="canCall" class="btn call" @click="doAction(GameActionType.Call)">
        Call {{ gameStore.amountToCall }}
      </button>
      
      <div v-if="canBet || canRaise" class="bet-controls">
        <input 
          type="range" 
          v-model.number="betAmount" 
          :min="minBetAmount" 
          :max="maxBetAmount"
          step="1"
        />
        <input type="number" v-model.number="betAmount" class="bet-input" />
        
        <button v-if="canBet" class="btn bet" @click="doAction(GameActionType.Bet, betAmount)">
          Bet {{ betAmount }}
        </button>
        <button v-if="canRaise" class="btn raise" @click="doAction(GameActionType.Raise, betAmount)">
          Raise {{ betAmount }}
        </button>
      </div>

      <button v-if="canAllIn" class="btn all-in" @click="doAction(GameActionType.AllIn)">All In</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from 'vue';
import { useGameStore } from '@/stores/game';
import { useWebSocket } from '@/composables/useWebSocket';
import { WSAction, GameActionType } from '@/types';

const gameStore = useGameStore();
const { send } = useWebSocket();

const betAmount = ref(0);
const timerWidth = ref(100);
let timerInterval: any = null;

const canFold = computed(() => gameStore.validActions.includes(GameActionType.Fold));
const canCheck = computed(() => gameStore.validActions.includes(GameActionType.Check));
const canCall = computed(() => gameStore.validActions.includes(GameActionType.Call));
const canBet = computed(() => gameStore.validActions.includes(GameActionType.Bet));
const canRaise = computed(() => gameStore.validActions.includes(GameActionType.Raise));
const canAllIn = computed(() => gameStore.validActions.includes(GameActionType.AllIn));

// Calculate min and max for slider
const myChips = computed(() => {
  if (gameStore.mySeatIdx === -1 || !gameStore.tableState) return 0;
  return gameStore.tableState.players[gameStore.mySeatIdx]?.chips || 0;
});

const minBetAmount = computed(() => gameStore.minRaise || gameStore.tableState?.minBet || 0);
const maxBetAmount = computed(() => myChips.value);

watch(() => gameStore.isMyTurn, (val) => {
  if (val) {
    betAmount.value = minBetAmount.value;
    startTimer();
  } else {
    stopTimer();
  }
});

function startTimer() {
  stopTimer();
  const totalTime = gameStore.timeRemaining || 15; // default 15s
  let timeLeft = totalTime;
  timerWidth.value = 100;
  
  timerInterval = setInterval(() => {
    timeLeft -= 0.1;
    timerWidth.value = (timeLeft / totalTime) * 100;
    if (timeLeft <= 0) stopTimer();
  }, 100);
}

function stopTimer() {
  if (timerInterval) clearInterval(timerInterval);
  timerInterval = null;
}

function doAction(actionType: GameActionType, amount: number = 0) {
  const payload = {
    game_action: actionType,
    amount: amount
  };
  
  send(WSAction.GameAction, payload);
}

onUnmounted(() => {
  stopTimer();
});
</script>

<style scoped>
.action-panel {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(0, 0, 0, 0.85);
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: white;
  z-index: 100;
}

.timer-bar {
  height: 4px;
  background: #e74c3c;
  transition: width 0.1s linear;
}

.controls {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
  font-weight: bold;
  cursor: pointer;
  text-transform: uppercase;
}

.fold { background: #95a5a6; color: white; }
.check { background: #3498db; color: white; }
.call { background: #3498db; color: white; }
.bet { background: #f1c40f; color: black; }
.raise { background: #f39c12; color: black; }
.all-in { background: #e74c3c; color: white; }

.bet-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(255,255,255,0.1);
  padding: 5px 10px;
  border-radius: 4px;
}

.bet-input {
  width: 60px;
}
</style>
