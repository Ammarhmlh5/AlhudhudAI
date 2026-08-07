import { useState, useEffect, useCallback } from 'react';
import axios from '../api/axios';

/**
 * useRoutingConfig Hook
 * 
 * Fetches and manages routing configuration for a tenant
 */
export const useRoutingConfig = (tenantId) => {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchConfig = useCallback(async () => {
    if (!tenantId) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await axios.get(`/api/v1/admin/routing/${tenantId}`);
      
      if (response.data) {
        setConfig(response.data);
      }
    } catch (err) {
      console.error('Failed to fetch routing config:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  return {
    config,
    loading,
    error,
    refetch: fetchConfig
  };
};

/**
 * useUpdateRoutingConfig Hook
 * 
 * Provides method to update routing configuration
 */
export const useUpdateRoutingConfig = () => {
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState(null);

  const updateConfig = useCallback(async (tenantId, newConfig) => {
    try {
      setUpdating(true);
      setError(null);

      await axios.put(`/api/v1/admin/routing/${tenantId}`, newConfig);
      
      return { success: true };
    } catch (err) {
      console.error('Failed to update routing config:', err);
      setError(err);
      return { success: false, error: err };
    } finally {
      setUpdating(false);
    }
  }, []);

  return {
    updateConfig,
    updating,
    error
  };
};

/**
 * useRoutingMetrics Hook
 * 
 * Fetches routing metrics for a tenant
 */
export const useRoutingMetrics = (tenantId, options = {}) => {
  const { autoRefresh = false, refreshInterval = 30000 } = options;
  
  const [metrics, setMetrics] = useState({
    total_requests: 0,
    successful_requests: 0,
    failed_requests: 0,
    average_latency_ms: 0
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchMetrics = useCallback(async () => {
    if (!tenantId) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await axios.get(`/api/v1/admin/routing/${tenantId}/metrics`);
      
      if (response.data) {
        setMetrics(response.data);
      }
    } catch (err) {
      console.error('Failed to fetch routing metrics:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  useEffect(() => {
    fetchMetrics();

    if (autoRefresh) {
      const interval = setInterval(fetchMetrics, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchMetrics, autoRefresh, refreshInterval]);

  return {
    metrics,
    loading,
    error,
    refetch: fetchMetrics
  };
};

/**
 * useRoutingHealth Hook
 * 
 * Monitors health status of dedicated server routing
 */
export const useRoutingHealth = (tenantId, serverId) => {
  const [health, setHealth] = useState({
    status: 'unknown',
    response_time_ms: 0,
    error_rate: 0,
    last_check: null
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchHealth = useCallback(async () => {
    if (!tenantId || !serverId) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await axios.get(`/api/v1/admin/health/servers/${serverId}`);
      
      const result = response.data && response.data.data ? response.data.data : response.data;
      if (result) {
        setHealth({
          status: result.health_status || 'unknown',
          response_time_ms: result.response_time_ms || 0,
          error_rate: result.error_rate || 0,
          last_check: result.last_check || null
        });
      }
    } catch (err) {
      console.error('Failed to fetch routing health:', err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [tenantId, serverId]);

  useEffect(() => {
    fetchHealth();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchHealth, 30000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  return {
    health,
    loading,
    error,
    refetch: fetchHealth
  };
};

/**
 * useRoutingValidation Hook
 * 
 * Validates routing configuration before saving
 */
export const useRoutingValidation = () => {
  const validateConfig = useCallback((config) => {
    const errors = [];

    // Validate routing type
    if (!config.routing_type) {
      errors.push('Routing type is required');
    }

    if (!['shared', 'dedicated'].includes(config.routing_type)) {
      errors.push('Invalid routing type. Must be "shared" or "dedicated"');
    }

    // Validate dedicated server configuration
    if (config.routing_type === 'dedicated') {
      if (!config.dedicated_server_id) {
        errors.push('Dedicated server ID is required for dedicated routing');
      }

      if (!config.dedicated_server_url) {
        errors.push('Dedicated server URL is required for dedicated routing');
      } else {
        // Validate URL format
        try {
          new URL(config.dedicated_server_url);
        } catch (err) {
          errors.push('Invalid server URL format');
        }
      }
    }

    return {
      isValid: errors.length === 0,
      errors
    };
  }, []);

  return {
    validateConfig
  };
};

/**
 * Combined hook for routing management
 */
export const useRouting = (tenantId, options = {}) => {
  const { config, loading: configLoading, error: configError, refetch: refetchConfig } = useRoutingConfig(tenantId);
  const { metrics, loading: metricsLoading, error: metricsError, refetch: refetchMetrics } = useRoutingMetrics(tenantId, options);
  const { updateConfig, updating, error: updateError } = useUpdateRoutingConfig();
  const { validateConfig } = useRoutingValidation();

  const saveConfig = useCallback(async (newConfig) => {
    // Validate before saving
    const validation = validateConfig(newConfig);
    if (!validation.isValid) {
      return {
        success: false,
        errors: validation.errors
      };
    }

    // Update config
    const result = await updateConfig(tenantId, newConfig);
    
    if (result.success) {
      // Refresh data after successful update
      refetchConfig();
      refetchMetrics();
    }

    return result;
  }, [tenantId, updateConfig, validateConfig, refetchConfig, refetchMetrics]);

  return {
    config,
    metrics,
    loading: configLoading || metricsLoading,
    updating,
    error: configError || metricsError || updateError,
    saveConfig,
    refetch: () => {
      refetchConfig();
      refetchMetrics();
    }
  };
};

export default useRouting;
