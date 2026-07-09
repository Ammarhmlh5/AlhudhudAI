import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, StatusBar, Dimensions } from 'react-native';
import { BottomTabNavigator } from '../components/BottomTabNavigator';

const { width } = Dimensions.get('window');

export const ServicesScreen = ({ navigation }: any) => {
    const services = [
        { id: 'sent', title: 'الرسائل المرسلة', icon: '✅', color: '#4CAF50', route: 'SentMessages' },
        { id: 'failed', title: 'الرسائل الفاشلة', icon: '❌', color: '#F44336', route: 'FailedMessages' },
        { id: 'recharge', title: 'شحن الرصيد', icon: '💳', color: '#2196F3', route: 'Support' },
        { id: 'packages', title: 'باقات الاشتراك', icon: '📦', color: '#FFB300', route: 'Support' },
    ];

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />
            <View style={styles.header}>
                <Text style={styles.title}>الخدمات</Text>
            </View>

            <ScrollView contentContainerStyle={styles.scroll}>
                <Text style={styles.sectionTitle}>خدمات بوابة الهدهد</Text>
                <View style={styles.grid}>
                    {services.map((service) => (
                        <TouchableOpacity
                            key={service.id}
                            style={styles.serviceCard}
                            onPress={() => navigation.navigate(service.route)}
                        >
                            <View style={[styles.iconContainer, { backgroundColor: service.color + '22' }]}>
                                <Text style={styles.icon}>{service.icon}</Text>
                            </View>
                            <Text style={styles.serviceTitle}>{service.title}</Text>
                        </TouchableOpacity>
                    ))}
                </View>

                <View style={styles.infoCard}>
                    <Text style={styles.infoTitle}>تخصيص الواجهة</Text>
                    <Text style={styles.infoDesc}>يمكنك اختيار الخدمات التي تود ظهورها كاختصارات في الواجهة الرئيسية قريباً.</Text>
                </View>
            </ScrollView>

            <BottomTabNavigator navigation={navigation} activeTab="services" />
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
        backgroundColor: '#1A1A1A',
        borderBottomWidth: 1,
        borderBottomColor: '#333',
        alignItems: 'center',
    },
    title: {
        fontSize: 20,
        fontWeight: 'bold',
        color: '#FFB300',
    },
    scroll: {
        padding: 20,
        paddingBottom: 120,
    },
    sectionTitle: {
        color: '#888',
        fontSize: 14,
        fontWeight: 'bold',
        marginBottom: 20,
        textAlign: 'right',
    },
    grid: {
        flexDirection: 'row-reverse',
        flexWrap: 'wrap',
        justifyContent: 'space-between',
    },
    serviceCard: {
        width: (width - 60) / 2,
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 20,
        marginBottom: 20,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#2A2A2A',
    },
    iconContainer: {
        width: 60,
        height: 60,
        borderRadius: 30,
        justifyContent: 'center',
        alignItems: 'center',
        marginBottom: 12,
    },
    icon: {
        fontSize: 30,
    },
    serviceTitle: {
        color: '#FFF',
        fontSize: 14,
        fontWeight: 'bold',
        textAlign: 'center',
    },
    infoCard: {
        backgroundColor: '#FFB30011',
        padding: 20,
        borderRadius: 16,
        borderWidth: 1,
        borderColor: '#FFB30033',
        marginTop: 10,
    },
    infoTitle: {
        color: '#FFB300',
        fontSize: 16,
        fontWeight: 'bold',
        marginBottom: 8,
        textAlign: 'right',
    },
    infoDesc: {
        color: '#AAA',
        fontSize: 13,
        lineHeight: 20,
        textAlign: 'right',
    },
});
