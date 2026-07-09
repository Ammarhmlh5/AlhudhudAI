import { NativeModules } from 'react-native';

/**
 * NativeSms.ts
 * 
 * JavaScript wrapper for the Native SMS Module.
 * Provides a clean interface to dispatch SMS via Android hardware.
 * 
 * Performance Requirements:
 * - Send SMS within 100ms
 * - Support 5-10 concurrent sends
 */

const { ALGatewaySmsModule } = NativeModules;
const SmsModule = ALGatewaySmsModule;

export interface SmsResult {
    success: boolean;
    error?: string;
    duration?: number;
}

export const NativeSms = {
    /**
     * Dispatches an SMS via the Android SmsManager.
     * @param id The internal message ID for tracking.
     * @param phoneNumber Destination phone number.
     * @param message Text content.
     * @returns Promise<boolean> - true if sent successfully, false otherwise
     */
    send: async (id: string, phoneNumber: string, message: string): Promise<boolean> => {
        const startTime = Date.now();
        
        try {
            // Validate inputs
            if (!phoneNumber || phoneNumber.trim().length === 0) {
                console.error(`Invalid phone number for message ${id}`);
                return false;
            }

            if (!message || message.trim().length === 0) {
                console.error(`Empty message content for ${id}`);
                return false;
            }

            // Call native module with timeout
            const result = await Promise.race([
                SmsModule.sendSMS(id, phoneNumber, message),
                new Promise<boolean>((_, reject) => 
                    setTimeout(() => reject(new Error('SMS send timeout')), 5000)
                ),
            ]);

            const duration = Date.now() - startTime;
            
            if (result) {
                console.log(`SMS sent successfully in ${duration}ms (ID: ${id})`);
                return true;
            } else {
                console.error(`SMS send failed (ID: ${id})`);
                return false;
            }
        } catch (e: any) {
            const duration = Date.now() - startTime;
            console.error(`SmsModule Native Error (${duration}ms): ${e.message}`);
            return false;
        }
    },

    /**
     * Check if SMS permissions are granted
     */
    checkPermissions: async (): Promise<boolean> => {
        try {
            if (SmsModule.checkPermissions) {
                return await SmsModule.checkPermissions();
            }
            return true; // Assume granted if method not available
        } catch (e) {
            console.error('Failed to check SMS permissions:', e);
            return false;
        }
    },

    /**
     * Request SMS permissions
     */
    requestPermissions: async (): Promise<boolean> => {
        try {
            if (SmsModule.requestPermissions) {
                return await SmsModule.requestPermissions();
            }
            return true;
        } catch (e) {
            console.error('Failed to request SMS permissions:', e);
            return false;
        }
    },
};
