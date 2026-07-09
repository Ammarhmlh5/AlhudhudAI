import { useAppStore } from './StorageService';
import axios from 'axios';
import { DatabaseService } from './DatabaseService';
import DeviceInfo from 'react-native-device-info';

/**
 * DeviceManager.ts
 * 
 * Manages the unique device serial number (UUID) and security logic.
 * Ensures the device is not banned before allowing SMS operations.
 */

export const DeviceManager = {
    /**
     * Retrieves a persistent unique ID for this hardware.
     */
    initializeIdentifier: async () => {
        const { deviceUuid, setDeviceUuid } = useAppStore.getState();

        if (!deviceUuid) {
            try {
                // Get real hardware ID
                const uniqueId = await DeviceInfo.getUniqueId();
                setDeviceUuid(uniqueId);
                DatabaseService.addLog(null, 'DEVICE_INIT', `Device Hardware ID initialized: ${uniqueId}`);
                return uniqueId;
            } catch (error) {
                // Fallback to more robust random ID if real hardware ID fails
                const timestamp = Date.now().toString(36).toUpperCase();
                const randomPart = Math.random().toString(36).substring(2, 10).toUpperCase();
                const fallbackId = `HW-${timestamp}-${randomPart}`;
                setDeviceUuid(fallbackId);
                return fallbackId;
            }
        }

        return deviceUuid;
    },

    /**
     * Checks with the central hub if this device is still authorized to work.
     * Part of the "Remote Kill Switch" feature.
     */
    checkAuthorization: async () => {
        const { deviceUuid, apiUrl, setBanned } = useAppStore.getState();

        try {
            const response = await axios.get(`${apiUrl}/api/v1/device/check-status`, {
                params: { device_id: deviceUuid }
            });

            if (response.data.status === 'BANNED' || response.data.status === 'LOCKED') {
                setBanned(true);
                DatabaseService.addLog(null, 'SECURITY_ALERT', 'Device has been remotely DISABLED by administrator.');
                return false;
            }

            setBanned(false);
            return true;
        } catch (err) {
            console.warn('Authorization check failed (Offline mode):', err);
            // In offline mode, we allow operation unless it was already banned
            return !useAppStore.getState().isBanned;
        }
    }
};
