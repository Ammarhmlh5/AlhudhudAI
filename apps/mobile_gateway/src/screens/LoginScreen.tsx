import React, { useState } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TextInput,
    TouchableOpacity,
    ActivityIndicator,
    Alert,
    StatusBar,
    KeyboardAvoidingView,
    Platform,
    ScrollView,
    Modal,
    Switch,
    Dimensions
} from 'react-native';
import { useAppStore } from '../services/StorageService';
import axios from 'axios';
import { DeviceManager } from '../services/DeviceManager';
import DeviceInfo from 'react-native-device-info';
import { DatabaseService } from '../services/DatabaseService';
import { wsService } from '../services/WebSocketService';
import { smsQueue } from '../services/SmsQueueManager';
import { BackgroundServiceControl } from '../services/BackgroundServiceControl';

const getErrorMessage = (error: any, fallback: string): string => {
    try {
        if (!error) return fallback;
        const errData = error.response?.data?.error;
        if (!errData) {
            const msg = error.message;
            if (typeof msg === 'string') return msg;
            if (typeof msg === 'object' && msg !== null) return JSON.stringify(msg);
            return fallback;
        }
        if (typeof errData === 'string') return errData;
        if (typeof errData === 'object') {
            const msg = errData.message || errData.code;
            if (typeof msg === 'string') return msg;
            return JSON.stringify(errData);
        }
        return String(errData) || fallback;
    } catch (e) {
        return fallback;
    }
};

