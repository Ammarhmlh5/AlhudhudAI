import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import apiClient from '../../api/axios';

export default function AlertsList() {
  const { t } = useTranslation();
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    severity: '',
    type: '',
    status: 'active',
    date_from: '',
    date_to: '',
  });
  const [selectedAlert, setSelectedAlert] = useState(null);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    fetchAlerts();
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchAlerts, 30000);
    return () => clearInterval(interval);
  }, [filters]);

  const fetchAlerts = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams();
      if (filters.severity) params.append('severity', filters.severity);
      if (filters.type) params.append('type', filters.type);
      if (filters.status) params.append('status', filters.status);
      if (filters.date_from) params.append('date_from', filters.date_from);
      if (filters.date_to) params.append('date_to', filters.date_to);

      const response = await apiClient.get(`/api/v1/admin/alerts?${params}`);

      setAlerts(response.data.alerts || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to load alerts');
    } finally {
      setLoading(false);
    }
  };

  const handleAcknowledge = async (alertId) => {
    try {
      await apiClient.put(`/api/v1/admin/alerts/${alertId}/acknowledge`);
      fetchAlerts();
    } catch (err) {
      console.error('Failed to acknowledge alert:', err);
    }
  };

  const handleResolve = async (alertId) => {
    try {
      await apiClient.put(`/api/v1/admin/alerts/${alertId}/resolve`);
      fetchAlerts();
    } catch (err) {
      console.error('Failed to resolve alert:', err);
    }
  };

  const getSeverityColor = (severity) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'info':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
    }
  };

  const getSeverityIcon = (severity) => {
    switch (severity) {
      case 'critical':
        return '🔴';
      case 'warning':
        return '⚠️';
      case 'info':
        return 'ℹ️';
      default:
        return '📢';
    }
  };

  if (loading && alerts.length === 0) {
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
          {t('alerts.title')}
        </h1>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {t('alerts.description')}
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <SummaryCard
          label={t('alerts.total')}
          value={alerts.length}
          icon="📊"
        />
        <SummaryCard
          label={t('alerts.critical')}
          value={alerts.filter(a => a.severity === 'critical').length}
          icon="🔴"
          color="text-red-600"
        />
        <SummaryCard
          label={t('alerts.warning')}
          value={alerts.filter(a => a.severity === 'warning').length}
          icon="⚠️"
          color="text-yellow-600"
        />
        <SummaryCard
          label={t('alerts.info')}
          value={alerts.filter(a => a.severity === 'info').length}
          icon="ℹ️"
          color="text-blue-600"
        />
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('alerts.filter_severity')}
            </label>
            <select
              value={filters.severity}
              onChange={(e) => setFilters({ ...filters, severity: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            >
              <option value="">{t('alerts.all_severities')}</option>
              <option value="critical">{t('alerts.critical')}</option>
              <option value="warning">{t('alerts.warning')}</option>
              <option value="info">{t('alerts.info')}</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('alerts.filter_type')}
            </label>
            <select
              value={filters.type}
              onChange={(e) => setFilters({ ...filters, type: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            >
              <option value="">{t('alerts.all_types')}</option>
              <option value="rating_change">{t('alerts.type_rating_change')}</option>
              <option value="server_failure">{t('alerts.type_server_failure')}</option>
              <option value="threshold_violation">{t('alerts.type_threshold')}</option>
              <option value="device_failure">{t('alerts.type_device_failure')}</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('alerts.filter_status')}
            </label>
            <select
              value={filters.status}
              onChange={(e) => setFilters({ ...filters, status: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            >
              <option value="">{t('alerts.all_statuses')}</option>
              <option value="active">{t('alerts.status_active')}</option>
              <option value="acknowledged">{t('alerts.status_acknowledged')}</option>
              <option value="resolved">{t('alerts.status_resolved')}</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('alerts.date_from')}
            </label>
            <input
              type="date"
              value={filters.date_from}
              onChange={(e) => setFilters({ ...filters, date_from: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('alerts.date_to')}
            </label>
            <input
              type="date"
              value={filters.date_to}
              onChange={(e) => setFilters({ ...filters, date_to: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
            />
          </div>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      {/* Alerts List */}
      <div className="space-y-4">
        {alerts.map((alert) => (
          <AlertCard
            key={alert.id}
            alert={alert}
            getSeverityColor={getSeverityColor}
            getSeverityIcon={getSeverityIcon}
            onAcknowledge={handleAcknowledge}
            onResolve={handleResolve}
            onViewDetails={() => {
              setSelectedAlert(alert);
              setShowModal(true);
            }}
          />
        ))}
      </div>

      {alerts.length === 0 && !loading && (
        <div className="text-center py-12 text-gray-500">
          {t('alerts.no_alerts')}
        </div>
      )}

      {/* Alert Details Modal */}
      {showModal && selectedAlert && (
        <AlertDetailsModal
          alert={selectedAlert}
          onClose={() => {
            setShowModal(false);
            setSelectedAlert(null);
          }}
          getSeverityColor={getSeverityColor}
          getSeverityIcon={getSeverityIcon}
        />
      )}
    </div>
  );
}

// Summary Card
function SummaryCard({ label, value, icon, color = 'text-gray-900' }) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600 dark:text-gray-400">{label}</p>
          <p className={`text-3xl font-bold ${color} dark:text-white mt-2`}>
            {value}
          </p>
        </div>
        <div className="text-4xl">{icon}</div>
      </div>
    </div>
  );
}

// Alert Card
function AlertCard({ alert, getSeverityColor, getSeverityIcon, onAcknowledge, onResolve, onViewDetails }) {
  const { t } = useTranslation();

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-3 mb-2">
            <span className="text-2xl">{getSeverityIcon(alert.severity)}</span>
            <span className={`px-3 py-1 text-xs font-semibold rounded ${getSeverityColor(alert.severity)}`}>
              {t(`alerts.${alert.severity}`)}
            </span>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {t(`alerts.type_${alert.alert_type}`)}
            </span>
          </div>

          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            {alert.title}
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {alert.message}
          </p>

          <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
            <span>
              {t('alerts.created')}: {new Date(alert.created_at).toLocaleString()}
            </span>
            {alert.acknowledged_at && (
              <span>
                {t('alerts.acknowledged')}: {new Date(alert.acknowledged_at).toLocaleString()}
              </span>
            )}
            {alert.resolved_at && (
              <span>
                {t('alerts.resolved')}: {new Date(alert.resolved_at).toLocaleString()}
              </span>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2 ml-4">
          <button
            onClick={onViewDetails}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {t('alerts.view_details')}
          </button>
          {alert.status === 'active' && (
            <button
              onClick={() => onAcknowledge(alert.id)}
              className="px-4 py-2 text-sm bg-yellow-600 text-white rounded-lg hover:bg-yellow-700"
            >
              {t('alerts.acknowledge')}
            </button>
          )}
          {(alert.status === 'active' || alert.status === 'acknowledged') && (
            <button
              onClick={() => onResolve(alert.id)}
              className="px-4 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700"
            >
              {t('alerts.resolve')}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// Alert Details Modal
function AlertDetailsModal({ alert, onClose, getSeverityColor, getSeverityIcon }) {
  const { t } = useTranslation();

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <span className="text-2xl">{getSeverityIcon(alert.severity)}</span>
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
              {alert.title}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Alert Info */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              {t('alerts.alert_info')}
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <InfoItem label={t('alerts.alert_id')} value={alert.id} />
              <InfoItem 
                label={t('alerts.severity')} 
                value={
                  <span className={`px-2 py-1 text-xs font-semibold rounded ${getSeverityColor(alert.severity)}`}>
                    {t(`alerts.${alert.severity}`)}
                  </span>
                } 
              />
              <InfoItem label={t('alerts.type')} value={t(`alerts.type_${alert.alert_type}`)} />
              <InfoItem label={t('alerts.status')} value={t(`alerts.status_${alert.status}`)} />
              <InfoItem label={t('alerts.created')} value={new Date(alert.created_at).toLocaleString()} />
              {alert.tenant_id && (
                <InfoItem label={t('alerts.company')} value={alert.tenant_name || alert.tenant_id} />
              )}
            </div>
          </div>

          {/* Message */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              {t('alerts.message')}
            </h3>
            <p className="text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
              {alert.message}
            </p>
          </div>

          {/* Context */}
          {alert.context && Object.keys(alert.context).length > 0 && (
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                {t('alerts.context')}
              </h3>
              <pre className="text-sm text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-700 p-4 rounded-lg overflow-x-auto">
                {JSON.stringify(alert.context, null, 2)}
              </pre>
            </div>
          )}

          {/* Timeline */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              {t('alerts.timeline')}
            </h3>
            <div className="space-y-3">
              <TimelineItem
                label={t('alerts.created')}
                value={new Date(alert.created_at).toLocaleString()}
              />
              {alert.acknowledged_at && (
                <TimelineItem
                  label={t('alerts.acknowledged')}
                  value={new Date(alert.acknowledged_at).toLocaleString()}
                  by={alert.acknowledged_by}
                />
              )}
              {alert.resolved_at && (
                <TimelineItem
                  label={t('alerts.resolved')}
                  value={new Date(alert.resolved_at).toLocaleString()}
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end p-6 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {t('alerts.close')}
          </button>
        </div>
      </div>
    </div>
  );
}

// Info Item
function InfoItem({ label, value }) {
  return (
    <div>
      <p className="text-sm text-gray-600 dark:text-gray-400">{label}</p>
      <p className="text-base font-medium text-gray-900 dark:text-white break-all">
        {value}
      </p>
    </div>
  );
}

// Timeline Item
function TimelineItem({ label, value, by }) {
  return (
    <div className="flex items-start gap-3">
      <div className="w-2 h-2 bg-blue-600 rounded-full mt-2" />
      <div>
        <p className="text-sm font-medium text-gray-900 dark:text-white">{label}</p>
        <p className="text-sm text-gray-600 dark:text-gray-400">{value}</p>
        {by && <p className="text-xs text-gray-500 dark:text-gray-500">by {by}</p>}
      </div>
    </div>
  );
}
