import React, { useRef, useState, useEffect } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ScrollView,
    StatusBar,
    Dimensions,
    Animated,
    NativeScrollEvent,
    NativeSyntheticEvent,
    Alert
} from 'react-native';
import { useAppStore } from '../services/StorageService';
import { BackgroundServiceControl } from '../services/BackgroundServiceControl';
import { smsQueue } from '../services/SmsQueueManager';
import { DatabaseService } from '../services/DatabaseService';
import { BottomTabNavigator } from '../components/BottomTabNavigator';
import { notificationService } from '../services/NotificationService';
import axios from 'axios';

const { width: screenWidth } = Dimensions.get('window');

/**
 * DashboardScreen.tsx
 * 
 * Redesigned monitoring screen with "E-Wallet" aesthetics.
 */

export const DashboardScreen = ({ navigation }: any) => {
    const store = useAppStore();
    const [stats, setStats] = useState({ pending: 0, sent: 0, failed: 0 });
    const [logs, setLogs] = useState<any[]>([]);
    const [sims, setSims] = useState<any[]>([]);
    const [activeCardIndex, setActiveCardIndex] = useState(0);
    const scrollX = useRef(new Animated.Value(0)).current;

    const [finance, setFinance] = useState({ cash_balance: 0, quota_balance: 0, message_limit: 0, message_used: 0 });

    React.useEffect(() => {
        const fetchData = async () => {
            setStats(DatabaseService.getStats());
            setLogs(DatabaseService.getLogs(5)); // Show last 5 logs
            const simInfo = await BackgroundServiceControl.getSimInfo();
            setSims(simInfo);

            // جلب عدد الإشعارات غير المقروءة من قاعدة البيانات المحلية
            try {
                // تهيئة خدمة الإشعارات إذا لم تكن موجودة
                notificationService.init();
                
                // جلب عدد الإشعارات غير المقروءة
                const unreadCount = notificationService.getTotalUnreadCount();
                store.setUnreadNotifications(unreadCount);
            } catch (err) {
                console.error("Failed to fetch notifications:", err);
            }

            try {
                const config = { headers: {} as Record<string, string> };
                if (store.token) config.headers['Authorization'] = `Bearer ${store.token}`;
                else if (store.apiKey) config.headers['X-API-Key'] = store.apiKey;
                
                const res = await axios.get(`${store.apiUrl}/api/v1/finance/balance`, config);
                setFinance(res.data);
            } catch (err) {
                console.error("Failed to fetch finance stats:", err);
            }
        };

        fetchData();
        const interval = setInterval(fetchData, 5000);
        return () => clearInterval(interval);
    }, []);

    const handleScroll = (event: NativeSyntheticEvent<NativeScrollEvent>) => {
        const contentOffsetX = event.nativeEvent.contentOffset.x;
        const index = Math.round(contentOffsetX / (screenWidth - 40));
        setActiveCardIndex(index);
    };

    const renderCarouselCard = (type: 'messages' | 'cash') => {
        if (type === 'messages') {
            return (
                <View style={[styles.carouselCard, styles.messagesCard]}>
                    <View style={styles.cardHeader}>
                        <Text style={styles.cardIcon}>📦</Text>
                        <Text style={styles.cardTitle}>باقة الرسائل</Text>
                    </View>
                    <View style={styles.cardBody}>
                        <View style={styles.balanceRow}>
                            <Text style={styles.balanceValue}>{finance.quota_balance || 0}</Text>
                            <Text style={styles.balanceLabel}>رسالة متبقية</Text>
                        </View>
                        <View style={styles.progressContainer}>
                            <View style={[styles.progressBar, { width: `${finance.message_limit > 0 ? (finance.message_used / finance.message_limit) * 100 : 0}%` }]} />
                        </View>
                        <Text style={styles.cardFooterText}>تم إرسال {finance.message_used || 0} رسالة من إجمالي {finance.message_limit || 0}</Text>
                    </View>
                </View>
            );
        } else {
            return (
                <View style={[styles.carouselCard, styles.cashCard]}>
                    <View style={styles.cardHeader}>
                        <Text style={styles.cardIcon}>💰</Text>
                        <Text style={styles.cardTitle}>رصيد الحساب</Text>
                    </View>
                    <View style={styles.cardBody}>
                        <View style={styles.balanceRow}>
                            <Text style={styles.balanceValue}>{finance.cash_balance?.toFixed(2) || '0.00'}</Text>
                            <Text style={styles.balanceLabel}>ر.س</Text>
                        </View>
                        <TouchableOpacity style={styles.rechargeBtn}>
                            <Text style={styles.rechargeBtnText}>شحن سريع</Text>
                        </TouchableOpacity>
                        <Text style={styles.cardFooterText}>آخر عملية شحن: منذ يومين</Text>
                    </View>
                </View>
            );
        }
    };

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />

            {/* Top Toolbar */}
            <View style={styles.toolbar}>
                <TouchableOpacity onPress={() => navigation.navigate('Support')} style={styles.toolbarBtn}>
                    <Text style={styles.toolbarIcon}>🤖</Text>
                </TouchableOpacity>
                <Text style={styles.appLogo}>الهدهد</Text>
                <TouchableOpacity style={styles.toolbarBtn} onPress={() => navigation.navigate('Notifications')}>
                    {store.unreadNotifications > 0 && (
                        <View style={styles.badge}>
                            <Text style={styles.badgeText}>
                                {store.unreadNotifications > 99 ? '99+' : store.unreadNotifications}
                            </Text>
                        </View>
                    )}
                    <Text style={styles.toolbarIcon}>🔔</Text>
                </TouchableOpacity>
            </View>

            <ScrollView contentContainerStyle={styles.mainScroll}>

                {/* Floating Carousel */}
                <View style={styles.carouselContainer}>
                    <ScrollView
                        horizontal
                        pagingEnabled
                        showsHorizontalScrollIndicator={false}
                        onScroll={Animated.event(
                            [{ nativeEvent: { contentOffset: { x: scrollX } } }],
                            { useNativeDriver: false, listener: handleScroll }
                        )}
                        scrollEventThrottle={16}
                        contentContainerStyle={styles.carouselScroll}
                    >
                        {renderCarouselCard('messages')}
                        {renderCarouselCard('cash')}
                    </ScrollView>

                    {/* Pagination Indicators */}
                    <View style={styles.pagination}>
                        {[0, 1].map((i) => (
                            <View
                                key={i}
                                style={[styles.dot, activeCardIndex === i ? styles.activeDot : null]}
                            />
                        ))}
                    </View>
                </View>

                {/* Connectivity Status Bar */}
                <TouchableOpacity 
                    style={[styles.connectivityBar, !store.isConnected && { borderColor: '#F4433655' }]}
                    onPress={async () => {
                        if (store.continuousWork) {
                            Alert.alert(
                                'تنبيه',
                                'وضع العمل المستمر نشط. لإيقاف أو تشغيل الاتصال يدوياً، يرجى تعطيل وضع العمل المستمر من الإعدادات أولاً.'
                            );
                            return;
                        }

                        if (store.isConnected) {
                            // Disconnect
                            const { wsService } = require('../services/WebSocketService');
                            wsService.disconnect();
                            if (store.isServiceRunning) {
                                await BackgroundServiceControl.stop();
                                store.toggleService(false);
                            }
                            Alert.alert('تم قطع الاتصال', 'تم قطع الاتصال بالسيرفر وإيقاف البوابة يدوياً.');
                        } else {
                            // Connect
                            const { wsService } = require('../services/WebSocketService');
                            wsService.connect();
                            if (!store.isServiceRunning) {
                                await BackgroundServiceControl.start();
                                store.toggleService(true);
                            }
                            Alert.alert('جاري الاتصال', 'تم بدء تشغيل البوابة وتوصيلها بالسيرفر.');
                        }
                    }}
                >
                    <View style={[styles.statusIndicator, { backgroundColor: store.isConnected ? '#4CAF50' : '#F44336' }]} />
                    <Text style={[styles.connectivityText, !store.isConnected && { color: '#F44336' }]}>
                        {store.isConnected ? 'متصل بالسيرفر (اضغط للفصل)' : 'غير متصل (اضغط للاتصال)'}
                    </Text>
                    <Text style={styles.deviceUuid}>ID: {store.deviceUuid?.substring(0, 8)}</Text>
                </TouchableOpacity>

                {/* SIM Cards Info */}
                <View style={styles.sectionHeader}>
                    <Text style={styles.sectionTitle}>الشرائح النشطة</Text>
                </View>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.simScroll}>
                    {sims.length > 0 ? sims.map((sim, index) => (
                        <View key={index} style={styles.simChip}>
                            <Text style={styles.simIcon}>📶</Text>
                            <View>
                                <Text style={styles.simName}>{sim.carrierName || 'SIM ' + (index + 1)}</Text>
                                <Text style={styles.simNumber}>{sim.number || 'Slot ' + (sim.slotIndex + 1)}</Text>
                            </View>
                        </View>
                    )) : (
                        <Text style={styles.emptySims}>لا توجد شرائح مكتشفة</Text>
                    )}
                </ScrollView>

                {/* Recent Activity List */}
                <View style={styles.sectionHeader}>
                    <TouchableOpacity onPress={() => navigation.navigate('Logs')}>
                        <Text style={styles.viewAllText}>عرض الكل</Text>
                    </TouchableOpacity>
                    <Text style={styles.sectionTitle}>آخر النشاطات</Text>
                </View>

                <View style={styles.activityContainer}>
                    {logs.map((log) => {
                        const isError = log.event.includes('ERROR') || log.event.includes('FAILED');
                        return (
                            <View key={log.id} style={styles.activityItem}>
                                <View style={[styles.activityIcon, { backgroundColor: isError ? '#F4433622' : '#4CAF5022' }]}>
                                    <Text style={{ fontSize: 16 }}>{isError ? '⚠️' : '✉️'}</Text>
                                </View>
                                <View style={styles.activityDetails}>
                                    <Text style={styles.activityEvent}>{log.event}</Text>
                                    <Text style={styles.activityTime}>{new Date(log.timestamp).toLocaleTimeString()}</Text>
                                </View>
                                <Text style={[styles.activityStatus, { color: isError ? '#F44336' : '#4CAF50' }]}>
                                    {isError ? 'فشل' : 'نجاح'}
                                </Text>
                            </View>
                        );
                    })}
                    {logs.length === 0 && <Text style={styles.emptyText}>لا يوجد نشاط مؤخراً</Text>}
                </View>

            </ScrollView>

            <BottomTabNavigator navigation={navigation} activeTab="home" />
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    toolbar: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 20,
        backgroundColor: '#1A1A1A',
    },
    toolbarBtn: {
        width: 44,
        height: 44,
        borderRadius: 12,
        backgroundColor: '#252525',
        justifyContent: 'center',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#333',
    },
    toolbarIcon: {
        fontSize: 20,
    },
    badge: {
        position: 'absolute',
        top: 6,
        right: 6,
        minWidth: 18,
        height: 18,
        borderRadius: 9,
        backgroundColor: '#F44336',
        zIndex: 1,
        borderWidth: 1.5,
        borderColor: '#252525',
        justifyContent: 'center',
        alignItems: 'center',
        paddingHorizontal: 4,
    },
    badgeText: {
        color: '#FFF',
        fontSize: 10,
        fontWeight: 'bold',
    },
    appLogo: {
        fontSize: 22,
        fontWeight: 'bold',
        color: '#FFB300',
    },
    mainScroll: {
        paddingBottom: 120,
    },
    carouselContainer: {
        marginTop: 20,
        height: 220,
    },
    carouselScroll: {
        paddingHorizontal: 20,
    },
    carouselCard: {
        width: screenWidth - 40,
        height: 190,
        borderRadius: 28,
        padding: 24,
        marginRight: 0,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 10 },
        shadowOpacity: 0.4,
        shadowRadius: 15,
        elevation: 10,
    },
    messagesCard: {
        backgroundColor: '#1E1E1E',
        borderWidth: 1,
        borderColor: '#FFB30033',
    },
    cashCard: {
        backgroundColor: '#1E1E1E',
        borderWidth: 1,
        borderColor: '#2196F333',
    },
    cardHeader: {
        flexDirection: 'row-reverse',
        alignItems: 'center',
        marginBottom: 20,
    },
    cardIcon: {
        fontSize: 24,
        marginLeft: 12,
    },
    cardTitle: {
        color: '#AAA',
        fontSize: 16,
        fontWeight: 'bold',
    },
    cardBody: {
        alignItems: 'center',
    },
    balanceRow: {
        flexDirection: 'row-reverse',
        alignItems: 'baseline',
        marginBottom: 10,
    },
    balanceValue: {
        color: '#FFF',
        fontSize: 36,
        fontWeight: 'bold',
    },
    balanceLabel: {
        color: '#888',
        fontSize: 14,
        marginRight: 8,
    },
    progressContainer: {
        width: '100%',
        height: 8,
        backgroundColor: '#252525',
        borderRadius: 4,
        marginBottom: 12,
        overflow: 'hidden',
    },
    progressBar: {
        height: '100%',
        backgroundColor: '#FFB300',
        borderRadius: 4,
    },
    cardFooterText: {
        color: '#666',
        fontSize: 11,
    },
    rechargeBtn: {
        backgroundColor: '#2196F322',
        paddingHorizontal: 25,
        paddingVertical: 10,
        borderRadius: 20,
        borderWidth: 1,
        borderColor: '#2196F3',
        marginBottom: 12,
    },
    rechargeBtnText: {
        color: '#2196F3',
        fontWeight: 'bold',
        fontSize: 14,
    },
    pagination: {
        flexDirection: 'row',
        justifyContent: 'center',
        marginTop: 10,
    },
    dot: {
        width: 6,
        height: 6,
        borderRadius: 3,
        backgroundColor: '#333',
        marginHorizontal: 4,
    },
    activeDot: {
        width: 20,
        backgroundColor: '#FFB300',
    },
    connectivityBar: {
        flexDirection: 'row-reverse',
        alignItems: 'center',
        backgroundColor: '#1A1A1A',
        marginHorizontal: 20,
        marginTop: 10,
        padding: 12,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: '#2A2A2A',
    },
    statusIndicator: {
        width: 10,
        height: 10,
        borderRadius: 5,
        marginLeft: 10,
    },
    connectivityText: {
        color: '#EEE',
        fontSize: 12,
        flex: 1,
        textAlign: 'right',
    },
    deviceUuid: {
        color: '#555',
        fontSize: 10,
    },
    sectionHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingHorizontal: 25,
        marginTop: 30,
        marginBottom: 15,
    },
    sectionTitle: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: 'bold',
    },
    viewAllText: {
        color: '#FFB300',
        fontSize: 13,
    },
    simScroll: {
        paddingLeft: 20,
    },
    simChip: {
        flexDirection: 'row-reverse',
        alignItems: 'center',
        backgroundColor: '#1E1E1E',
        padding: 15,
        borderRadius: 18,
        marginRight: 15,
        borderWidth: 1,
        borderColor: '#2A2A2A',
    },
    simIcon: {
        fontSize: 20,
        marginLeft: 12,
    },
    simName: {
        color: '#FFF',
        fontSize: 14,
        fontWeight: 'bold',
        textAlign: 'right',
    },
    simNumber: {
        color: '#666',
        fontSize: 11,
        textAlign: 'right',
    },
    emptySims: {
        color: '#444',
        paddingVertical: 10,
    },
    activityContainer: {
        marginHorizontal: 20,
        backgroundColor: '#1E1E1E',
        borderRadius: 24,
        padding: 10,
        borderWidth: 1,
        borderColor: '#2A2A2A',
    },
    activityItem: {
        flexDirection: 'row-reverse',
        alignItems: 'center',
        padding: 15,
        borderBottomWidth: 1,
        borderBottomColor: '#252525',
    },
    activityIcon: {
        width: 44,
        height: 44,
        borderRadius: 14,
        justifyContent: 'center',
        alignItems: 'center',
        marginLeft: 15,
    },
    activityDetails: {
        flex: 1,
    },
    activityEvent: {
        color: '#EEE',
        fontSize: 14,
        fontWeight: '600',
        textAlign: 'right',
    },
    activityTime: {
        color: '#555',
        fontSize: 11,
        textAlign: 'right',
        marginTop: 2,
    },
    activityStatus: {
        fontSize: 12,
        fontWeight: 'bold',
    },
    emptyText: {
        color: '#444',
        textAlign: 'center',
        paddingVertical: 20,
    },
});
