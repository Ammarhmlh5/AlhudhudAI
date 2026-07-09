import React, { useEffect, useState } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, View, StyleSheet } from 'react-native';
import { DashboardScreen } from './src/screens/DashboardScreen';
import { SettingsScreen } from './src/screens/SettingsScreen';
import { LogsScreen } from './src/screens/LogsScreen';
import { LockedScreen } from './src/screens/LockedScreen';
import { FailedMessagesScreen } from './src/screens/FailedMessagesScreen';
import { SentMessagesScreen } from './src/screens/SentMessagesScreen';
import { SupportScreen } from './src/screens/SupportScreen';
import { AppInitializer } from './src/services/AppInitializer';
import { useAppStore } from './src/services/StorageService';

import { LoginScreen } from './src/screens/LoginScreen';
import { ServicesScreen } from './src/screens/ServicesScreen';

const Stack = createNativeStackNavigator();

function App() {
  const [loading, setLoading] = useState(true);
  const store = useAppStore();
  const { isBanned, isAuthenticated, deviceUuid, token, apiKey } = store;

  useEffect(() => {
    const init = async () => {
      await AppInitializer.run();
      setLoading(false);
    };
    init();
  }, []);

  if (loading) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color="#FFB300" />
      </View>
    );
  }

  // Smart Navigation Logic
  let initialRoute = "Login";
  if (isBanned) {
    initialRoute = "Locked";
  } else if (isAuthenticated && token) {
    // If authenticated, we usually want Dashboard. 
    // LoginScreen now handles the "missing device/setup" case internally if needed.
    initialRoute = "Dashboard";
  }

  return (
    <NavigationContainer>
      <Stack.Navigator
        initialRouteName={initialRoute}
        screenOptions={{ headerShown: false }}
      >
        <Stack.Screen name="Login" component={LoginScreen} />
        <Stack.Screen name="Dashboard" component={DashboardScreen} />
        <Stack.Screen name="Settings" component={SettingsScreen} />
        <Stack.Screen name="Logs" component={LogsScreen} />
        <Stack.Screen name="SentMessages" component={SentMessagesScreen} />
        <Stack.Screen name="FailedMessages" component={FailedMessagesScreen} />
        <Stack.Screen name="Support" component={SupportScreen} />
        <Stack.Screen name="Services" component={ServicesScreen} />
        <Stack.Screen name="Locked" component={LockedScreen} />
      </Stack.Navigator>
    </NavigationContainer>
  );
}

const styles = StyleSheet.create({
  loadingContainer: {
    flex: 1,
    backgroundColor: '#121212',
    justifyContent: 'center',
    alignItems: 'center',
  },
});

export default App;
