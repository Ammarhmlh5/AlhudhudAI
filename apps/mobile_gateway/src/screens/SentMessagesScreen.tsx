import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, StatusBar } from 'react-native';
import { DatabaseService } from '../services/DatabaseService';

/**
 * SentMessagesScreen.tsx
 * 
 * Shows the queue of messages arriving from the server and their sending statuses.
 */

export const SentMessagesScreen = ({ navigation }: any) => {
    const [messages, setMessages] = useState<any[]>([]);

    const fetchMessages = () => {
        const results = DatabaseService.getQueueMessages();
        setMessages(results);
    };

    useEffect(() => {
        fetchMessages();
        // Refresh every 3 seconds to show live queue updates
        const interval = setInterval(fetchMessages, 3000);
        return () => clearInterval(interval);
    }, []);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'SENT': return '#4CAF50';
            case 'DELIVERED': return '#2196F3';
            case 'PENDING': return '#FF9800';
            case 'BUFFERED': return '#9C27B0';
            case 'PROCESSING': return '#00BCD4';
            default: return '#888';
        }
    };

    const getStatusText = (status: string) => {
        switch (status) {
            case 'SENT': return 'تم الإرسال';
            case 'DELIVERED': return 'تم التسليم';
            case 'PENDING': return 'قيد الانتظار';
            case 'BUFFERED': return 'في الطابور';
            case 'PROCESSING': return 'جاري المعالجة';
            default: return status;
        }
    };

    const renderItem = ({ item }: { item: any }) => (
        <View style={styles.card}>
            <View style={styles.row}>
                <Text style={styles.phone}>{item.phone_number}</Text>
                <Text style={styles.date}>{new Date(item.created_at).toLocaleTimeString()}</Text>
            </View>
            <Text style={styles.content} numberOfLines={2}>{item.content}</Text>
            
            <View style={styles.statusRow}>
                <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) + '33' }]}>
                    <Text style={[styles.statusText, { color: getStatusColor(item.status) }]}>
                        {getStatusText(item.status)}
                    </Text>
                </View>
                {item.attempts > 0 && (
                    <Text style={styles.attempts}>المحاولات: {item.attempts}</Text>
                )}
            </View>
        </View>
    );

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />
            <View style={[styles.header, { flexDirection: 'row-reverse' }]}>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.backText}>رجوع →</Text>
                </TouchableOpacity>
                <Text style={styles.title}>طابور الرسائل</Text>
                <TouchableOpacity onPress={fetchMessages}>
                    <Text style={styles.refreshText}>🔄</Text>
                </TouchableOpacity>
            </View>

            <FlatList
                data={messages}
                keyExtractor={(item) => item.id}
                renderItem={renderItem}
                contentContainerStyle={styles.list}
                ListEmptyComponent={<Text style={styles.empty}>لا توجد رسائل في الطابور حالياً.</Text>}
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
        borderLeftColor: '#4CAF50'
    },
    row: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 5 },
    phone: { color: '#FFF', fontWeight: 'bold' },
    date: { color: '#666', fontSize: 12 },
    content: { color: '#CCC', fontSize: 14, marginBottom: 10 },
    statusRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 5 },
    statusBadge: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 12 },
    statusText: { fontSize: 12, fontWeight: 'bold' },
    attempts: { color: '#888', fontSize: 12 },
    empty: { color: '#444', textAlign: 'center', marginTop: 50 }
});
