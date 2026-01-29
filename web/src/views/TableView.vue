<template>
  <div class="table-view">
    <div class="table-header">
      <div class="left-controls">
        <button class="nav-btn" @click="goBack">← Lobby</button>
      </div>
      <div class="table-title">Table #{{ $route.params.id }}</div>
      <div class="right-controls">
        <button v-if="gameStore.mySeatIdx !== -1" class="nav-btn red" @click="showCashOut = true">
          Leave & Cash Out
        </button>
      </div>
    </div>

    <div class="game-area">
      <PokerTable @sit-request="onSitRequest" />
      
      <!-- Hand Cards positioned absolutely or part of layout -->
      <div class="hand-cards-container" v-if="gameStore.myCards.length > 0">
        <HandCards />
      </div>
    </div>

    <ActionPanel />

    <BuyInModal 
      v-if="showBuyIn" 
      :balance="walletStore.balance" 
      @cancel="showBuyIn = false" 
      @confirm="handleBuyIn" 
    />

    <CashOutModal 
      v-if="showCashOut" 
      :chips="myChips" 
      @cancel="showCashOut = false" 
      @confirm="handleCashOut" 
    />

    <div v-if="status !== 'connected'" class="connection-overlay">
      <div class="spinner"></div>
      <div>{{ status === 'connecting' ? 'Connecting...' : 'Disconnected' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useGameStore } from '@/stores/game';
import { useAuthStore } from '@/stores/auth';
import { useWalletStore } from '@/stores/wallet';
import { useWebSocket } from '@/composables/useWebSocket';
import { WSAction } from '@/types';

import PokerTable from '@/components/table/PokerTable.vue';
import ActionPanel from '@/components/player/ActionPanel.vue';
import HandCards from '@/components/player/HandCards.vue';
import BuyInModal from '@/components/common/BuyInModal.vue';
import CashOutModal from '@/components/common/CashOutModal.vue';

const route = useRoute();
const router = useRouter();
const gameStore = useGameStore();
const authStore = useAuthStore();
const walletStore = useWalletStore();
const { connect, disconnect, status, send } = useWebSocket();

const showBuyIn = ref(false);
const showCashOut = ref(false);
const pendingSeat = ref(-1);

const myChips = computed(() => {
  if (gameStore.mySeatIdx === -1 || !gameStore.tableState) return 0;
  return gameStore.tableState.players[gameStore.mySeatIdx]?.chips || 0;
});

onMounted(async () => {
  if (!authStore.isLoggedIn) {
    router.push('/login');
    return;
  }
  
  try {
    const ticket = await authStore.getTicket();
    connect(ticket);
    
    // If already connected (e.g. from Lobby), we need to join now because watch won't fire
    if (status.value === 'connected') {
        send(WSAction.GetBalance);
        const tableId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;
        send(WSAction.JoinTable, { table_id: tableId });
    }
  } catch (e) {
    console.error(e);
    router.push('/login');
  }
});

watch(status, (newVal) => {
  if (newVal === 'connected') {
    send(WSAction.GetBalance);
    const tableId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;
    send(WSAction.JoinTable, { table_id: tableId });
  }
});

onUnmounted(() => {
  disconnect();
});

function goBack() {
  if (gameStore.mySeatIdx !== -1) {
    showCashOut.value = true;
  } else {
    send(WSAction.LeaveTable);
    router.push('/lobby');
  }
}

function onSitRequest(seatIdx: number) {
  pendingSeat.value = seatIdx;
  showBuyIn.value = true;
}

const pendingBuyInAmount = ref(0);

function handleBuyIn(amount: number) {
  showBuyIn.value = false;
  pendingBuyInAmount.value = amount;
  send(WSAction.SitDown, { seat_no: pendingSeat.value });
  // We wait for mySeatIdx to update (meaning successful sit)
}

// Watch for successful sit to trigger buy-in
watch(() => gameStore.mySeatIdx, (newSeat) => {
  if (newSeat !== -1 && pendingBuyInAmount.value > 0) {
    send(WSAction.BuyIn, { amount: pendingBuyInAmount.value });
    pendingBuyInAmount.value = 0;
  }
});

function handleCashOut() {
  showCashOut.value = false;
  // Send commands in order. Server should handle them sequentially if sent over TCP.
  // But we want to ensure we don't route away until we are "done".
  
  // Logic: 
  // 1. CashOut
  // 2. StandUp
  // 3. LeaveTable
  // 4. Router push
  
  // To strictly wait, we'd need response correlation.
  // For MVP, we can chain them with small delays to allow server state updates to propagate back,
  // or just send them and assume success, but delay routing.
  
  send(WSAction.CashOut);
  // Give it a moment? Or just fire next? 
  // If we stand up immediately after cashout req, server might error if cashout isn't done?
  // Usually server handles message queue sequentially.
  
  setTimeout(() => {
      send(WSAction.StandUp);
      setTimeout(() => {
          send(WSAction.LeaveTable);
          setTimeout(() => {
              router.push('/lobby');
          }, 200);
      }, 200);
  }, 200);
}
</script>

<style scoped>
.table-view {
  width: 100vw;
  height: 100vh;
  background: #34495e;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-header {
  height: 50px;
  background: #2c3e50;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  color: white;
  z-index: 10;
}

.nav-btn {
  background: #7f8c8d;
  border: none;
  color: white;
  padding: 5px 15px;
  border-radius: 4px;
  cursor: pointer;
}

.nav-btn.red {
  background: #c0392b;
}

.game-area {
  flex: 1;
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
}

.hand-cards-container {
  position: absolute;
  bottom: 80px; /* Above action panel */
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
}

.connection-overlay {
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.7);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: white;
  z-index: 999;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #3498db;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
