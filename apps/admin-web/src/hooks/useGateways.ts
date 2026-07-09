import { useState, useEffect } from 'react';
import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface Gateway {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  status: string;
  telecom_provider: string;
  telecom_username: string;
  telecom_balance: number;
  balance_alert_threshold: number;
  last_balance_check: string;
  subscription_type: string;
  subscription_fee: number;
  server_rental_fee: number;
  last_health_check: string;
  health_status: string;
  created_at: string;
  updated_at: string;
}

interface GatewayStats {
  gateway_id: string;
  date: string;
  messages_sent: number;
  messages_delivered: number;
  messages_failed: number;
  avg_response_time: number;
  max_response_time: number;
  uptime_percentage: number;
}

interface ResourceUsage {
  gateway_id: string;
  timestamp: string;
  cpu_usage: number;
  memory_usage: number;
  bandwidth_usage: number;
  database_connections: number;
  queries_per_second: number;
  redis_memory_usage: number;
}

interface CreateGatewayData {
  name: string;
  description: string;
  telecom_provider: string;
  telecom_username: string;
  telecom_password: string;
  telecom_api_key?: string;
  subscription_type: string;
  subscription_fee: number;
  server_rental_fee: number;
  balance_alert_threshold: number;
}

interface UpdateGatewayData {
  name?: string;
  description?: string;
  status?: string;
  subscription_fee?: number;
  server_rental_fee?: number;
  balance_alert_threshold?: number;
}

// Hook لجلب جميع البوابات
export const useGateways = () => {
  const [gateways, setGateways] = useState<Gateway[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchGateways = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`${API_URL}/api/gateways`, {
        withCredentials: true,
      });

      setGateways(response.data.gateways || []);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.error || 'فشل في جلب البوابات');
      // Handled

    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchGateways();
  }, []);

  return { gateways, loading, error, refetch: fetchGateways };
};

// Hook لإنشاء بوابة جديدة
export const useCreateGateway = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createGateway = async (data: CreateGatewayData): Promise<Gateway | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.post(`${API_URL}/api/gateways`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data.gateway;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في إنشاء البوابة';
      setError(errorMsg);
      // Handled

      return null;
    } finally {
      setLoading(false);
    }
  };

  return { createGateway, loading, error };
};

// Hook لجلب بوابة واحدة
export const useGateway = (gatewayId: string | null) => {
  const [gateway, setGateway] = useState<Gateway | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!gatewayId) {
      setGateway(null);
      setLoading(false);
      return;
    }

    const fetchGateway = async () => {
      try {
        setLoading(true);
        const response = await axios.get(`${API_URL}/api/gateways/${gatewayId}`, {
          withCredentials: true,
        });

        setGateway(response.data.gateway);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب البوابة');
        // Handled

      } finally {
        setLoading(false);
      }
    };

    fetchGateway();
  }, [gatewayId]);

  return { gateway, loading, error };
};

// Hook لتحديث بوابة
export const useUpdateGateway = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateGateway = async (
    gatewayId: string,
    data: UpdateGatewayData
  ): Promise<Gateway | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.put(`${API_URL}/api/gateways/${gatewayId}`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data.gateway;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تحديث البوابة';
      setError(errorMsg);
      console.error('Error updating gateway:', err);
      return null;
    } finally {
      setLoading(false);
    }
  };

  return { updateGateway, loading, error };
};

// Hook لجلب إحصائيات البوابة
export const useGatewayStats = (gatewayId: string | null, dateRange?: { from: string; to: string }) => {
  const [stats, setStats] = useState<GatewayStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!gatewayId) {
      setStats([]);
      setLoading(false);
      return;
    }

    const fetchStats = async () => {
      try {
        setLoading(true);
        const params = new URLSearchParams();
        if (dateRange?.from) params.append('from', dateRange.from);
        if (dateRange?.to) params.append('to', dateRange.to);

        const response = await axios.get(
          `${API_URL}/api/gateways/${gatewayId}/stats?${params.toString()}`,
          {
            withCredentials: true,
          }
        );

        setStats(response.data.stats || []);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب الإحصائيات');
        console.error('Error fetching gateway stats:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, [gatewayId, dateRange?.from, dateRange?.to]);

  return { stats, loading, error };
};

// Hook لجلب استخدام الموارد
export const useResourceUsage = (gatewayId: string | null, limit: number = 100) => {
  const [usage, setUsage] = useState<ResourceUsage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsage = async () => {
    if (!gatewayId) {
      setUsage([]);
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const response = await axios.get(
        `${API_URL}/api/gateways/${gatewayId}/resource-usage?limit=${limit}`,
        {
          withCredentials: true,
        }
      );

      setUsage(response.data.usage || []);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.error || 'فشل في جلب استخدام الموارد');
      console.error('Error fetching resource usage:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsage();
  }, [gatewayId, limit]);

  return { usage, loading, error, refetch: fetchUsage };
};

// Hook لتعليق/تفعيل البوابة
export const useGatewayActions = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const suspendGateway = async (gatewayId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await axios.post(
        `${API_URL}/api/gateways/${gatewayId}/suspend`,
        {},
        {
          withCredentials: true,
        }
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تعليق البوابة';
      setError(errorMsg);
      console.error('Error suspending gateway:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  const activateGateway = async (gatewayId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await axios.post(
        `${API_URL}/api/gateways/${gatewayId}/activate`,
        {},
        {
          withCredentials: true,
        }
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تفعيل البوابة';
      setError(errorMsg);
      console.error('Error activating gateway:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  const terminateGateway = async (gatewayId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await axios.post(
        `${API_URL}/api/gateways/${gatewayId}/terminate`,
        {},
        {
          withCredentials: true,
        }
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في إيقاف البوابة';
      setError(errorMsg);
      console.error('Error terminating gateway:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { suspendGateway, activateGateway, terminateGateway, loading, error };
};
