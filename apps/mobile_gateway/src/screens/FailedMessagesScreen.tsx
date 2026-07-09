import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, Alert, StatusBar } from 'react-native';
import { DatabaseService } from '../services/DatabaseService';
import { smsQueue } from '../services/SmsQueueManager';

/**
 * FailedMessagesScreen.tsx
 * 
 * Shows messages that failed to send with detailed reasons and a retry button.
 */

export const FailedMessagesScreen = ({ navigation }: any) => {
    const [failedMessages, setFailedMessages] = useState<any[]>([]);

    const fetchFailed = () => {
        const results = DatabaseService.getFailedMessages();
        setFailedMessages(results);
    };

    useEffect(() => {
        fetchFailed();
    }, []);

    const handleRetry = async (msg: any) => {
        // Reset status to PENDING so queue picks it up
        DatabaseService.updateMessageStatus(msg.id, 'PENDING');
        smsQueue.start(); // Ensure queue is running
        Alert.alert('تمت الإعادة', 'تمت إضافة الرسالة لزمام الانتظار مرة أخرى.');
        fetchFailed();
    };

    const renderItem = ({ item }: { item: any }) => (
        <View style={styles.card}>
            <View style={styles.row}>
                <Text style={styles.phone}>{item.phone_number}</Text>
                <Text style={styles.date}>{new Date(item.created_at).toLocaleTimeString()}</Text>
            </View>
            <Text style={styles.content} numberOfLines={2}>{item.content}</Text>
            <Text style={styles.reason}>سبب الفشل: {item.failure_reason}</Text>

            <TouchableOpacity style={styles.retryButton} onPress={() => handleRetry(item)}>
                <Text style={styles.retryText}>إعادة الإرسال 🔄</Text>
            </TouchableOpacity>
        </View>
    );

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />
            <View style={[styles.header, { flexDirection: 'row-reverse' }]}>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.backText}>رجوع →</Text>
                </TouchableOpacity>
                <Text style={styles.title}>رسائل فاشلة</Text>
                <TouchableOpacity onPress={fetchFailed}>
                    <Text style={styles.refreshText}>🔄</Text>
                </TouchableOpacity>
            </View>

            <FlatList
                data={failedMessages}
                keyExtractor={(item) => item.id}
                renderItem={renderItem}
                contentContainerStyle={styles.list}
                ListEmptyComponent={<Text style={styles.empty}>لا توجد رسائل فاشلة حالياً.</Text>}
            />
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: '#121212' },
    header: {
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 20,
        backgroundColor: '#1E1E1E',
        alignItems: 'center',
        justifyContent: 'space-between',
    },
    title: { color: '#FFB300', fontSize: 18, fontWeight: 'bold' },
    backText: { color: '#2196F3', fontSize: 16 },
    refreshText: { fontSize: 20 },
    list: { padding: 15 },
    card: {
        backgroundColor: '#1E1E1E',
        borderRadius: 12,
        padding: 15,
        marginBottom: 15,
        borderLeftWidth: 4,
        borderLeftColor: '#F44336'
    },
    row: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 5 },
    phone: { color: '#FFF', fontWeight: 'bold' },
    date: { color: '#666', fontSize: 12 },
    content: { color: '#CCC', fontSize: 14, marginBottom: 10 },
    reason: { color: '#F44336', fontSize: 12, marginBottom: 15, backgroundColor: 'rgba(244, 67, 54, 0.1)', padding: 5, borderRadius: 4 },
    retryButton: {
        backgroundColor: '#4CAF50',
        paddingVertical: 10,
        borderRadius: 8,
        alignItems: 'center'
    },
    retryText: { color: '#FFF', fontWeight: 'bold' },
    empty: { color: '#444', textAlign: 'center', marginTop: 50 }
});
