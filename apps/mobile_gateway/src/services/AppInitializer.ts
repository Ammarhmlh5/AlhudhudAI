import { initDatabase, DatabaseService } from './DatabaseService';
import { DeviceManager } from './DeviceManager';
import { wsService } from './WebSocketService';
import { useAppStore } from './StorageService';
import { smsQueue } from './SmsQueueManager';
import { BackgroundServiceControl } from './BackgroundServiceControl';
import { initSmsListener } from './SmsListener';
import { CodePushService } from './CodePushService';

import { PermissionsGuard } from './PermissionsGuard';

/**
 * AppInitializer.ts
 * 
 * Orchestrates the startup sequence of the application.
 */

export const AppInitializer = {
    run: async () => {
        try {
            console.log('[INIT] Starting ALHudhudAI Gateway...');

            // 0. Initialize DB (CRITICAL: Must be first so logs can be saved)
            initDatabase();
            DatabaseService.addLog(null, 'APP_START', 'Database initialized successfully.');
            console.log('[INIT] Database ready.');

            // 1. Permissions Guard
            console.log('[INIT] Checking permissions...');
            const permissionsOk = await PermissionsGuard.checkAndRequest();
            if (!permissionsOk) {
                console.warn('[INIT] Running with limited permissions.');
            }

            // 2. Initialize Listeners
            initSmsListener();
            console.log('[INIT] SMS Listeners ready.');
            CodePushService.checkForUpdates();

            // 3. Request Battery Optimization Ignore
            await BackgroundServiceControl.requestBatteryOptimization();

            // 4. Initialize Identity (Serial Number)
            const hardwareId = await DeviceManager.initializeIdentifier();
            DatabaseService.addLog(null, 'IDENTITY_INIT', `Device Hardware ID: ${hardwareId}`);

            // 5. Check Authentication & Registration
            const { isAuthenticated, deviceUuid } = useAppStore.getState();

            if (isAuthenticated && deviceUuid) {
                // 6. Check Admin Status (Remote Kill Switch)
                const authorized = await DeviceManager.checkAuthorization();
                if (!authorized) {
                    console.error('[INIT] Device is unauthorized/banned.');
                    return;
                }

                // 7. Auto-enable service if not already enabled
                useAppStore.getState().toggleService(true);
                console.log('[INIT] Service auto-enabled.');

                // 8. Start Background Foreground Service
                console.log('[INIT] Starting Background Gateway Service...');
                await BackgroundServiceControl.start();

                // 9. Connect WebSocket (always connect when authenticated)
                wsService.connect();

                // 10. Start SMS Queue (always start to process pending messages)
                smsQueue.start();
                console.log('[INIT] SMS Queue and WebSocket started.');
            } else {
                console.log('[INIT] Waiting for user login/setup.');
            }

            // 8. Periodic Maintenance
            setInterval(() => {
                console.log('[MAINTENANCE] Cleaning up old data...');
                DatabaseService.cleanupOldLogs(7);
                DatabaseService.cleanupOldEvents(7);
            }, 86400000);

            console.log('[INIT] Ready.');
        } catch (error) {
            console.error('[INIT] CRITICAL FAILURE DURING STARTUP:', error);
            // Even if everything fails, we want the UI to stop loading and hopefully show something
        }
    }
};
