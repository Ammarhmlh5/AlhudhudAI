/**
 * NotificationSettingsService.ts
 * 
 * Handles fetching and caching notification settings from the server.
 * Provides tenant isolation and configurable notification preferences.
 */

import axios from 'axios';
import { useAppStore } from './StorageService';

export interface NotificationPreference {
    enabled: boolean;
}

export interface NotificationPreferences {
    reconnection_alert: NotificationPreference;      // إشعار عند إعادة الاتصال
    package_expiry_warning: NotificationPreference;  // تحذير قرب انتهاء الباقة
    package_expiry_alert: NotificationPreference;    // إشعار انتهاء الباقة
    low_balance_warning: NotificationPreference;     // تحذير انخفاض الرصيد
    message_pending_alert: NotificationPreference;   // رسائل معلقة
    delivery_failed_alert: NotificationPreference;   // فشل التسليم
    system_update: NotificationPreference;           // تحديثات النظام
    new_message: NotificationPreference;             // رسالة جديدة
}

class NotificationSettingsService {
    private preferences: NotificationPreferences | null = null;
    private lastFetch: number = 0;
    private cacheTimeout: number = 5 * 60 * 1000; // 5 minutes

    /**
     * Fetch notification preferences from server
     */
    async fetchPreferences(): Promise<NotificationPreferences | null> {
        const store = useAppStore.getState();
        
        if (!store.apiUrl) {
            console.warn('API URL not configured');
            return null;
        }

        try {
            const config = { headers: {} as Record<string, string> };
            if (store.token) config.headers['Authorization'] = `Bearer ${store.token}`;
            else if (store.apiKey) config.headers['X-API-Key'] = store.apiKey;

            const response = await axios.get(
                `${store.apiUrl}/api/v1/notifications/preferences`,
                config
            );

            this.preferences = response.data as NotificationPreferences;
            this.lastFetch = Date.now();

            console.log('✅ Notification preferences fetched:', this.preferences);
            return this.preferences;
        } catch (error: any) {
            console.error('Failed to fetch notification preferences:', error.message);
            return null;
        }
    }

    /**
     * Get cached preferences or fetch from server
     */
    async getPreferences(forceRefresh = false): Promise<NotificationPreferences | null> {
        // Return cached if valid
        if (!forceRefresh && this.preferences && (Date.now() - this.lastFetch) < this.cacheTimeout) {
            return this.preferences;
        }

        return await this.fetchPreferences();
    }

    /**
     * Check if a specific notification type is enabled
     */
    async isEnabled(notificationType: keyof NotificationPreferences): Promise<boolean> {
        const preferences = await this.getPreferences();
        
        if (!preferences) {
            // Default to true if server is not reachable
            return notificationType !== 'new_message';
        }

        return preferences[notificationType]?.enabled ?? true;
    }

    /**
     * Check if reconnection alert is enabled
     */
    async isReconnectionAlertEnabled(): Promise<boolean> {
        return await this.isEnabled('reconnection_alert');
    }

    /**
     * Check if package expiry warning is enabled
     */
    async isPackageExpiryWarningEnabled(): Promise<boolean> {
        return await this.isEnabled('package_expiry_warning');
    }

    /**
     * Check if package expiry alert is enabled
     */
    async isPackageExpiryAlertEnabled(): Promise<boolean> {
        return await this.isEnabled('package_expiry_alert');
    }

    /**
     * Check if low balance warning is enabled
     */
    async isLowBalanceWarningEnabled(): Promise<boolean> {
        return await this.isEnabled('low_balance_warning');
    }

    /**
     * Check if message pending alert is enabled
     */
    async isMessagePendingAlertEnabled(): Promise<boolean> {
        return await this.isEnabled('message_pending_alert');
    }

    /**
     * Check if delivery failed alert is enabled
     */
    async isDeliveryFailedAlertEnabled(): Promise<boolean> {
        return await this.isEnabled('delivery_failed_alert');
    }

    /**
     * Check if system update notifications are enabled
     */
    async isSystemUpdateEnabled(): Promise<boolean> {
        return await this.isEnabled('system_update');
    }

    /**
     * Check if new message notifications are enabled
     */
    async isNewMessageEnabled(): Promise<boolean> {
        return await this.isEnabled('new_message');
    }

    /**
     * Update notification preferences on server
     */
    async updatePreferences(preferences: Partial<NotificationPreferences>): Promise<boolean> {
        const store = useAppStore.getState();
        
        if (!store.apiUrl) {
            console.warn('API URL not configured');
            return false;
        }

        try {
            const config = { headers: {} as Record<string, string> };
            if (store.token) config.headers['Authorization'] = `Bearer ${store.token}`;
            else if (store.apiKey) config.headers['X-API-Key'] = store.apiKey;

            await axios.put(
                `${store.apiUrl}/api/v1/notifications/preferences`,
                preferences,
                config
            );

            // Invalidate cache
            this.preferences = null;
            this.lastFetch = 0;

            console.log('✅ Notification preferences updated');
            return true;
        } catch (error: any) {
            console.error('Failed to update notification preferences:', error.message);
            return false;
        }
    }

    /**
     * Clear cached preferences
     */
    clearCache(): void {
        this.preferences = null;
        this.lastFetch = 0;
    }
}

export const notificationSettingsService = new NotificationSettingsService();