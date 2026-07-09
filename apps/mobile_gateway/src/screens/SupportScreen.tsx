import React, { useState } from 'react';
import { View, Text, StyleSheet, TextInput, TouchableOpacity, ScrollView, Alert, Image, StatusBar } from 'react-native';
import DocumentPicker, { types } from 'react-native-document-picker';
import { useAppStore } from '../services/StorageService';

/**
 * SupportScreen.tsx
 * 
 * Allows users to send technical support messages and attach document/receipts.
 */

export const SupportScreen = ({ navigation }: any) => {
    const store = useAppStore();
    const [message, setMessage] = useState('');
    const [attachment, setAttachment] = useState<any>(null);
    const [loading, setLoading] = useState(false);

    const handlePickDocument = async () => {
        try {
            const res = await DocumentPicker.pick({
                type: [types.allFiles],
            });
            setAttachment(res[0]);
        } catch (err) {
            if (DocumentPicker.isCancel(err)) {
                // User cancelled the picker
            } else {
                throw err;
            }
        }
    };

    const handleSend = async () => {
        if (!message.trim()) {
            Alert.alert('تنبيه', 'يرجى كتابة نص الرسالة أولاً.');
            return;
        }

        setLoading(true);
        // Simulate sending to support API
        setTimeout(() => {
            setLoading(false);
            Alert.alert(
                'تم الإرسال',
                'تم استلام طلبك بنجاح، فريق الدعم سيتواصل معك عبر الرسائل قريباً.',
                [{ text: 'حسناً', onPress: () => navigation.goBack() }]
            );
        }, 2000);
    };

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />
            <View style={[styles.header, { flexDirection: 'row-reverse' }]}>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.backText}>رجوع →</Text>
                </TouchableOpacity>
                <Text style={styles.title}>الدعم الفني</Text>
                <View style={{ width: 40 }} />
            </View>

            <ScrollView contentContainerStyle={styles.content}>
                <Text style={styles.description}>
                    يمكنك هنا التواصل مع فريق الدعم الفني مباشرة، أو إرسال مستندات التحصيل وقسائم السداد.
                </Text>

                <View style={[styles.section, { alignItems: 'flex-end' }]}>
                    <Text style={styles.label}>رسالتك</Text>
                    <TextInput
                        style={[styles.input, styles.textArea]}
                        multiline
                        numberOfLines={6}
                        value={message}
                        onChangeText={setMessage}
                        placeholder="اكتب هنا تفاصيل طلبك أو المشكلة التي تواجهها..."
                        placeholderTextColor="#444"
                        textAlignVertical="top"
                    />

                    <Text style={styles.label}>المرفقات (اختياري)</Text>
                    <TouchableOpacity style={styles.attachmentButton} onPress={handlePickDocument}>
                        <Text style={styles.attachmentText}>
                            {attachment ? `📎 ${attachment.name}` : 'إرفاق مستند أو صورة 📄'}
                        </Text>
                    </TouchableOpacity>
                    {attachment && (
                        <TouchableOpacity onPress={() => setAttachment(null)}>
                            <Text style={styles.removeText}>إزالة المرفق ❌</Text>
                        </TouchableOpacity>
                    )}
                </View>

                <TouchableOpacity
                    style={[styles.sendButton, loading && { opacity: 0.5 }]}
                    onPress={handleSend}
                    disabled={loading}
                >
                    <Text style={styles.sendButtonText}>{loading ? 'جاري الإرسال...' : 'إرسال الطلب'}</Text>
                </TouchableOpacity>

                <View style={styles.infoBox}>
                    <Text style={styles.infoTitle}>للمساعدة العاجلة:</Text>
                    <Text style={styles.infoText}>رقم الدعم: +966 XX XXX XXXX</Text>
                    <Text style={styles.infoText}>البريد: support@alhudhud.test</Text>
                </View>
            </ScrollView>
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
        borderBottomWidth: 1,
        borderBottomColor: '#333'
    },
    title: { color: '#FFB300', fontSize: 18, fontWeight: 'bold' },
    backText: { color: '#2196F3', fontSize: 16 },
    content: { padding: 20 },
    description: { color: '#888', textAlign: 'center', lineHeight: 22, marginBottom: 30 },
    section: { marginBottom: 20 },
    label: { color: '#FFB300', fontSize: 14, fontWeight: 'bold', marginBottom: 10 },
    input: {
        backgroundColor: '#1E1E1E',
        color: '#FFF',
        borderRadius: 8,
        padding: 15,
        borderWidth: 1,
        borderColor: '#333',
        textAlign: 'right',
        width: '100%'
    },
    textArea: { height: 120 },
    attachmentButton: {
        width: '100%',
        height: 56,
        backgroundColor: '#263238',
        borderRadius: 8,
        borderStyle: 'dashed',
        borderWidth: 1,
        borderColor: '#455A64',
        justifyContent: 'center',
        alignItems: 'center',
        marginTop: 5
    },
    attachmentText: { color: '#90A4AE', fontSize: 14 },
    removeText: { color: '#F44336', fontSize: 12, marginTop: 10, textAlign: 'right' },
    sendButton: {
        backgroundColor: '#2196F3',
        height: 56,
        borderRadius: 8,
        justifyContent: 'center',
        alignItems: 'center',
        marginTop: 20
    },
    sendButtonText: { color: '#FFF', fontWeight: 'bold', fontSize: 16 },
    infoBox: {
        marginTop: 40,
        padding: 20,
        backgroundColor: '#1A1A1A',
        borderRadius: 8,
        borderWidth: 1,
        borderColor: '#222',
        alignItems: 'center'
    },
    infoTitle: { color: '#FFF', fontWeight: 'bold', marginBottom: 10 },
    infoText: { color: '#666', fontSize: 13, marginBottom: 5 }
});
