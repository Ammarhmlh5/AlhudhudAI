import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import apiClient from '../../api/axios';

export default function DistributionConfig() {
  const { t } = useTranslation();
  const { tenantId } = useParams();
  const [config, setConfig] = useState(null);
  const [devices, setDevices] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);
  const [selectedStrategy, setSelectedStrategy] = useState('round_robin');
  const [strategyConfig, setStrategyConfig] = useState({});

  useEffect(() => {
    if (tenantId) {
      fetchDistributionData();
    }
  }, [tenantId]);

  const fetchDistributionData = async () => {
    try {
      setLoading(true);
      const [configRes, devicesRes, metricsRes] = await Promise.all([
        apiClient.get(`/api/v1/admin/distribution/${tenantId}`),
        apiClient.get(`/api/v1/admin/distribution/${tenantId}/devices`),
        apiClient.get(`/api/v1/admin/distribution/${tenantId}/metrics`),
      ]);

      setConfig(configRes.data);
      setDevices(devicesRes.data.devices || []);
      setMetrics(metricsRes.data.metrics || []);
      setSelectedStrategy(configRes.data.strategy_type || 'round_robin');
      setStrategyConfig(configRes.data.strategy_config || {});
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to load distribution configuration');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    try {
      setSaving(true);
      await apiClient.put(`/api/v1/admin/distribution/${tenantId}`, {
        strategy_type: selectedStrategy,
        strategy_config: strategyConfig,
      });
      await fetchDistributionData();
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">{t('loading')}</div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
          {t('distribution.title')}
        </h1>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {t('distribution.description')}
        </p>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      {/* Strategy Selector */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('distribution.strategy_selector')}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <StrategyCard
            id="round_robin"
            title={t('distribution.round_robin')}
            description={t('distribution.round_robin_desc')}
            icon="🔄"
            selected={selectedStrategy === 'round_robin'}
            onSelect={() => setSelectedStrategy('round_robin')}
          />
          <StrategyCard
            id="load_balancing"
            title={t('distribution.load_balancing')}
            description={t('distribution.load_balancing_desc')}
            icon="⚖️"
            selected={selectedStrategy === 'load_balancing'}
            onSelect={() => setSelectedStrategy('load_balancing')}
          />
          <StrategyCard
            id="priority"
            title={t('distribution.priority')}
            description={t('distribution.priority_desc')}
            icon="⭐"
            selected={selectedStrategy === 'priority'}
            onSelect={() => setSelectedStrategy('priority')}
          />
          <StrategyCard
            id="custom"
            title={t('distribution.custom')}
            description={t('distribution.custom_desc')}
            icon="⚙️"
            selected={selectedStrategy === 'custom'}
            onSelect={() => setSelectedStrategy('custom')}
          />
        </div>
      </div>

      {/* Strategy Configuration */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('distribution.strategy_config')}
        </h2>
        <StrategyConfigForm
          strategy={selectedStrategy}
          config={strategyConfig}
          onChange={setStrategyConfig}
        />
        <div className="mt-6">
          <button
            onClick={handleSaveConfig}
            disabled={saving}
            className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? t('distribution.saving') : t('distribution.save_config')}
          </button>
        </div>
      </div>

      {/* Device Pool Manager */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('distribution.device_pool')}
        </h2>
        <DevicePoolManager devices={devices} onRefresh={fetchDistributionData} />
      </div>

      {/* Distribution Metrics */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('distribution.metrics')}
        </h2>
        <DistributionMetrics metrics={metrics} />
      </div>
    </div>
  );
}

// Strategy Card Component
function StrategyCard({ id, title, description, icon, selected, onSelect }) {
  return (
    <button
      onClick={onSelect}
      className={`p-6 rounded-lg border-2 transition-all text-left ${
        selected
          ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20'
          : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
      }`}
    >
      <div className="text-4xl mb-3">{icon}</div>
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
        {title}
      </h3>
      <p className="text-sm text-gray-600 dark:text-gray-400">{description}</p>
    </button>
  );
}

// Strategy Configuration Form Component
function StrategyConfigForm({ strategy, config, onChange }) {
  const { t } = useTranslation();

  const handleChange = (key, value) => {
    onChange({ ...config, [key]: value });
  };

  if (strategy === 'round_robin') {
    return (
      <div className="text-sm text-gray-600 dark:text-gray-400">
        {t('distribution.round_robin_no_config')}
      </div>
    );
  }

  if (strategy === 'load_balancing') {
    return (
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('distribution.load_threshold')}
          </label>
          <input
            type="number"
            value={config.load_threshold || 80}
            onChange={(e) => handleChange('load_threshold', parseInt(e.target.value))}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            min="0"
            max="100"
          />
        </div>
      </div>
    );
  }

  if (strategy === 'priority') {
    return (
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('distribution.priority_levels')}
          </label>
          <input
            type="number"
            value={config.priority_levels || 3}
            onChange={(e) => handleChange('priority_levels', parseInt(e.target.value))}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            min="1"
            max="10"
          />
        </div>
      </div>
    );
  }

  if (strategy === 'custom') {
    return (
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('distribution.custom_rules')}
          </label>
          <textarea
            value={JSON.stringify(config.custom_rules || {}, null, 2)}
            onChange={(e) => {
              try {
                const rules = JSON.parse(e.target.value);
                handleChange('custom_rules', rules);
              } catch (err) {
                // Invalid JSON, ignore
              }
            }}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white font-mono text-sm"
            rows="10"
          />
        </div>
      </div>
    );
  }

  return null;
}

// Device Pool Manager Component
function DevicePoolManager({ devices, onRefresh }) {
  const { t } = useTranslation();

  if (!devices || devices.length === 0) {
    return (
      <div className="text-gray-500 text-center py-4">
        {t('distribution.no_devices')}
      </div>
    );
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'online':
        return 'bg-green-500';
      case 'offline':
        return 'bg-red-500';
      case 'busy':
        return 'bg-yellow-500';
      default:
        return 'bg-gray-500';
    }
  };

  return (
    <div className="space-y-3">
      {devices.map((device) => (
        <div
          key={device.id}
          className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700 rounded-lg"
        >
          <div className="flex items-center gap-4">
            <div className={`w-3 h-3 rounded-full ${getStatusColor(device.status)}`} />
            <div>
              <div className="text-sm font-medium text-gray-900 dark:text-white">
                {device.device_name}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">
                {t('distribution.priority')}: {device.priority_level || 0}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-6">
            <div className="text-sm text-gray-600 dark:text-gray-400">
              {t('distribution.load')}: {device.current_load || 0}%
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              {t('distribution.messages')}: {device.total_messages_sent || 0}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">
              {t('distribution.success_rate')}: {((device.successful_messages / (device.total_messages_sent || 1)) * 100).toFixed(1)}%
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// Distribution Metrics Component
function DistributionMetrics({ metrics }) {
  const { t } = useTranslation();

  if (!metrics || metrics.length === 0) {
    return (
      <div className="text-gray-500 text-center py-4">
        {t('distribution.no_metrics')}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-900">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('distribution.device')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('distribution.assigned')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('distribution.sent')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('distribution.success_rate')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('distribution.avg_time')}
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {metrics.map((metric) => (
            <tr key={metric.device_id}>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {metric.device_name}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {metric.messages_assigned || 0}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {metric.messages_sent || 0}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {((metric.successful_messages / (metric.messages_sent || 1)) * 100).toFixed(1)}%
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {metric.average_processing_time_ms || 0}ms
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
