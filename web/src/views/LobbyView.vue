<template>
  <div class="lobby-view">
    <header>
      <div class="brand">The Nuts</div>
      <div class="user-info">
        <span>👤 {{ authStore.username }}</span>
        <!-- Wallet store might typically fetch balance from API. For MVP we assume we have it or it's 0. -->
        <span>💰 {{ walletStore.balance }}</span>
        <button @click="logout" class="btn-logout">Logout</button>
      </div>
    </header>
    
    <main>
      <h1>Lobby</h1>
      
      <div class="tables-list">
        <!-- Mock Tables -->
        <div class="table-card">
          <div class="table-info">
            <h3>No Limit Hold'em - $1/$2</h3>
            <p>Table #1</p>
          </div>
          <div class="table-action">
            <button @click="joinTable('table1')">Join</button>
          </div>
        </div>
        
         <div class="table-card">
          <div class="table-info">
            <h3>No Limit Hold'em - $2/$4</h3>
            <p>Table #2</p>
          </div>
          <div class="table-action">
            <button @click="joinTable('table2')">Join</button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import { onMounted, onUnmounted, watch } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useWalletStore } from '@/stores/wallet';
import { useWebSocket } from '@/composables/useWebSocket';
import { WSAction } from '@/types';

const router = useRouter();
const authStore = useAuthStore();
const walletStore = useWalletStore();
const { connect, disconnect, status, send } = useWebSocket();

onMounted(async () => {
  if (authStore.isLoggedIn) {
     try {
       const ticket = await authStore.getTicket();
       connect(ticket);
     } catch (e) {
       console.error("Lobby connection failed", e);
     }
  }
});

watch(status, (newVal) => {
  if (newVal === 'connected') {
    // Request balance explicitly since it might not be pushed automatically
    send(WSAction.GetBalance);
  }
});

onUnmounted(() => {
  disconnect();
});

function logout() {
  authStore.logout();
  router.push('/login');
}

function joinTable(tableId: string) {
  router.push(`/table/${tableId}`);
}
</script>

<style scoped>
.lobby-view {
  min-height: 100vh;
  background: #2c3e50;
  color: white;
}

header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 30px;
  background: #34495e;
  box-shadow: 0 2px 5px rgba(0,0,0,0.2);
}

.brand {
  font-size: 20px;
  font-weight: bold;
}

.user-info {
  display: flex;
  gap: 20px;
  align-items: center;
}

.btn-logout {
  padding: 5px 10px;
  background: #c0392b;
  border: none;
  color: white;
  border-radius: 4px;
  cursor: pointer;
}

main {
  padding: 30px;
  max-width: 800px;
  margin: 0 auto;
}

.tables-list {
  display: grid;
  gap: 20px;
  margin-top: 20px;
}

.table-card {
  background: #ecf0f1;
  color: #2c3e50;
  padding: 20px;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.table-card button {
  padding: 10px 30px;
  background: #27ae60;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}
</style>
