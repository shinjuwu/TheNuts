<template>
  <div class="login-view">
    <div class="auth-card">
      <h1>The Nuts Poker</h1>
      
      <div class="tabs">
        <button :class="{ active: isLogin }" @click="isLogin = true">Login</button>
        <button :class="{ active: !isLogin }" @click="isLogin = false">Register</button>
      </div>
      
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label>Username</label>
          <input v-model="username" type="text" required />
        </div>
        
        <div class="form-group" v-if="!isLogin">
          <label>Email</label>
          <input v-model="email" type="email" required />
        </div>
        
        <div class="form-group">
          <label>Password</label>
          <input v-model="password" type="password" required minlength="6" />
        </div>
        
        <div class="error" v-if="error">{{ error }}</div>
        
        <button type="submit" :disabled="loading">
          {{ isLogin ? 'Login' : 'Register' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();

const isLogin = ref(true);
const username = ref('');
const email = ref('');
const password = ref('');
const error = ref('');
const loading = ref(false);

async function handleSubmit() {
  error.value = '';
  loading.value = true;
  
  try {
    if (isLogin.value) {
      await authStore.login(username.value, password.value);
      router.push('/lobby');
    } else {
      await authStore.register(username.value, email.value, password.value);
      isLogin.value = true;
      error.value = 'Registration successful! Please login.';
      // optionally auto login
    }
  } catch (e: any) {
    error.value = e.message || 'Authentication failed';
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.login-view {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #2c3e50;
  color: white;
}

.auth-card {
  background: #34495e;
  padding: 30px;
  border-radius: 8px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 4px 15px rgba(0,0,0,0.3);
}

.tabs {
  display: flex;
  margin-bottom: 20px;
  border-bottom: 2px solid #2c3e50;
}

.tabs button {
  flex: 1;
  padding: 10px;
  background: none;
  border: none;
  color: #bdc3c7;
  cursor: pointer;
  font-weight: bold;
}

.tabs button.active {
  color: #3498db;
  border-bottom: 2px solid #3498db;
  margin-bottom: -2px;
}

.form-group {
  margin-bottom: 15px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
}

.form-group input {
  width: 100%;
  padding: 8px;
  border-radius: 4px;
  border: 1px solid #7f8c8d;
  background: #2c3e50;
  color: white;
}

button[type="submit"] {
  width: 100%;
  padding: 10px;
  background: #27ae60;
  border: none;
  border-radius: 4px;
  color: white;
  font-weight: bold;
  cursor: pointer;
}

button[type="submit"]:disabled {
  background: #7f8c8d;
  cursor: not-allowed;
}

.error {
  color: #e74c3c;
  margin-bottom: 15px;
  text-align: center;
}
</style>
