import { create } from 'zustand';
import { api, setAccessToken, setSessionExpiredHandler } from './api';

// Remove credentials persisted by the pre-Phase-1 session implementation.
localStorage.removeItem('stocker_access');
localStorage.removeItem('stocker_user');

let restorePromise;

export const useAppStore = create((set, get) => ({
    user: null,
    sessionReady: false,
    theme: localStorage.getItem('stocker_theme') ?? 'dark',
    setSession: (token, user) => {
        setAccessToken(token);
        set({ user, sessionReady: true });
    },
    restoreSession: () => {
        if (!restorePromise) {
            restorePromise = api.restore()
                .then(({ data: user }) => set({ user, sessionReady: true }))
                .catch(() => { setAccessToken(''); set({ user: null, sessionReady: true }); })
                .finally(() => { restorePromise = undefined; });
        }
        return restorePromise;
    },
    clearSession: () => {
        setAccessToken('');
        set({ user: null, sessionReady: true });
    },
    logout: async () => {
        try { await api.logout(); }
        finally { get().clearSession(); }
    },
    toggleTheme: () => {
        const theme = get().theme === 'dark' ? 'light' : 'dark';
        localStorage.setItem('stocker_theme', theme);
        set({ theme });
    }
}));

setSessionExpiredHandler(() => useAppStore.getState().clearSession());
