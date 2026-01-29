import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useWalletStore = defineStore('wallet', () => {
    const balance = ref(0);

    // Updates balance from server event or API
    function setBalance(amount: number) {
        balance.value = amount;
    }

    function deduct(amount: number) {
        if (balance.value >= amount) {
            balance.value -= amount;
        }
    }

    function add(amount: number) {
        balance.value += amount;
    }

    return {
        balance,
        setBalance,
        deduct,
        add
    };
});
