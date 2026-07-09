import { Platform, Alert, Linking } from 'react-native';
import { check, request, PERMISSIONS, RESULTS } from 'react-native-permissions';
import { DatabaseService } from './DatabaseService';

/**
 * PermissionsGuard.ts
 * 
 * Ensures the app has critical permissions (SMS, Phone State) to function.
 */

export const PermissionsGuard = {
    checkAndRequest: async (): Promise<boolean> => {
        if (Platform.OS !== 'android') return true;

        const permissions = [
            PERMISSIONS.ANDROID.SEND_SMS,
            PERMISSIONS.ANDROID.READ_PHONE_STATE,
        ];

        // Only add POST_NOTIFICATIONS if available in the library/OS version
        const androidPermissions = PERMISSIONS.ANDROID as any;
        if (androidPermissions.POST_NOTIFICATIONS) {
            permissions.push(androidPermissions.POST_NOTIFICATIONS);
        }

        let allGranted = true;

        for (const permission of permissions) {
            const status = await check(permission);

            if (status !== RESULTS.GRANTED) {
                const result = await request(permission);
                if (result !== RESULTS.GRANTED) {
                    allGranted = false;
                    DatabaseService.addLog(null, 'PERMISSION_DENIED', `User denied: ${permission}`);
                } else {
                    DatabaseService.addLog(null, 'PERMISSION_GRANTED', `Permission granted: ${permission}`);
                }
            }
        }

        if (!allGranted) {
            Alert.alert(
                'صلاحيات مطلوبة',
                'يحتاج هذا التطبيق إلى صلاحيات الرسائل (SMS) وحالة الهاتف للعمل كبوابة إرسال. يرجى تفعيلها من الإعدادات.',
                [
                    { text: 'إلغاء', style: 'cancel' },
                    { text: 'فتح الإعدادات', onPress: () => Linking.openSettings() }
                ]
            );
        }

        return allGranted;
    }
};
