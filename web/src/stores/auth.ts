import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import * as api from '@/services/api';
import type { Player } from '@/types';

export const useAuthStore = defineStore('auth', () => {
    const token = ref<string>(localStorage.getItem('token') || '');
    const player = ref<Partial<Player> | null>(null);
    const username = ref<string>(localStorage.getItem('username') || '');

    const isLoggedIn = computed(() => !!token.value);

    function setToken(newToken: string) {
        token.value = newToken;
        localStorage.setItem('token', newToken);
    }

    function setUser(user: any) {
        // Assuming backend returns some user info on login
        player.value = user;
        username.value = user.username;
        localStorage.setItem('username', user.username);
    }

    async function login(name: string, pass: string) {
        const data = await api.login(name, pass);
        setToken(data.token);
        setUser({ id: data.player_id, username: data.username, displayName: data.display_name });
    }

    async function register(name: string, email: string, pass: string) {
        await api.register(name, email, pass);
    }

    function logout() {
        token.value = '';
        player.value = null;
        username.value = '';
        localStorage.removeItem('token');
        localStorage.removeItem('username');
    }

    async function getTicket(): Promise<string> {
        if (!token.value) throw new Error('Not logged in');
        const data = await api.getTicket(token.value);
        return data.ticket;
    }

    return {
        token,
        player,
        username,
        isLoggedIn,
        login,
        register,
        logout,
        getTicket
    };
});
