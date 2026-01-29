export type Card = string; // e.g., "AS", "KH"

export const PlayerStatus = {
    SittingOut: "SittingOut",
    Playing: "Playing",
    Folded: "Folded",
    AllIn: "AllIn",
    Left: "Left"
} as const;
export type PlayerStatus = typeof PlayerStatus[keyof typeof PlayerStatus];

export interface Player {
    id: string;
    name: string;
    seatIdx: number;
    chips: number;
    currentBet: number;
    status: PlayerStatus;
    hasActed: boolean;
    avatar?: string;
}

export const GameState = {
    Waiting: "Waiting",
    PreFlop: "PreFlop",
    Flop: "Flop",
    Turn: "Turn",
    River: "River",
    Showdown: "Showdown",
    Finished: "Finished"
} as const;
export type GameState = typeof GameState[keyof typeof GameState];

export interface TableState {
    tableId: string;
    state: GameState;
    players: (Player | null)[]; // 9 seats, null if empty
    communityCards: Card[];
    dealerPos: number;
    currentPos: number; // Action on this seat index
    minBet: number;
    potTotal: number;
    smallBlind: number;
    bigBlind: number;
}

// WebSocket 請求 (Client -> Server)
export const WSAction = {
    JoinTable: "JOIN_TABLE",
    LeaveTable: "LEAVE_TABLE",
    SitDown: "SIT_DOWN",
    StandUp: "STAND_UP",
    BuyIn: "BUY_IN",
    CashOut: "CASH_OUT",
    GameAction: "GAME_ACTION",
    GetBalance: "GET_BALANCE"
} as const;
export type WSAction = typeof WSAction[keyof typeof WSAction];

export const GameActionType = {
    Fold: "FOLD",
    Check: "CHECK",
    Call: "CALL",
    Bet: "BET",
    Raise: "RAISE",
    AllIn: "ALL_IN"
} as const;
export type GameActionType = typeof GameActionType[keyof typeof GameActionType];

export interface WSRequest {
    action: WSAction;
    table_id?: string;
    amount?: number;
    seat_no?: number;
    game_action?: GameActionType;
    // Add trace info if needed
    timestamp?: string;
    trace_id?: string;
}

// WebSocket 回應 (Server -> Client)
export const WSEventType = {
    HandStart: "HAND_START",
    HoleCards: "HOLE_CARDS",
    BlindsPosted: "BLINDS_POSTED",
    YourTurn: "YOUR_TURN",
    PlayerAction: "PLAYER_ACTION",
    CommunityCards: "COMMUNITY_CARDS",
    ShowdownResult: "SHOWDOWN_RESULT",
    WinByFold: "WIN_BY_FOLD",
    HandEnd: "HAND_END",
    ActionTimeout: "ACTION_TIMEOUT",
    TableState: "TABLE_STATE",
    Error: "ERROR",
    // New types
    BuyInSuccess: "BUY_IN_SUCCESS",
    CashOutSuccess: "CASH_OUT_SUCCESS",
    JoinTableSuccess: "JOIN_TABLE_SUCCESS",
    LeaveTableSuccess: "LEAVE_TABLE_SUCCESS",
    SitDownSuccess: "SIT_DOWN_SUCCESS",
    StandUpSuccess: "STAND_UP_SUCCESS",
    BalanceInfo: "BALANCE_INFO"
} as const;
export type WSEventType = typeof WSEventType[keyof typeof WSEventType];

export interface WSResponse<T = any> {
    type: WSEventType;
    payload: T;
    timestamp: string;
    trace_id?: string;
}

// Payloads
export interface HandStartPayload {
    hand_id: string;
    small_blind_pos: number;
    big_blind_pos: number;
}

export interface HoleCardsPayload {
    cards: Card[];
}

export interface YourTurnPayload {
    valid_actions: GameActionType[];
    amount_to_call: number;
    min_raise: number;
    time_remaining: number; // seconds
}

export interface PlayerActionPayload {
    seat_idx: number;
    action: GameActionType;
    amount: number;
}

export interface CommunityCardsPayload {
    street: string; // "Flop", "Turn", "River"
    cards: Card[];
}

export interface ShowdownResultPayload {
    winners: { seat_idx: number; amount: number; hand_rank: string }[];
    hands: { seat_idx: number; cards: Card[]; hand_rank: string }[];
}

export interface WinByFoldPayload {
    winner_seat_idx: number;
    amount: number;
}
