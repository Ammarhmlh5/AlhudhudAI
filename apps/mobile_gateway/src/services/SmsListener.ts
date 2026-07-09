import { NativeEventEmitter, NativeModules, Platform } from 'react-native';
import { DatabaseService } from './DatabaseService';
import { wsService } from './WebSocketService';
import { useAppStore } from './StorageService';

/**
 * SmsListener.ts
 * 
 * Listens for Native events (Sent/Delivered) and updates the local DB and Server.
 */

export const initSmsListener = () => {
    const { ALGatewaySmsModule } = NativeModules;

    if (!ALGatewaySmsModule) {
        console.warn('[SmsListener] ALGatewaySmsModule native module not available. SMS events will not be captured.');
        return;
    }

    const smsEventEmitter = new NativeEventEmitter(ALGatewaySmsModule);

    // 1. Listen for "Sent" confirmation
    smsEventEmitter.addListener('onSmsSent', (event) => {
        console.log(`[NATIVE] SMS Sent event for ${event.id}: ${event.status}`);
        const isSuccess = event.status === 'SUCCESS';
        const status = isSuccess ? 'SENT' : 'FAILED';
        const errorCode = isSuccess ? undefined : event.status; // e.g., "ERROR_2"

        if (isSuccess) {
            useAppStore.getState().incrementSent();
        }

        DatabaseService.updateMessageStatus(event.id, status);
        DatabaseService.markAsUnsynced(event.id); // Mark for periodic sync until server ACKs
        DatabaseService.addLog(event.id, `NATIVE_SENT_${status}`, `Result code: ${event.status}`);

        const ok = wsService.sendStatusReport(event.id, status, errorCode);
        if (ok) DatabaseService.markAsSynced(event.id);
    });

    // 2. Listen for "Delivered" confirmation (Carrier ACK)
    smsEventEmitter.addListener('onSmsDelivered', (event) => {
        console.log(`[NATIVE] SMS Delivered event for ${event.id}, status: ${event.status}, reason: ${event.reason || 'N/A'}`);

        let status = 'DELIVERED';
        if (event.status === 'FAILED') status = 'FAILED';

        DatabaseService.updateMessageStatus(event.id, status);
        DatabaseService.markAsUnsynced(event.id);

        const ok = wsService.sendStatusReport(event.id, status, event.reason);
        if (ok) DatabaseService.markAsSynced(event.id);
    });
};
