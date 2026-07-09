const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');

/**
 * Metro configuration
 * https://reactnative.dev/docs/metro
 *
 * @type {import('metro-config').MetroConfig}
 */
const config = {
    maxWorkers: 1,
    resolver: {
        resolverMainFields: ['react-native', 'browser', 'main'],
        conditionNames: ['react-native', 'browser', 'import', 'require'],
    },
};

module.exports = mergeConfig(getDefaultConfig(__dirname), config);
