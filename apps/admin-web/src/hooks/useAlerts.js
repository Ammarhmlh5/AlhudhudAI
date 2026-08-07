import { useState, useEffect, useCallback } from 'react';
import axios from '../api/axios';

/**
 * useAlerts Hook
 * 
 * Manages alert data fetching and operations
 * Provides methods for acknowledging and resolving alerts
 */
export const useAlerts = (filters = {}, options = {}) => {
  const { autoRefresh = false, refreshInterval = 30000 } = options;
  
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [summary, setSummary] = useState({
    total: 0,
    critical: 0,
    warning: 0,
    info: 0
  });

  const fetchAlerts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const params = new URLSearchParams();
      if (filters.severity) params.append('severity', filters.severity);
      if (filters.type) params.append('type', filters.type);
      if (filters.status) params.append('status', filters.status);
      if (filters.dateFrom) params.append('date_from', filters.dateFrom);
      if (filters.dateTo) params.append('date_to', filters.dateTo);

      const response = await axios.get(`/api/v1/admin/alerts?${params.toString()}`);
      
      const result = response.data && response.data.data ? response.data.data : response.data;
      if (result) {
        const list = result.alerts || [];
        setAlerts(list);
        setSummary(result.summary || {
          total: list.length,
          critical: list.filter(a => a.severity === 'critical').length,
          warning: list.filter(a => a.severity === 'warning').length,
          info: list.filter(a => a.severity === 'info').length
        });
      }
    } catch (err) {
      console.error('Failed to fetch alerts:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchAlerts();

    if (autoRefresh) {
      const interval = setInterval(fetchAlerts, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchAlerts, autoRefresh, refreshInterval]);

  const refetch = useCallback(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  return {
    alerts,
    summary,
    loading,
    error,
    refetch
  };
};

/**
 * useAcknowledgeAlert Hook
 * 
 * Provides method to acknowledge an alert
 */
export const useAcknowledgeAlert = () => {
  const [acknowledging, setAcknowledging] = useState(false);
  const [error, setError] = useState(null);

  const acknowledgeAlert = useCallback(async (alertId) => {
    try {
      setAcknowledging(true);
      setError(null);

      await axios.put(`/api/v1/admin/alerts/${alertId}/acknowledge`);
      
      return { success: true };
    } catch (err) {
      console.error('Failed to acknowledge alert:', err);
      setError(err);
      return { success: false, error: err };
    } finally {
      setAcknowledging(false);
    }
  }, []);

  return {
    acknowledgeAlert,
    acknowledging,
    error
  };
};

/**
 * useResolveAlert Hook
 * 
 * Provides method to resolve an alert
 */
export const useResolveAlert = () => {
  const [resolving, setResolving] = useState(false);
  const [error, setError] = useState(null);

  const resolveAlert = useCallback(async (alertId) => {
    try {
      setResolving(true);
      setError(null);

      await axios.put(`/api/v1/admin/alerts/${alertId}/resolve`);
      
      return { success: true };
    } catch (err) {
      console.error('Failed to resolve alert:', err);
      setError(err);
      return { success: false, error: err };
    } finally {
      setResolving(false);
    }
  }, []);

  return {
    resolveAlert,
    resolving,
    error
  };
};

/**
 * useAlertDetails Hook
 * 
 * Fetches detailed information for a specific alert
 */
export const useAlertDetails = (alertId) => {
  const [alert, setAlert] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!alertId) {
      setLoading(false);
      return;
    }

    const fetchAlertDetails = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await axios.get(`/api/v1/admin/alerts/${alertId}`);
        
        if (response.data) {
          setAlert(response.data.data || response.data);
        }
      } catch (err) {
        console.error('Failed to fetch alert details:', err);
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    fetchAlertDetails();
  }, [alertId]);

  return {
    alert,
    loading,
    error
  };
};

/**
 * useAlertConfig Hook
 * 
 * Manages alert configuration (thresholds and notification channels)
 */
export const useAlertConfig = () => {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);

  const fetchConfig = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await axios.get('/api/v1/admin/alerts/config');
      
      if (response.data) {
        setConfig(response.data.data || response.data);
      }
    } catch (err) {
      console.error('Failed to fetch alert config:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const saveConfig = useCallback(async (newConfig) => {
    try {
      setSaving(true);
      setError(null);

      await axios.post('/api/v1/admin/alerts/config', newConfig);
      setConfig(newConfig);
      
      return { success: true };
    } catch (err) {
      console.error('Failed to save alert config:', err);
      setError(err);
      return { success: false, error: err };
    } finally {
      setSaving(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  return {
    config,
    loading,
    saving,
    error,
    saveConfig,
    refetch: fetchConfig
  };
};

export default useAlerts;
