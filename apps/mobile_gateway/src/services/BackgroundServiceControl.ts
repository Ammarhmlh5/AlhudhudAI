import { NativeModules } from 'react-native';

/**
 * BackgroundServiceControl.ts
 * 
 * JS interface to start/stop the Android Foreground Service.
 */

const { GatewayModule } = NativeModules;

export const BackgroundServiceControl = {
    start: async (): Promise<boolean> => {
        try {
            return await GatewayModule.startService();
        } catch (e) {
            console.error('Failed to start Foreground Service:', e);
            return false;
        }
    },

    stop: async (): Promise<boolean> => {
        try {
            return await GatewayModule.stopService();
        } catch (e) {
            console.error('Failed to stop Foreground Service:', e);
            return false;
        }
    },

    requestBatteryOptimization: async (): Promise<boolean> => {
        try {
            return await GatewayModule.requestBatteryOptimization();
        } catch (e) {
            console.warn('Failed to request battery optimization ignore:', e);
            return false;
        }
    },

    getSimInfo: async (): Promise<any[]> => {
        try {
            return await GatewayModule.getSimInfo();
        } catch (e) {
            console.error('Failed to get SIM info:', e);
            return [];
        }
    }
};
