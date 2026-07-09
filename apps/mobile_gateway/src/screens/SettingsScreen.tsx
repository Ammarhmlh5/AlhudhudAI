import React, { useState } from 'react';
import { View, Text, StyleSheet, TextInput, TouchableOpacity, ScrollView, Alert, StatusBar, KeyboardAvoidingView, Platform, Switch } from 'react-native';
import { useAppStore } from '../services/StorageService';
import { BackgroundServiceControl } from '../services/BackgroundServiceControl';
import { BottomTabNavigator } from '../components/BottomTabNavigator';
import { DatabaseService } from '../services/DatabaseService';

/**
 * SettingsScreen.tsx
 * 
 * Screen to configure API URL and Key, with Arabic localization.
 */

export const SettingsScreen = ({ navigation }: any) => {
    const store = useAppStore();
    const [url, setUrl] = useState(store.apiUrl);

    const handleSave = () => {
        if (!url) {
            Alert.alert('تنبيه', 'يرجى إدخال رابط الخادم بشكل صحيح.');
            return;
        }

        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            Alert.alert('خطأ', 'الرابط يجب أن يبدأ بـ http:// أو https://');
            return;
        }

        store.setApiUrl(url);
        Alert.alert('تم بنجاح', 'تم حفظ الإعدادات بنجاح.');
    };

    return (
        <KeyboardAvoidingView
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            style={styles.container}
        >
            <StatusBar barStyle="light-content" />
            <View style={styles.header}>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.backText}>← رجوع</Text>
                </TouchableOpacity>
                <Text style={styles.title}>الإعدادات</Text>
                <View style={{ width: 40 }} />
            </View>

            <ScrollView style={styles.content}>
                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>إعدادات الخادم (API)</Text>

                    <Text style={styles.label}>رابط الخادم (API URL)</Text>
                    <TextInput
                        style={styles.input}
                        value={url}
                        onChangeText={setUrl}
                        placeholder="https://api.example.com"
                        placeholderTextColor="#444"
                        autoCapitalize="none"
                    />

                    <TouchableOpacity style={styles.saveButton} onPress={handleSave}>
                        <Text style={styles.saveButtonText}>حفظ الإعدادات</Text>
                    </TouchableOpacity>
                </View>

                <View style={styles.divider} />

                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>إدارة الطاقة والوقاية</Text>
                    <Text style={styles.description}>
                        عند تفعيل "وضع العمل المستمر"، سيظل التطبيق يعمل في الخلفية ويستقبل الرسائل حتى عند تسجيل الخروج من واجهة التطبيق.
                    </Text>

                    <View style={styles.settingRow}>
                        <Text style={styles.settingLabel}>تفعيل وضع العمل المستمر</Text>
                        <Switch
                            value={store.continuousWork}
                            onValueChange={async (value) => {
                                store.setContinuousWork(value);
                                if (value) {
                                    await BackgroundServiceControl.requestBatteryOptimization();
                                }
                            }}
                            trackColor={{ false: '#333', true: '#FFB300' }}
                            thumbColor={store.continuousWork ? '#FFF' : '#AAA'}
                        />
                    </View>
                </View>

                <View style={styles.divider} />

                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>إدارة الحساب</Text>
                    <Text style={styles.description}>
                        لتسجيل الخروج وإلغاء ربط هذا الهاتف بالحساب الحالي.
                    </Text>

                    <TouchableOpacity
                        style={[styles.saveButton, { backgroundColor: '#F44336' }]}
                        onPress={() => {
                            Alert.alert(
                                'تسجيل الخروج',
                                store.continuousWork 
                                    ? 'هل أنت متأكد من رغبتك في تسجيل الخروج؟ (سيستمر التطبيق في العمل في الخلفية لاستقبال وإرسال الرسائل)'
                                    : 'هل أنت متأكد من رغبتك في تسجيل الخروج وإيقاف البوابة؟',
                                [
                                    { text: 'إلغاء', style: 'cancel' },
                                    {
                                        text: 'خروج',
                                        style: 'destructive',
                                        onPress: async () => {
                                            const keepRunning = store.continuousWork;
                                            if (!keepRunning) {
                                                if (store.isServiceRunning) {
                                                    await BackgroundServiceControl.stop();
                                                }
                                                const { wsService } = require('../services/WebSocketService');
                                                wsService.disconnect();
                                            }
                                            
                                            DatabaseService.addLog(null, 'LOGOUT_SUCCESS', `User logged out. Keep background service running: ${keepRunning}`);
                                            
                                            if (keepRunning) {
                                                store.logoutKeepCredentials();
                                            } else {
                                                store.logout();
                                            }
                                            navigation.replace('Login');
                                        }
                                    }
                                ]
                            );
                        }}
                    >
                        <Text style={styles.saveButtonText}>تسجيل الخروج</Text>
                    </TouchableOpacity>
                </View>

                <Text style={styles.note}>
                    ملاحظة: قد تتطلب بعض التغييرات إعادة تشغيل البوابة لتصبح فعالة.
                </Text>

                <View style={{ height: 100 }} />
            </ScrollView>

            <BottomTabNavigator navigation={navigation} activeTab="settings" />
        </KeyboardAvoidingView>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    header: {
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 20,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottomWidth: 1,
        borderBottomColor: '#333',
    },
    backText: {
        color: '#2196F3',
        fontSize: 16,
    },
    title: {
        fontSize: 18,
        fontWeight: 'bold',
        color: '#FFB300',
    },
    content: {
        flex: 1,
    },
    section: {
        padding: 20,
    },
    sectionTitle: {
        color: '#FFB300',
        fontSize: 16,
        fontWeight: 'bold',
        marginBottom: 20,
        textAlign: 'right',
    },
    label: {
        color: '#888',
        fontSize: 14,
        marginBottom: 8,
        textAlign: 'right',
    },
    input: {
        backgroundColor: '#1E1E1E',
        color: '#FFF',
        borderRadius: 8,
        padding: 12,
        marginBottom: 20,
        borderWidth: 1,
        borderColor: '#333',
        textAlign: 'right',
    },
    saveButton: {
        backgroundColor: '#2196F3',
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
        marginTop: 10,
    },
    saveButtonText: {
        color: '#FFF',
        fontWeight: 'bold',
        fontSize: 16,
    },
    description: {
        color: '#888',
        fontSize: 13,
        lineHeight: 20,
        marginBottom: 20,
        textAlign: 'right',
    },
    divider: {
        height: 1,
        backgroundColor: '#333',
        marginHorizontal: 20,
    },
    note: {
        color: '#444',
        fontSize: 12,
        textAlign: 'center',
        paddingHorizontal: 40,
        marginTop: 20,
    },
    settingRow: {
        flexDirection: 'row-reverse',
        justifyContent: 'space-between',
        alignItems: 'center',
        backgroundColor: '#1E1E1E',
        padding: 15,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: '#333',
        marginTop: 10,
    },
    settingLabel: {
        color: '#FFF',
        fontSize: 15,
        fontWeight: 'bold',
    },
});