export const LoginScreen = ({ navigation }: any) => {
    const store = useAppStore();
    const [mode, setMode] = useState<'login' | 'register'>('login');
    const [email, setEmail] = useState('');
    const [phone, setPhone] = useState('');
    const [countryCode, setCountryCode] = useState('+966');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [apiUrl, setApiUrl] = useState(store.apiUrl);
    const [loading, setLoading] = useState(false);

    // Automation Modal States
    const [showSetupModal, setShowSetupModal] = useState(false);
    const [setupType, setSetupType] = useState<'new' | 'replace' | null>(null);
    const [isPrimary, setIsPrimary] = useState(true);
    const [existingDevices, setExistingDevices] = useState<any[]>([]);
    const [tempAuthData, setTempAuthData] = useState<any>(null);
    const [selectedDeviceToReplace, setSelectedDeviceToReplace] = useState<string | null>(null);

    const handleSubmit = async () => {
        if (!email || !password || !apiUrl) {
            Alert.alert('خطأ', 'يرجى إدخال البيانات الأساسية');
            return;
        }

        if (mode === 'register') {
            if (!phone || !countryCode || !confirmPassword) {
                Alert.alert('خطأ', 'يرجى إكمال بيانات التسجيل');
                return;
            }
            if (password !== confirmPassword) {
                Alert.alert('خطأ', 'كلمات المرور غير متطابقة');
                return;
            }
        }

        setLoading(true);
        try {
            store.setApiUrl(apiUrl);

            if (mode === 'login') {
                const response = await axios.post(`${apiUrl}/api/v1/auth/login`, { email, password });
                await handleAuthSuccess(response.data);
            } else {
                const response = await axios.post(`${apiUrl}/api/v1/tenants/register`, {
                    email,
                    phone,
                    country_code: countryCode,
                    password,
                    confirm_password: confirmPassword
                });
                // After registration, the system returns typical login response
                await handleAuthSuccess(response.data, true);
            }
        } catch (error: any) {
            console.error('Submit error:', error);
            const message = getErrorMessage(error, 'فشل الاتصال بالخادم');
            Alert.alert(mode === 'login' ? 'فشل تسجيل الدخول' : 'فشل التسجيل', message);
        } finally {
            setLoading(false);
        }
    };

    const handleAuthSuccess = async (data: any, isNewRegistration = false) => {
        const { token } = data;
        store.setToken(token);
        store.setAuthenticated(true);
        DatabaseService.addLog(null, 'LOGIN_SUCCESS', `User ${email} authenticated successfully.`);

        try {
            const hardwareId = await DeviceManager.initializeIdentifier();
            store.setDeviceUuid(hardwareId);
            const config = { headers: { Authorization: `Bearer ${token}` } };
            console.log('[AUTH] Checking device status for hardware:', hardwareId);
            const devicesRes = await axios.get(`${apiUrl}/api/v1/devices/list`, config);
            const devices = devicesRes.data || [];
            const existing = devices.find((d: any) => d.hardware_id === hardwareId);

            if (existing) {
                // Device already exists, just continue
                console.log('[AUTH] Device recognized, syncing key.');
                store.setApiKey(existing.device_pair_key || 'synced');
                // Auto-start service immediately
                store.toggleService(true);
                await BackgroundServiceControl.start();
                wsService.connect();
                smsQueue.start();
                navigation.replace('Dashboard');
            } else if (isNewRegistration || !devices || devices.length === 0) {
                // New User or First Device -> Auto pair as primary without asking
                console.log('[AUTH] New hardware detected, attempting auto-pairing...');
                const model = await DeviceInfo.getModel();
                const deviceName = `${model} (Gateway)`;
                const devPhone = phone || data.tenant?.phone || '05xxxxxxxx';
                await autoPairDevice(token, hardwareId, deviceName, true, devPhone);
            } else {
                // Existing user WITH other devices, connecting a NEW unrecognized phone.
                // We should ask them what they want to do (New Device vs Replace).
                console.log('[AUTH] Unrecognized hardware for existing account, prompting setup.');
                setExistingDevices(devices);
                setTempAuthData(data);
                setShowSetupModal(true);
            }
        } catch (e: any) {
            console.error('[AUTH] Critical device check failure:', e.response?.data || e.message);
            // Engineering Fix: DON'T navigate to Dashboard if we can't verify/pair the device.
            // Clear auth and stay on login so the user knows something is wrong.
            store.setAuthenticated(false);
            store.setToken(null);

            const errorMsg = getErrorMessage(e, 'فشل التحقق من ملكية الجهاز. يرجى التأكد من أن حسابك مكتمل البيانات.');
            Alert.alert('خطأ في إعداد الجهاز', errorMsg);
        }
    };

    const autoPairDevice = async (token: string, hardwareId: string, name: string, primary: boolean, phoneNumber: string, replaceId?: string) => {
        setLoading(true);
        try {
            const config = { headers: { Authorization: `Bearer ${token}` } };
            const model = await DeviceInfo.getModel();

            // Priority for phone number: 
            // 1. Current registration input (phone)
            // 2. Fallback to existing store if available
            // 3. Last fallback dummy
            const devicePhone = phoneNumber || '05xxxxxxxx';

            console.log('[PAIR] Attempting to pair hardware:', hardwareId);
            const pairResponse = await axios.post(`${apiUrl}/api/v1/devices/pair`, {
                device_name: name || model,
                hardware_id: hardwareId,
                phone_number: devicePhone,
                is_primary: primary,
                replace_device_id: replaceId
            }, config);

            const { device_pair_key } = pairResponse.data;
            store.setApiKey(device_pair_key);
            store.setDeviceUuid(hardwareId);
            store.toggleService(true);

            // Auto-start service immediately after pairing
            await BackgroundServiceControl.start();
            wsService.connect();
            smsQueue.start();

            DatabaseService.addLog(null, 'DEVICE_PAIRED', `Device paired successfully as primary=${primary}. HardwareID: ${hardwareId}`);

            Alert.alert('تم بنجاح', 'تم إعداد وربط هاتفك آلياً بنجاح');
            navigation.replace('Dashboard');
        } catch (error: any) {
            console.error('[PAIR] Pairing execution failed:', error.response?.data || error.message);
            // Clear auth as we shouldn't be "logged in" without a paired device in this mode
            store.setAuthenticated(false);
            store.setToken(null);

            const message = getErrorMessage(error, 'فشل ربط الجهاز بالقاعدة. يرجى المحاولة لاحقاً.');
            DatabaseService.addLog(null, 'PAIR_FAILED', `Device pairing failed: ${message}`);
            Alert.alert('خطأ في الربط', message);
        } finally {
            setLoading(false);
            setShowSetupModal(false);
        }
    };

    return (
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.container}>
            <StatusBar barStyle="light-content" />
            <ScrollView contentContainerStyle={styles.scroll}>
                <View style={styles.logoContainer}>
                    <Text style={styles.logo}>الهدهد</Text>
                    <Text style={styles.tagline}>بوابة الرسائل الذكية</Text>
                </View>

                <View style={styles.form}>
                    <View style={styles.tabContainer}>
                        <TouchableOpacity
                            style={[styles.tab, mode === 'login' && styles.activeTab]}
                            onPress={() => setMode('login')}
                        >
                            <Text style={[styles.tabText, mode === 'login' && styles.activeTabText]}>دخول</Text>
                        </TouchableOpacity>
                        <TouchableOpacity
                            style={[styles.tab, mode === 'register' && styles.activeTab]}
                            onPress={() => setMode('register')}
                        >
                            <Text style={[styles.tabText, mode === 'register' && styles.activeTabText]}>تسجيل جديد</Text>
                        </TouchableOpacity>
                    </View>

                    <Text style={styles.label}>رابط الخادم (API URL)</Text>
                    <TextInput
                        style={styles.input}
                        value={apiUrl}
                        onChangeText={setApiUrl}
                        placeholder="http://192.168.1.10:8082"
                        placeholderTextColor="#444"
                        autoCapitalize="none"
                    />

                    <Text style={styles.label}>البريد الإلكتروني</Text>
                    <TextInput
                        style={styles.input}
                        value={email}
                        onChangeText={setEmail}
                        placeholder="example@mail.com"
                        placeholderTextColor="#444"
                        keyboardType="email-address"
                        autoCapitalize="none"
                    />

                    {mode === 'register' && (
                        <>
                            <View style={styles.row}>
                                <View style={{ flex: 2 }}>
                                    <Text style={styles.label}>رقم الهاتف</Text>
                                    <TextInput
                                        style={styles.input}
                                        value={phone}
                                        onChangeText={setPhone}
                                        placeholder="50xxxxxxx"
                                        placeholderTextColor="#444"
                                        keyboardType="phone-pad"
                                    />
                                </View>
                                <View style={{ flex: 1, marginLeft: 10 }}>
                                    <Text style={styles.label}>الكود</Text>
                                    <TextInput
                                        style={styles.input}
                                        value={countryCode}
                                        onChangeText={setCountryCode}
                                        placeholder="+966"
                                        placeholderTextColor="#444"
                                    />
                                </View>
                            </View>
                        </>
                    )}

                    <Text style={styles.label}>كلمة المرور</Text>
                    <TextInput
                        style={styles.input}
                        value={password}
                        onChangeText={setPassword}
                        placeholder="********"
                        placeholderTextColor="#444"
                        secureTextEntry
                    />

                    {mode === 'register' && (
                        <>
                            <Text style={styles.label}>تأكيد كلمة المرور</Text>
                            <TextInput
                                style={styles.input}
                                value={confirmPassword}
                                onChangeText={setConfirmPassword}
                                placeholder="********"
                                placeholderTextColor="#444"
                                secureTextEntry
                            />
                        </>
                    )}

                    <TouchableOpacity style={styles.button} onPress={handleSubmit} disabled={loading}>
                        {loading ? <ActivityIndicator color="#FFF" /> : (
                            <Text style={styles.buttonText}>{mode === 'login' ? 'دخول' : 'إنشاء حساب جديد'}</Text>
                        )}
                    </TouchableOpacity>
                </View>
            </ScrollView>

            {/* Smart Onboarding Modal */}
            <Modal visible={showSetupModal} transparent animationType="slide">
                <View style={styles.modalOverlay}>
                    <View style={styles.modalContent}>
                        <Text style={styles.modalTitle}>هاتف جديد مكتشف</Text>
                        <Text style={styles.modalSub}>هذا الهاتف غير مسجل في حسابك. ماذا تريد أن تفعل؟</Text>

                        <TouchableOpacity
                            style={[styles.optionCard, setupType === 'new' && styles.optionCardActive]}
                            onPress={() => setSetupType('new')}
                        >
                            <Text style={styles.optionTitle}>إضافة كجهاز جديد</Text>
                            <Text style={styles.optionDesc}>سيتم إضافة هذا الهاتف كبوابة إرسال إضافية</Text>
                        </TouchableOpacity>

                        <TouchableOpacity
                            style={[styles.optionCard, setupType === 'replace' && styles.optionCardActive]}
                            onPress={() => setSetupType('replace')}
                        >
                            <Text style={styles.optionTitle}>استبدال جهاز قديم</Text>
                            <Text style={styles.optionDesc}>اختر جهازاً لتعطيله واستخدام هذا الهاتف بدلاً منه</Text>
                        </TouchableOpacity>

                        {setupType === 'new' && (
                            <View style={styles.priorityRow}>
                                <Text style={styles.label}>جعله الجهاز الرئيسي (الأكثر أولوية)؟</Text>
                                <Switch value={isPrimary} onValueChange={setIsPrimary} />
                            </View>
                        )}

                        {setupType === 'replace' && (
                            <View style={{ marginBottom: 20 }}>
                                <Text style={styles.label}>اختر الجهاز الذي تريد استبداله:</Text>
                                {existingDevices.map((d: any) => (
                                    <TouchableOpacity
                                        key={d.hardware_id}
                                        style={[styles.deviceOption, selectedDeviceToReplace === d.hardware_id && styles.deviceOptionActive]}
                                        onPress={() => setSelectedDeviceToReplace(d.hardware_id)}
                                    >
                                        <Text style={{ color: '#FFF', textAlign: 'right' }}>{d.device_name} ({d.phone_number || 'بدون رقم'})</Text>
                                    </TouchableOpacity>
                                ))}
                            </View>
                        )}

                        <TouchableOpacity
                            style={styles.confirmButton}
                            onPress={() => {
                                const devPhone = phone || tempAuthData?.tenant?.phone || '05xxxxxxxx';
                                autoPairDevice(
                                    store.token!,
                                    store.deviceUuid!,
                                    "هاتف جديد",
                                    isPrimary,
                                    devPhone,
                                    selectedDeviceToReplace || undefined
                                );
                            }}
                            disabled={!setupType || (setupType === 'replace' && !selectedDeviceToReplace)}
                        >
                            <Text style={styles.confirmButtonText}>متابعة</Text>
                        </TouchableOpacity>
                    </View>
                </View>
            </Modal>
        </KeyboardAvoidingView>
    );
};

