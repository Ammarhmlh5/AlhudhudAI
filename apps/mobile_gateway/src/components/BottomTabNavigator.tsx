import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Dimensions, Platform } from 'react-native';
import { useAppStore } from '../services/StorageService';
import { DatabaseService } from '../services/DatabaseService';
import { BackgroundServiceControl } from '../services/BackgroundServiceControl';

const { width } = Dimensions.get('window');

interface BottomTabProps {
    navigation: any;
    activeTab: 'home' | 'services' | 'reports' | 'settings';
}

export const BottomTabNavigator = ({ navigation, activeTab }: BottomTabProps) => {
    const store = useAppStore();

    const tabs = [
        { id: 'settings', label: 'الإعدادات', icon: '⚙️', route: 'Settings' },
        { id: 'reports', label: 'التقارير', icon: '📊', route: 'Logs' },
        { id: 'center', label: '', icon: '', route: 'toggle' }, // Placeholder for the big center button
        { id: 'services', label: 'الخدمات', icon: '🛠️', route: 'Services' },
        { id: 'home', label: 'الرئيسية', icon: '🏠', route: 'Dashboard' },
    ];

    return (
        <View style={styles.container}>
            <View style={styles.tabBar}>
                {tabs.map((tab) => {
                    if (tab.id === 'center') {
                        return (
                            <TouchableOpacity
                                key={tab.id}
                                style={styles.centerButtonContainer}
                                onPress={async () => {
                                    const nextStatus = !store.isServiceRunning;
                                    if (nextStatus) {
                                        const started = await BackgroundServiceControl.start();
                                        if (started) {
                                            const { wsService } = require('../services/WebSocketService');
                                            wsService.connect();
                                            store.toggleService(true);
                                            DatabaseService.addLog(null, 'SERVICE_STARTED', 'Gateway background service started.');
                                        }
                                    } else {
                                        const stopped = await BackgroundServiceControl.stop();
                                        if (stopped) {
                                            const { wsService } = require('../services/WebSocketService');
                                            wsService.disconnect();
                                            store.toggleService(false);
                                            DatabaseService.addLog(null, 'SERVICE_STOPPED', 'Gateway background service stopped.');
                                        }
                                    }
                                }}
                                activeOpacity={0.8}
                            >
                                <View style={[styles.centerButton, { backgroundColor: store.isServiceRunning ? '#F44336' : '#4CAF50' }]}>
                                    <Text style={styles.centerButtonIcon}>{store.isServiceRunning ? '⏹️' : '▶️'}</Text>
                                </View>
                                <Text style={styles.centerLabel}>{store.isServiceRunning ? 'إيقاف' : 'تشغيل'}</Text>
                            </TouchableOpacity>
                        );
                    }

                    const isActive = activeTab === tab.id;

                    return (
                        <TouchableOpacity
                            key={tab.id}
                            style={styles.tabItem}
                            onPress={() => navigation.navigate(tab.route)}
                        >
                            <Text style={[styles.tabIcon, isActive && styles.activeTabIcon]}>{tab.icon}</Text>
                            <Text style={[styles.tabLabel, isActive && styles.activeTabLabel]}>{tab.label}</Text>
                            {isActive && <View style={styles.activeDot} />}
                        </TouchableOpacity>
                    );
                })}
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        position: 'absolute',
        bottom: 0,
        width: '100%',
        height: Platform.OS === 'ios' ? 90 : 75,
        backgroundColor: '#1A1A1A',
        borderTopWidth: 1,
        borderTopColor: '#333',
        paddingBottom: Platform.OS === 'ios' ? 20 : 0,
    },
    tabBar: {
        flexDirection: 'row',
        justifyContent: 'space-around',
        alignItems: 'center',
        height: '100%',
    },
    tabItem: {
        alignItems: 'center',
        justifyContent: 'center',
        width: width / 5,
    },
    tabIcon: {
        fontSize: 20,
        marginBottom: 4,
        opacity: 0.5,
    },
    activeTabIcon: {
        opacity: 1,
    },
    tabLabel: {
        color: '#888',
        fontSize: 10,
        fontWeight: '600',
    },
    activeTabLabel: {
        color: '#FFB300',
    },
    activeDot: {
        width: 4,
        height: 4,
        borderRadius: 2,
        backgroundColor: '#FFB300',
        marginTop: 4,
    },
    centerButtonContainer: {
        top: -25,
        alignItems: 'center',
        justifyContent: 'center',
        width: width / 5,
    },
    centerButton: {
        width: 60,
        height: 60,
        borderRadius: 30,
        justifyContent: 'center',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.5,
        shadowRadius: 5,
        elevation: 8,
        borderWidth: 3,
        borderColor: '#1A1A1A',
    },
    centerButtonIcon: {
        fontSize: 24,
        color: '#FFF',
    },
    centerLabel: {
        color: '#FFF',
        fontSize: 10,
        fontWeight: 'bold',
        marginTop: 4,
    },
});
