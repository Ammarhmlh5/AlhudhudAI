import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, StatusBar, Platform } from 'react-native';
import { DatabaseService } from '../services/DatabaseService';
import { BottomTabNavigator } from '../components/BottomTabNavigator';

/**
 * LogsScreen.tsx
 * 
 * Diagnostic screen to view internal SQLite logs.
 */

export const LogsScreen = ({ navigation }: any) => {
    const [logs, setLogs] = useState<any[]>([]);

    const fetchLogs = () => {
        const data = DatabaseService.getLogs(100);
        setLogs(data);
    };

    useEffect(() => {
        fetchLogs();
    }, []);

    const renderItem = ({ item }: any) => {
        const isError = item.event.includes('ERROR') || item.event.includes('FAILED') || item.event.includes('REJECTED');

        return (
            <View style={styles.logItem}>
                <View style={styles.logHeader}>
                    <Text style={[styles.logEvent, { color: isError ? '#F44336' : '#4CAF50' }]}>
                        {item.event}
                    </Text>
                    <Text style={styles.logTime}>{new Date(item.timestamp).toLocaleTimeString()}</Text>
                </View>
                {item.message_id && <Text style={styles.logMsgId}>ID: {item.message_id}</Text>}
                <Text style={styles.logDetails}>{item.details}</Text>
            </View>
        );
    };

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />
            <View style={[styles.header, { flexDirection: 'row-reverse' }]}>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.backText}>رجوع →</Text>
                </TouchableOpacity>
                <Text style={styles.title}>سجلات النظام</Text>
                <TouchableOpacity onPress={fetchLogs}>
                    <Text style={styles.refreshText}>🔄</Text>
                </TouchableOpacity>
            </View>

            <FlatList
                data={logs}
                keyExtractor={(item) => item.id.toString()}
                renderItem={renderItem}
                contentContainerStyle={[styles.list, { paddingBottom: 100 }]}
                ListEmptyComponent={<Text style={styles.empty}>لا توجد سجلات حالياً.</Text>}
            />

            <BottomTabNavigator navigation={navigation} activeTab="reports" />
        </View>
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
    refreshText: {
        fontSize: 20,
    },
    title: {
        fontSize: 18,
        fontWeight: 'bold',
        color: '#FFB300',
    },
    list: {
        padding: 10,
    },
    logItem: {
        backgroundColor: '#1E1E1E',
        padding: 15,
        borderRadius: 8,
        marginBottom: 10,
        borderLeftWidth: 4,
        borderLeftColor: '#333',
    },
    logHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        marginBottom: 5,
    },
    logEvent: {
        fontWeight: 'bold',
        fontSize: 12,
    },
    logTime: {
        color: '#666',
        fontSize: 10,
    },
    logMsgId: {
        color: '#888',
        fontSize: 10,
        fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
        marginBottom: 5,
    },
    logDetails: {
        color: '#CCC',
        fontSize: 13,
        lineHeight: 18,
    },
    empty: {
        color: '#444',
        textAlign: 'center',
        marginTop: 50,
    },
});