const { width: screenWidth } = Dimensions.get('window');

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212'
    },
    scroll: {
        flexGrow: 1,
        justifyContent: 'center',
        paddingHorizontal: screenWidth > 400 ? 40 : 25,
        paddingVertical: 40
    },
    logoContainer: {
        alignItems: 'center',
        marginBottom: screenWidth > 400 ? 50 : 30
    },
    logo: {
        fontSize: screenWidth > 400 ? 64 : 52,
        fontWeight: 'bold',
        color: '#FFB300',
        textShadowColor: 'rgba(255, 179, 0, 0.3)',
        textShadowOffset: { width: 0, height: 4 },
        textShadowRadius: 10
    },
    tagline: {
        color: '#666',
        fontSize: 16,
        marginTop: 5,
        letterSpacing: 1
    },
    form: {
        backgroundColor: '#1E1E1E',
        padding: 24,
        borderRadius: 24,
        borderWidth: 1,
        borderColor: '#2A2A2A',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 10 },
        shadowOpacity: 0.3,
        shadowRadius: 20,
        elevation: 10
    },
    tabContainer: {
        flexDirection: 'row-reverse',
        marginBottom: 25,
        backgroundColor: '#121212',
        borderRadius: 14,
        padding: 4
    },
    tab: {
        flex: 1,
        paddingVertical: 14,
        alignItems: 'center',
        borderRadius: 10
    },
    activeTab: {
        backgroundColor: '#FFB300'
    },
    tabText: {
        color: '#666',
        fontWeight: 'bold',
        fontSize: 14
    },
    activeTabText: {
        color: '#000'
    },
    label: {
        color: '#888',
        marginBottom: 8,
        textAlign: 'right',
        fontSize: 13,
        fontWeight: '600'
    },
    input: {
        backgroundColor: '#1A1A1A',
        color: '#FFF',
        borderRadius: 16,
        padding: 18,
        marginBottom: 20,
        textAlign: 'right',
        borderWidth: 1,
        borderColor: '#333',
        fontSize: 15,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.2,
        shadowRadius: 4,
    },
    row: {
        flexDirection: 'row-reverse',
        gap: 12,
    },
    button: {
        backgroundColor: '#FFB300',
        padding: 20,
        borderRadius: 20,
        alignItems: 'center',
        marginTop: 15,
        shadowColor: '#FFB300',
        shadowOffset: { width: 0, height: 8 },
        shadowOpacity: 0.4,
        shadowRadius: 12,
        elevation: 10,
    },
    buttonText: {
        color: '#000',
        fontSize: 18,
        fontWeight: '900'
    },
    modalOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.85)',
        justifyContent: 'flex-end'
    },
    modalContent: {
        backgroundColor: '#1E1E1E',
        borderTopLeftRadius: 32,
        borderTopRightRadius: 32,
        padding: 24,
        paddingBottom: 40
    },
    modalTitle: {
        color: '#FFB300',
        fontSize: 22,
        fontWeight: 'bold',
        textAlign: 'center',
        marginBottom: 12
    },
    modalSub: {
        color: '#999',
        textAlign: 'center',
        marginBottom: 25,
        fontSize: 14,
        lineHeight: 20
    },
    optionCard: {
        backgroundColor: '#252525',
        padding: 20,
        borderRadius: 18,
        marginBottom: 16,
        borderWidth: 1,
        borderColor: '#333'
    },
    optionCardActive: {
        borderColor: '#FFB300',
        backgroundColor: '#302610'
    },
    optionTitle: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: 'bold',
        textAlign: 'right'
    },
    optionDesc: {
        color: '#888',
        textAlign: 'right',
        marginTop: 6,
        fontSize: 13
    },
    deviceOption: {
        backgroundColor: '#252525',
        padding: 15,
        borderRadius: 12,
        marginBottom: 10,
        borderWidth: 1,
        borderColor: '#333'
    },
    deviceOptionActive: {
        borderColor: '#FFB300',
        backgroundColor: '#302610'
    },
    priorityRow: {
        flexDirection: 'row-reverse',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginVertical: 20,
        backgroundColor: '#181818',
        padding: 15,
        borderRadius: 12
    },
    confirmButton: {
        backgroundColor: '#4CAF50',
        padding: 20,
        borderRadius: 18,
        alignItems: 'center',
        shadowColor: '#4CAF50',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 6,
        elevation: 6
    },
    confirmButtonText: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: 'bold'
    }
});
