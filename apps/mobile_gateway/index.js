/**
 * @format
 */

import { AppRegistry } from 'react-native';
import { enableScreens } from 'react-native-screens';
import App from './App';
import { name as appName } from './app.json';

// Disable native screens optimization temporarily to fix the TypeError: right operand of 'in' is not an object
enableScreens(false);

AppRegistry.registerComponent(appName, () => App);
