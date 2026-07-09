import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, StatusBar } from 'react-native';
import { useAppStore } from '../services/StorageService';

/**
 * LockedScreen.tsx
 * 
 * Displayed when the device is banned or deactivated by the admin.
 */

export const LockedScreen = () => {
    const { deviceUuid } = useAppStore();

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" />

            <View style={styles.iconContainer}>
                <Text style={styles.lockIcon}>🔒</Text>
            </View>

            <Text style={styles.title}>GATEWAY LOCKED</Text>
            <Text style={styles.message}>
                This device has been deactivated by the administrator.
                Please contact support to reactive your gateway license.
            </Text>

            <Text style={styles.serialText}>Device Serial: {deviceUuid}</Text>

            <TouchableOpacity style={styles.button}>
                <Text style={styles.buttonText}>CONTACT SUPPORT</Text>
            </TouchableOpacity>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
        justifyContent: 'center',
        alignItems: 'center',
        padding: 40,
    },
    iconContainer: {
        marginBottom: 20,
    },
    lockIcon: {
        fontSize: 60,
    },
    title: {
        color: '#F44336',
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 10,
    },
    message: {
        color: '#888',
        textAlign: 'center',
        fontSize: 14,
        lineHeight: 22,
        marginBottom: 30,
    },
    serialText: {
        color: '#444',
        fontSize: 12,
        marginBottom: 40,
    },
    button: {
        backgroundColor: '#333',
        paddingVertical: 15,
        paddingHorizontal: 30,
        borderRadius: 8,
        borderWidth: 1,
        borderColor: '#444',
    },
    buttonText: {
        color: '#FFF',
        fontWeight: 'bold',
    },
});
