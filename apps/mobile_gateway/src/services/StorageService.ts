import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';

interface AppState {
    deviceUuid: string | null;
    apiUrl: string;
    apiKey: string | null;
    token: string | null;
    isAuthenticated: boolean;
    isBanned: boolean;
    isServiceRunning: boolean;
    totalSent: number;
    balance: number;
    isConnected: boolean;
    continuousWork: boolean;
    unreadNotifications: number; // عدد الإشعارات غير المقروءة

    setDeviceUuid: (uuid: string | null) => void;
    setApiUrl: (url: string) => void;
    setApiKey: (key: string) => void;
    setToken: (token: string | null) => void;
    setAuthenticated: (status: boolean) => void;
    setBanned: (status: boolean) => void;
    toggleService: (status: boolean) => void;
    incrementSent: () => void;
    setBalance: (val: number) => void;
    setConnected: (status: boolean) => void;
    setContinuousWork: (status: boolean) => void;
    setUnreadNotifications: (count: number) => void;
    incrementUnreadNotifications: () => void;
    clearUnreadNotifications: () => void;
    logout: () => void;
    logoutKeepCredentials: () => void;
}

export const useAppStore = create<AppState>()(
    persist(
        (set) => ({
            deviceUuid: null,
            apiUrl: 'http://192.168.8.136:8085', // Go backend API port
            apiKey: null,
            token: null,
            isAuthenticated: false,
            isBanned: false,
            isServiceRunning: false,
            totalSent: 0,
            balance: 0,
            isConnected: false,
            continuousWork: false,
            unreadNotifications: 0,

            setDeviceUuid: (uuid) => set({ deviceUuid: uuid }),
            setApiUrl: (url) => set({ apiUrl: url }),
            setApiKey: (key) => set({ apiKey: key }),
            setToken: (token) => set({ token }),
            setAuthenticated: (status) => set({ isAuthenticated: status }),
            setBanned: (status) => set({ isBanned: status }),
            toggleService: (status) => set({ isServiceRunning: status }),
            incrementSent: () => set((state) => ({ totalSent: state.totalSent + 1 })),
            setBalance: (val) => set({ balance: val }),
            setConnected: (status) => set({ isConnected: status }),
            setContinuousWork: (status) => set({ continuousWork: status }),
            setUnreadNotifications: (count) => set({ unreadNotifications: count }),
            incrementUnreadNotifications: () => set((state) => ({ 
                unreadNotifications: state.unreadNotifications + 1 
            })),
            clearUnreadNotifications: () => set({ unreadNotifications: 0 }),
            logout: () => set({
                token: null,
                isAuthenticated: false,
                apiKey: null,
                deviceUuid: null,
                isServiceRunning: false,
                isConnected: false,
                continuousWork: false,
                unreadNotifications: 0
            }),
            logoutKeepCredentials: () => set({
                token: null,
                isAuthenticated: false
            }),
        }),
        {
            name: 'alhudhudai-storage',
            storage: createJSONStorage(() => AsyncStorage),
        }
    )
);
