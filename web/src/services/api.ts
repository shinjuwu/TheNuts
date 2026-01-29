const BASE_URL = '/api/auth';

interface LoginResponse {
    token: string;
    player_id: string;
    account_id: string;
    username: string;
    display_name: string;
}

interface TicketResponse {
    ticket: string;
}

export async function login(username: string, password: string): Promise<LoginResponse> {
    const res = await fetch(`${BASE_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.message || 'Login failed');
    }

    return res.json();
}

export async function register(username: string, email: string, password: string): Promise<void> {
    const res = await fetch(`${BASE_URL}/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, email, password })
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.message || 'Registration failed');
    }
}

export async function getTicket(token: string): Promise<TicketResponse> {
    const res = await fetch(`${BASE_URL}/ticket`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        }
    });

    if (!res.ok) {
        throw new Error('Failed to get ticket');
    }

    return res.json();
}
