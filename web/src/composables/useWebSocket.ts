import { ref } from 'vue';
import { useGameStore } from '@/stores/game';
import { type WSRequest, type WSResponse, WSAction } from '@/types';

// Global state to share connection across components
const socket = ref<WebSocket | null>(null);
const status = ref<'connecting' | 'connected' | 'disconnected'>('disconnected');
const reconnectAttempts = ref(0);
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let intentionalClose = false;

export function useWebSocket() {
    const gameStore = useGameStore();

    const connect = (ticket: string) => {
        if (socket.value?.readyState === WebSocket.OPEN) {
            status.value = 'connected';
            return;
        }

        if (status.value === 'connecting') return;

        intentionalClose = false;
        status.value = 'connecting';

        // Determine WS URL
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?ticket=${ticket}`;

        console.log('Connecting to WS:', wsUrl);
        const ws = new WebSocket(wsUrl);
        socket.value = ws;

        ws.onopen = () => {
            console.log('WS Connected');
            status.value = 'connected';
            reconnectAttempts.value = 0;
            if (reconnectTimer) clearTimeout(reconnectTimer);
        };

        ws.onmessage = (event) => {
            try {
                const response: WSResponse = JSON.parse(event.data);
                gameStore.handleEvent(response);
            } catch (e) {
                console.error('Failed to parse WS message:', event.data, e);
            }
        };

        ws.onclose = (event) => {
            console.log('WS Closed', event.code, event.reason);
            status.value = 'disconnected';
            socket.value = null;

            if (!intentionalClose) {
                attemptReconnect();
            }
        };

        ws.onerror = (error) => {
            console.error('WS Error:', error);
        };
    };

    const attemptReconnect = async () => {
        if (reconnectAttempts.value > 5) {
            console.log('Max reconnect attempts reached');
            return;
        }

        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 30000);
        console.log(`Reconnecting in ${delay}ms... (Attempt ${reconnectAttempts.value + 1})`);

        reconnectTimer = setTimeout(async () => {
            reconnectAttempts.value++;
            try {
                // Dynamic import to avoid potential circular dependency
                const { useAuthStore } = await import('@/stores/auth');
                const authStore = useAuthStore();
                const newTicket = await authStore.getTicket();
                connect(newTicket);
            } catch (e) {
                console.error("Failed to get ticket during reconnect:", e);
                attemptReconnect();
            }
        }, delay);
    };

    const send = (action: WSAction, payload: any = {}) => {
        if (socket.value && socket.value.readyState === WebSocket.OPEN) {
            const req: WSRequest = {
                action,
                ...payload
            };
            socket.value.send(JSON.stringify(req));
            console.log('WS Sent:', req);
        } else {
            console.warn('WS not connected, cannot send:', action);
        }
    };

    const disconnect = () => {
        intentionalClose = true;
        if (reconnectTimer) clearTimeout(reconnectTimer);
        socket.value?.close();
        socket.value = null;
        status.value = 'disconnected';
    };

    return {
        status,
        connect,
        send,
        disconnect
    };
}
