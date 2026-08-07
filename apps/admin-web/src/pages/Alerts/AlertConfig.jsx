import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Save, Bell, Mail, MessageSquare, AlertTriangle } from 'lucide-react';
import axios from '../../api/axios';

/**
 * AlertConfig Component
 * 
 * Configure alert thresholds and notification channels
 */
const AlertConfig = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState({
    thresholds: {
      response_time_warning: 200,
      response_time_critical: 500,
      error_rate_warning: 1,
      error_rate_critical: 5,
      success_rate_warning: 95,
      success_rate_critical: 90
    },
    notification_channels: {
      email: {
        enabled: true,
        recipients: []
      },
      sms: {
        enabled: false,
        recipients: []
      },
      in_app: {
        enabled: true
      }
    }
  });

  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/v1/admin/alerts/config');
      if (response.data) {
        setConfig(response.data.data || response.data);
      }
    } catch (error) {
      console.error('Failed to fetch alert config:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await axios.post('/api/v1/admin/alerts/config', config);
      alert('Configuration saved successfully!');
    } catch (error) {
      console.error('Failed to save alert config:', error);
      alert('Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };

  const updateThreshold = (key, value) => {
    setConfig(prev => ({
      ...prev,
      thresholds: {
        ...prev.thresholds,
        [key]: parseFloat(value) || 0
      }
    }));
  };

  const toggleChannel = (channel) => {
    setConfig(prev => ({
      ...prev,
      notification_channels: {
        ...prev.notification_channels,
        [channel]: {
          ...prev.notification_channels[channel],
          enabled: !prev.notification_channels[channel].enabled
        }
      }
    }));
  };

  if (loading) {
    return (
      <div className="p-8">
        <div className="flex items-center justify-center h-64">
          <div className="text-slate-400">{t('loading')}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white">Alert Configuration</h1>
          <p className="text-slate-400 mt-1">Configure alert thresholds and notification channels</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-3 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white rounded-xl font-bold transition-all"
        >
          <Save className="w-5 h-5" />
          {saving ? 'Saving...' : 'Save Configuration'}
        </button>
      </div>

      {/* Threshold Configuration */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-6">
        <div className="flex items-center gap-3 pb-4 border-b border-slate-800">
          <AlertTriangle className="w-6 h-6 text-amber-500" />
          <h2 className="text-xl font-bold text-white">Alert Thresholds</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Response Time Thresholds */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Response Time (ms)</h3>
            
            <div>
              <label className="block text-sm text-slate-400 mb-2">Warning Threshold</label>
              <input
                type="number"
                value={config.thresholds.response_time_warning}
                onChange={(e) => updateThreshold('response_time_warning', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-amber-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger warning alert when response time exceeds this value</p>
            </div>

            <div>
              <label className="block text-sm text-slate-400 mb-2">Critical Threshold</label>
              <input
                type="number"
                value={config.thresholds.response_time_critical}
                onChange={(e) => updateThreshold('response_time_critical', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-rose-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger critical alert when response time exceeds this value</p>
            </div>
          </div>

          {/* Error Rate Thresholds */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Error Rate (%)</h3>
            
            <div>
              <label className="block text-sm text-slate-400 mb-2">Warning Threshold</label>
              <input
                type="number"
                step="0.1"
                value={config.thresholds.error_rate_warning}
                onChange={(e) => updateThreshold('error_rate_warning', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-amber-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger warning alert when error rate exceeds this percentage</p>
            </div>

            <div>
              <label className="block text-sm text-slate-400 mb-2">Critical Threshold</label>
              <input
                type="number"
                step="0.1"
                value={config.thresholds.error_rate_critical}
                onChange={(e) => updateThreshold('error_rate_critical', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-rose-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger critical alert when error rate exceeds this percentage</p>
            </div>
          </div>

          {/* Success Rate Thresholds */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white">Success Rate (%)</h3>
            
            <div>
              <label className="block text-sm text-slate-400 mb-2">Warning Threshold</label>
              <input
                type="number"
                step="0.1"
                value={config.thresholds.success_rate_warning}
                onChange={(e) => updateThreshold('success_rate_warning', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-amber-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger warning alert when success rate falls below this percentage</p>
            </div>

            <div>
              <label className="block text-sm text-slate-400 mb-2">Critical Threshold</label>
              <input
                type="number"
                step="0.1"
                value={config.thresholds.success_rate_critical}
                onChange={(e) => updateThreshold('success_rate_critical', e.target.value)}
                className="w-full px-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-rose-500"
              />
              <p className="text-xs text-slate-500 mt-1">Trigger critical alert when success rate falls below this percentage</p>
            </div>
          </div>
        </div>
      </div>

      {/* Notification Channels */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-6">
        <div className="flex items-center gap-3 pb-4 border-b border-slate-800">
          <Bell className="w-6 h-6 text-blue-500" />
          <h2 className="text-xl font-bold text-white">Notification Channels</h2>
        </div>

        <div className="space-y-4">
          {/* Email Notifications */}
          <div className="flex items-start gap-4 p-4 bg-slate-800/50 rounded-lg">
            <Mail className="w-6 h-6 text-blue-500 mt-1" />
            <div className="flex-1">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold text-white">Email Notifications</h3>
                <button
                  onClick={() => toggleChannel('email')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    config.notification_channels.email.enabled
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-700 text-slate-400'
                  }`}
                >
                  {config.notification_channels.email.enabled ? 'Enabled' : 'Disabled'}
                </button>
              </div>
              <p className="text-sm text-slate-400">Send alert notifications via email</p>
            </div>
          </div>

          {/* SMS Notifications */}
          <div className="flex items-start gap-4 p-4 bg-slate-800/50 rounded-lg">
            <MessageSquare className="w-6 h-6 text-emerald-500 mt-1" />
            <div className="flex-1">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold text-white">SMS Notifications</h3>
                <button
                  onClick={() => toggleChannel('sms')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    config.notification_channels.sms.enabled
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-700 text-slate-400'
                  }`}
                >
                  {config.notification_channels.sms.enabled ? 'Enabled' : 'Disabled'}
                </button>
              </div>
              <p className="text-sm text-slate-400">Send alert notifications via SMS</p>
            </div>
          </div>

          {/* In-App Notifications */}
          <div className="flex items-start gap-4 p-4 bg-slate-800/50 rounded-lg">
            <Bell className="w-6 h-6 text-purple-500 mt-1" />
            <div className="flex-1">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold text-white">In-App Notifications</h3>
                <button
                  onClick={() => toggleChannel('in_app')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    config.notification_channels.in_app.enabled
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-700 text-slate-400'
                  }`}
                >
                  {config.notification_channels.in_app.enabled ? 'Enabled' : 'Disabled'}
                </button>
              </div>
              <p className="text-sm text-slate-400">Show alert notifications in the admin interface</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AlertConfig;
