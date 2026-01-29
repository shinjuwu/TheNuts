import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import {
    type TableState,
    type Card,
    type WSResponse,
    WSEventType,
    GameActionType,
    PlayerStatus
} from '@/types';
import { useAuthStore } from './auth';
import { useWalletStore } from './wallet';

export const useGameStore = defineStore('game', () => {
    const tableState = ref<TableState | null>(null);
    const myCards = ref<Card[]>([]);
    const validActions = ref<GameActionType[]>([]);
    const amountToCall = ref(0);
    const minRaise = ref(0);
    const isMyTurn = ref(false);
    const timeRemaining = ref(0);
    const lastError = ref<string>('');

    const authStore = useAuthStore();
    const walletStore = useWalletStore();

    const mySeatIdx = computed(() => {
        if (!tableState.value || !authStore.username) return -1;
        const player = tableState.value.players.find(p => p && p.name === authStore.username);
        return player ? player.seatIdx : -1;
    });

    function handleEvent(res: WSResponse) {
        console.log('GameStore handling:', res.type, res.payload);

        switch (res.type) {
            case WSEventType.TableState:
                tableState.value = res.payload;
                break;

            case WSEventType.BuyInSuccess:
                if (res.payload.wallet_balance !== undefined) {
                    walletStore.setBalance(res.payload.wallet_balance);
                }
                break;
            case WSEventType.CashOutSuccess:
                if (res.payload.wallet_balance !== undefined) {
                    walletStore.setBalance(res.payload.wallet_balance);
                }
                break;
            case WSEventType.BalanceInfo:
                if (res.payload.wallet_balance !== undefined) {
                    walletStore.setBalance(res.payload.wallet_balance);
                }
                break;

            case WSEventType.HandStart:
                resetHand();
                break;

            case WSEventType.HoleCards:
                myCards.value = res.payload.cards;
                break;

            case WSEventType.YourTurn:
                isMyTurn.value = true;
                validActions.value = res.payload.valid_actions;
                amountToCall.value = res.payload.amount_to_call;
                minRaise.value = res.payload.min_raise;
                timeRemaining.value = res.payload.time_remaining;
                break;

            case WSEventType.PlayerAction:
                const { seat_idx, action, amount } = res.payload;
                updatePlayerAction(seat_idx, action, amount);

                if (seat_idx === mySeatIdx.value) {
                    isMyTurn.value = false;
                    validActions.value = [];
                }
                break;

            case WSEventType.CommunityCards:
                if (tableState.value) {
                    // Assuming payload.cards contains the new set of community cards
                    tableState.value.communityCards = res.payload.cards;
                }
                break;

            case WSEventType.ShowdownResult:
                // TODO: Handle showing winner
                break;

            case WSEventType.Error:
                lastError.value = res.payload.message || 'Unknown error';
                break;
        }
    }

    function resetHand() {
        myCards.value = [];
        isMyTurn.value = false;
        validActions.value = [];
        if (tableState.value) {
            tableState.value.communityCards = [];
            tableState.value.potTotal = 0;
            tableState.value.players.forEach(p => {
                if (p) {
                    p.currentBet = 0;
                    p.hasActed = false;
                    if (p.status !== PlayerStatus.SittingOut) {
                        p.status = PlayerStatus.Playing;
                    }
                }
            });
        }
    }

    function updatePlayerAction(seatIdx: number, action: GameActionType, amount: number) {
        if (!tableState.value) return;
        const p = tableState.value.players[seatIdx];
        if (p) {
            if (action === GameActionType.Fold) {
                p.status = PlayerStatus.Folded;
            }
            // Assuming 'amount' is the amount added to the pot in this action
            if (amount > 0) {
                p.currentBet += amount;
                p.chips -= amount;
                tableState.value.potTotal += amount;
            }
            p.hasActed = true;
        }
    }

    return {
        tableState,
        myCards,
        validActions,
        amountToCall,
        minRaise,
        isMyTurn,
        timeRemaining,
        lastError,
        mySeatIdx,
        handleEvent
    };
});
