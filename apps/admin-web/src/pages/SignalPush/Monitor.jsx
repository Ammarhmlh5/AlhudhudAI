import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import apiClient from '../../api/axios';

export default function SignalPushMonitor() {
  const { t } = useTranslation();
  const { tenantId } = useParams();
  const [config, setConfig] = useState(null);
  const [statistics, setStatistics] = useState(null);
  const [violations, setViolations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [forcingMode, setForcingMode] = useState(false);

  useEffect(() => {
    if (tenantId) {
      fetchMonitorData();
    }
  }, [tenantId]);

  useEffect(() => {
    // Auto-refresh every 30 seconds
    const interval = setInterval(() => {
      if (tenantId) {
        fetchMonitorData();
      }
    }, 30000);
    return () => clearInterval(interval);
  }, [tenantId]);

  const fetchMonitorData = async () => {
    try {
      setLoading(true);
      // Backend only has stats endpoint currently
      const statsRes = await apiClient.get(`/api/v1/admin/signalpush/stats?tenant_id=${tenantId}`);
      
      setStatistics(statsRes.data);
      // Mock or fallback for missing endpoints
      setConfig({ current_mode: statsRes.data.mode || 'signal' });
      setViolations(statsRes.data.violations || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to load Signal/Push monitor data');
    } finally {
      setLoading(false);
    }
  };

  const handleForceMode = async (mode) => {
    try {
      setForcingMode(true);
      await apiClient.post(`/api/v1/admin/signalpush/force-mode?tenant_id=${tenantId}`, { mode });
      await fetchMonitorData();
      setError(null);
    } catch (err) {
      console.warn("Force mode not yet implemented on backend");
      setError("Force mode feature is coming soon");
    } finally {
      setForcingMode(false);
    }
  };

  if (loading && !statistics) {
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
          {t('signalpush.title')}
        </h1>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {t('signalpush.description')}
        </p>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      {/* Current Mode & Status */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              {t('signalpush.current_mode')}
            </h2>
            <div className="mt-2 flex items-center gap-3">
              <span
                className={`px-4 py-2 text-lg font-bold rounded-lg ${
                  config?.current_mode === 'signal'
                    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                    : 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                }`}
              >
                {config?.current_mode?.toUpperCase() || 'UNKNOWN'}
              </span>
              {config?.forced_mode && (
                <span className="px-3 py-1 text-sm font-semibold rounded bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                  {t('signalpush.forced')}
                </span>
              )}
            </div>
          </div>
          <ModeControl
            currentMode={config?.current_mode}
            onForceMode={handleForceMode}
            disabled={forcingMode}
          />
        </div>
      </div>

      {/* Timing Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title={t('signalpush.current_mpm')}
          value={statistics?.current_mpm?.toFixed(1) || '0'}
          subtitle={t('signalpush.messages_per_minute')}
          icon="📊"
        />
        <StatCard
          title={t('signalpush.daily_avg_mpm')}
          value={statistics?.daily_avg_mpm?.toFixed(1) || '0'}
          subtitle={t('signalpush.24h_average')}
          icon="📈"
        />
        <StatCard
          title={t('signalpush.violation_rate')}
          value={`${((statistics?.violation_count / (statistics?.total_messages || 1)) * 100).toFixed(1)}%`}
          subtitle={`${statistics?.violation_count || 0} ${t('signalpush.violations')}`}
          icon="⚠️"
          alert={statistics?.violation_count > 0}
        />
        <StatCard
          title={t('signalpush.anomaly_score')}
          value={statistics?.anomaly_score?.toFixed(2) || '0'}
          subtitle={t(`signalpush.anomaly_${statistics?.anomaly_level || 'none'}`)}
          icon="🔍"
          alert={statistics?.is_anomaly}
        />
      </div>

      {/* Anomaly Detection Status */}
      {statistics?.is_anomaly && (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
          <div className="flex items-start gap-4">
            <div className="text-4xl">⚠️</div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-yellow-900 dark:text-yellow-200">
                {t('signalpush.anomaly_detected')}
              </h3>
              <p className="mt-2 text-sm text-yellow-800 dark:text-yellow-300">
                {statistics?.anomaly_reason || t('signalpush.anomaly_reason_unknown')}
              </p>
              <div className="mt-4 grid grid-cols-3 gap-4">
                <div>
                  <div className="text-xs text-yellow-700 dark:text-yellow-400">
                    {t('signalpush.z_score')}
                  </div>
                  <div className="text-lg font-bold text-yellow-900 dark:text-yellow-200">
                    {statistics?.z_score?.toFixed(2) || '0'}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-yellow-700 dark:text-yellow-400">
                    {t('signalpush.peak_ratio')}
                  </div>
                  <div className="text-lg font-bold text-yellow-900 dark:text-yellow-200">
                    {statistics?.peak_ratio?.toFixed(2) || '0'}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-yellow-700 dark:text-yellow-400">
                    {t('signalpush.anomaly_level')}
                  </div>
                  <div className="text-lg font-bold text-yellow-900 dark:text-yellow-200">
                    {statistics?.anomaly_level || 'none'}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Violation Trends Chart */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('signalpush.violation_trends')}
        </h2>
        <ViolationTrendsChart violations={violations} />
      </div>

      {/* Recent Violations */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('signalpush.recent_violations')}
        </h2>
        <RecentViolations violations={violations.slice(0, 10)} />
      </div>

      {/* Timing Configuration */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          {t('signalpush.timing_config')}
        </h2>
        <TimingConfig config={config} />
      </div>
    </div>
  );
}

// Stat Card Component
function StatCard({ title, value, subtitle, icon, alert }) {
  return (
    <div className={`bg-white dark:bg-gray-800 rounded-lg shadow p-6 ${alert ? 'border-2 border-yellow-500' : ''}`}>
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
            {title}
          </p>
          <p className="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
            {value}
          </p>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {subtitle}
          </p>
        </div>
        <div className="text-4xl">{icon}</div>
      </div>
    </div>
  );
}

// Mode Control Component
function ModeControl({ currentMode, onForceMode, disabled }) {
  const { t } = useTranslation();
  const [showConfirm, setShowConfirm] = useState(false);
  const [selectedMode, setSelectedMode] = useState(null);

  const handleConfirm = () => {
    if (selectedMode) {
      onForceMode(selectedMode);
      setShowConfirm(false);
      setSelectedMode(null);
    }
  };

  return (
    <div className="flex gap-2">
      <button
        onClick={() => {
          setSelectedMode('signal');
          setShowConfirm(true);
        }}
        disabled={disabled || currentMode === 'signal'}
        className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {t('signalpush.force_signal')}
      </button>
      <button
        onClick={() => {
          setSelectedMode('push');
          setShowConfirm(true);
        }}
        disabled={disabled || currentMode === 'push'}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {t('signalpush.force_push')}
      </button>

      {/* Confirmation Dialog */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              {t('signalpush.confirm_mode_change')}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              {t('signalpush.confirm_mode_message', { mode: selectedMode?.toUpperCase() })}
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => {
                  setShowConfirm(false);
                  setSelectedMode(null);
                }}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                {t('signalpush.cancel')}
              </button>
              <button
                onClick={handleConfirm}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium"
              >
                {t('signalpush.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Violation Trends Chart Component
function ViolationTrendsChart({ violations }) {
  const { t } = useTranslation();

  if (!violations || violations.length === 0) {
    return (
      <div className="text-gray-500 text-center py-8">
        {t('signalpush.no_violations')}
      </div>
    );
  }

  // Group violations by hour
  const hourlyData = violations.reduce((acc, violation) => {
    const hour = new Date(violation.created_at).getHours();
    acc[hour] = (acc[hour] || 0) + 1;
    return acc;
  }, {});

  const maxCount = Math.max(...Object.values(hourlyData));

  return (
    <div className="space-y-2">
      {Object.entries(hourlyData)
        .sort(([a], [b]) => parseInt(a) - parseInt(b))
        .map(([hour, count]) => {
          const percentage = (count / maxCount) * 100;
          return (
            <div key={hour}>
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {hour}:00
                </span>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {count} {t('signalpush.violations')}
                </span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                <div
                  className="bg-red-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </div>
          );
        })}
    </div>
  );
}

// Recent Violations Component
function RecentViolations({ violations }) {
  const { t } = useTranslation();

  if (!violations || violations.length === 0) {
    return (
      <div className="text-gray-500 text-center py-4">
        {t('signalpush.no_violations')}
      </div>
    );
  }

  const getSeverityColor = (severity) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'minor':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
    }
  };

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-900">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('signalpush.time')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('signalpush.message_type')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('signalpush.expected_gap')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('signalpush.actual_gap')}
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {t('signalpush.severity')}
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {violations.map((violation) => (
            <tr key={violation.id}>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {new Date(violation.created_at).toLocaleTimeString()}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {violation.message_type}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {violation.expected_gap_ms}ms
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                {violation.actual_gap_ms}ms
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`px-2 py-1 text-xs font-semibold rounded ${getSeverityColor(violation.violation_severity)}`}>
                  {violation.violation_severity}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Timing Configuration Component
function TimingConfig({ config }) {
  const { t } = useTranslation();

  if (!config) {
    return (
      <div className="text-gray-500 text-center py-4">
        {t('signalpush.no_config')}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <ConfigItem
        label={t('signalpush.signal_min_gap')}
        value={`${config.signal_min_gap_ms || 0}ms`}
      />
      <ConfigItem
        label={t('signalpush.push_min_gap')}
        value={`${config.push_min_gap_ms || 0}ms`}
      />
      <ConfigItem
        label={t('signalpush.violation_threshold')}
        value={`${config.violation_rate_threshold || 0}%`}
      />
      <ConfigItem
        label={t('signalpush.z_score_threshold')}
        value={config.z_score_threshold || 0}
      />
      <ConfigItem
        label={t('signalpush.peak_ratio_threshold')}
        value={config.peak_ratio_threshold || 0}
      />
      <ConfigItem
        label={t('signalpush.current_mode')}
        value={config.current_mode?.toUpperCase() || 'UNKNOWN'}
      />
    </div>
  );
}

// Config Item Component
function ConfigItem({ label, value }) {
  return (
    <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
      <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">
        {label}
      </div>
      <div className="text-lg font-semibold text-gray-900 dark:text-white">
        {value}
      </div>
    </div>
  );
}
