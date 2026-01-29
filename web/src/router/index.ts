import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import LobbyView from '../views/LobbyView.vue'
import TableView from '../views/TableView.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            redirect: '/lobby'
        },
        {
            path: '/login',
            name: 'login',
            component: LoginView
        },
        {
            path: '/lobby',
            name: 'lobby',
            component: LobbyView,
            meta: { requiresAuth: true }
        },
        {
            path: '/table/:id',
            name: 'table',
            component: TableView,
            meta: { requiresAuth: true }
        }
    ]
})

router.beforeEach((to, _from, next) => {
    const authStore = useAuthStore()
    // Ensure init check might be needed if state is lost on refresh but localStorage has token
    // The store init handles localStorage reading in its definition
    if (to.meta.requiresAuth && !authStore.isLoggedIn) {
        next('/login')
    } else {
        next()
    }
})

export default router
