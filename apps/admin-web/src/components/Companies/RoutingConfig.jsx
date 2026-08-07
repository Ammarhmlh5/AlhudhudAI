import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Save, Server, Activity, TrendingUp } from 'lucide-react';
import axios from '../../api/axios';

/**
 * RoutingConfig Component
 * 
 * Configure message routing for a company
 * - Routing type: shared or dedicated
 * - Dedicated server configuration
 * - Routing metrics display
 */
const RoutingConfig = ({ tenantId }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState({
    routing_type: 'shared',
    dedicated_server_id: '',
    dedicated_server_url: '',
    health_status: 'unknown'
  });
  const [metrics, setMetrics] = useState({
    total_requests: 0,
    successful_requests: 0,
    failed_requests: 0,
    average_latency_ms: 0
  });

  useEffect(() => {
    if (tenantId) {
      fetchConfig();
      fetchMetrics();
    }
  }, [tenantId]);

  const fetchConfig = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`/api/v1/admin/routing/${tenantId}`);
      if (response.data) {
        setConfig(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch routing config:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchMetrics = async () => {
    try {
      const response = await axios.get(`/api/v1/admin/routing/${tenantId}/metrics`);
      if (response.data) {
        setMetrics(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch routing metrics:', error);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await axios.put(`/api/v1/admin/routing/${tenantId}`, config);
      alert(t('routingConfig.saveSuccess'));
      fetchMetrics(); // Refresh metrics after save
    } catch (error) {
      console.error('Failed to save routing config:', error);
      alert(t('routingConfig.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const updateConfig = (key, value) => {
    setConfig(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const getHealthStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'healthy':
        return 'text-emerald-500 bg-emerald-500/10 border-emerald-500/20';
      case 'degraded':
        return 'text-amber-500 bg-amber-500/10 border-amber-500/20';
      case 'unhealthy':
        return 'text-rose-500 bg-rose-500/10 border-rose-500/20';
      default:
        return 'text-slate-500 bg-slate-500/10 border-slate-500/20';
    }
  };

  const getHealthStatusLabel = (status) => {
    switch (status?.toLowerCase()) {
      case 'healthy':
        return t('servers.healthy');
      case 'degraded':
        return t('servers.degraded');
      case 'unhealthy':
        return t('servers.unhealthy');
      default:
        return t('routingConfig.healthUnknown');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-400">{t('loading')}</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Server className="w-6 h-6 text-blue-500" />
          <h2 className="text-xl font-bold text-white">{t('routingConfig.title')}</h2>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-4 py-2 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white rounded-lg font-medium transition-all"
        >
          <Save className="w-4 h-4" />
          {saving ? t('distribution.saving') : t('routingConfig.save')}
        </button>
      </div>

      {/* Routing Type Selector */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
        <h3 className="text-lg font-semibold text-white">{t('routingConfig.routingType')}</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Shared Routing */}
          <button
            onClick={() => updateConfig('routing_type', 'shared')}
            className={`p-4 rounded-lg border-2 transition-all text-left ${
              config.routing_type === 'shared'
                ? 'border-blue-500 bg-blue-500/10'
                : 'border-slate-700 bg-slate-800/50 hover:border-slate-600'
            }`}
          >
            <div className="flex items-center gap-3 mb-2">
              <div className={`w-4 h-4 rounded-full border-2 ${
                config.routing_type === 'shared'
                  ? 'border-blue-500 bg-blue-500'
                  : 'border-slate-600'
              }`}>
                {config.routing_type === 'shared' && (
                  <div className="w-full h-full rounded-full bg-white scale-50" />
                )}
              </div>
              <span className="font-semibold text-white">{t('routingConfig.sharedRouting')}</span>
            </div>
            <p className="text-sm text-slate-400">
              {t('routingConfig.sharedRoutingDesc')}
            </p>
          </button>

          {/* Dedicated Routing */}
          <button
            onClick={() => updateConfig('routing_type', 'dedicated')}
            className={`p-4 rounded-lg border-2 transition-all text-left ${
              config.routing_type === 'dedicated'
                ? 'border-blue-500 bg-blue-500/10'
                : 'border-slate-700 bg-slate-800/50 hover:border-slate-600'
            }`}
          >
            <div className="flex items-center gap-3 mb-2">
              <div className={`w-4 h-4 rounded-full border-2 ${
                config.routing_type === 'dedicated'
                  ? 'border-blue-500 bg-blue-500'
                  : 'border-slate-600'
              }`}>
                {config.routing_type === 'dedicated' && (
                  <div className="w-full h-full rounded-full bg-white scale-50" />
                )}
              </div>
              <span className="font-semibold text-white">{t('routingConfig.dedicatedServer')}</span>
            </div>
            <p className="text-sm text-slate-400">
              {t('routingConfig.dedicatedServerDesc')}
            </p>
          </button>
        </div>
      </div>

      {/* Dedicated Server Configuration */}
      {config.routing_type === 'dedicated' && (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
          <h3 className="text-lg font-semibold text-white">{t('routingConfig.dedicatedServerSettings')}</h3>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm text-slate-400 mb-2">{t('servers.server_id')}</label>
              <input
                type="text"
                value={config.dedicated_server_id}
                onChange={(e) => updateConfig('dedicated_server_id', e.target.value)}
                placeholder={t('routingConfig.serverIdPlaceholder')}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm text-slate-400 mb-2">{t('servers.server_url')}</label>
              <input
                type="url"
                value={config.dedicated_server_url}
                onChange={(e) => updateConfig('dedicated_server_url', e.target.value)}
                placeholder="https://server.example.com"
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* Health Status */}
            {config.dedicated_server_id && (
              <div>
                <label className="block text-sm text-slate-400 mb-2">{t('routingConfig.healthStatus')}</label>
                <div className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg border ${getHealthStatusColor(config.health_status)}`}>
                  <Activity className="w-4 h-4" />
                  <span className="font-medium">{getHealthStatusLabel(config.health_status)}</span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Routing Metrics */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
        <div className="flex items-center gap-3 mb-4">
          <TrendingUp className="w-6 h-6 text-emerald-500" />
          <h3 className="text-lg font-semibold text-white">{t('routingConfig.routingMetrics')}</h3>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-sm text-slate-400 mb-1">{t('routingConfig.totalRequests')}</p>
            <p className="text-2xl font-bold text-white">{metrics.total_requests.toLocaleString()}</p>
          </div>

          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-sm text-slate-400 mb-1">{t('routingConfig.successful')}</p>
            <p className="text-2xl font-bold text-emerald-500">{metrics.successful_requests.toLocaleString()}</p>
          </div>

          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-sm text-slate-400 mb-1">{t('routingConfig.failed')}</p>
            <p className="text-2xl font-bold text-rose-500">{metrics.failed_requests.toLocaleString()}</p>
          </div>

          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-sm text-slate-400 mb-1">{t('routingConfig.avgLatency')}</p>
            <p className="text-2xl font-bold text-blue-500">{metrics.average_latency_ms}ms</p>
          </div>
        </div>

        {/* Success Rate */}
        {metrics.total_requests > 0 && (
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-slate-400">{t('routingConfig.successRate')}</span>
              <span className="text-sm font-medium text-white">
                {((metrics.successful_requests / metrics.total_requests) * 100).toFixed(2)}%
              </span>
            </div>
            <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-emerald-500 transition-all"
                style={{
                  width: `${(metrics.successful_requests / metrics.total_requests) * 100}%`
                }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Warning for Dedicated Routing */}
      {config.routing_type === 'dedicated' && (
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-xl p-4">
          <div className="flex items-start gap-3">
            <Activity className="w-5 h-5 text-amber-500 mt-0.5" />
            <div>
              <h4 className="font-semibold text-amber-500 mb-1">{t('routingConfig.noFallbackPolicy')}</h4>
              <p className="text-sm text-amber-400/80">
                {t('routingConfig.noFallbackPolicyDesc')}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RoutingConfig;
