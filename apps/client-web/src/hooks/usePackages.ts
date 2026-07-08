import { useState, useEffect } from 'react';
import apiClient from '../api/axios';

interface Package {
  id: string;
  tenant_id: string;
  service_id: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  messages_included: number;
  overage_rate: number;
  rollover_enabled: boolean;
  rollover_limit: number;
  rollover_expiry_months: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface Subscription {
  id: string;
  tenant_id: string;
  package_id: string;
  start_date: string;
  end_date: string;
  status: string;
  messages_included: number;
  messages_used: number;
  messages_remaining: number;
  overage_messages: number;
  overage_cost: number;
  auto_renew: boolean;
  created_at: string;
  updated_at: string;
  package?: Package;
}

interface SubscribeData {
  package_id: string;
  auto_renew?: boolean;
  idempotency_key?: string;
}

interface Usage {
  subscription_id: string;
  messages_included: number;
  messages_used: number;
  messages_remaining: number;
  rollover_messages: number;
  overage_messages: number;
  overage_cost: number;
  total_available: number;
}

// Hook لجلب جميع الباقات المتاحة
export const usePackages = () => {
  const [packages, setPackages] = useState<Package[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPackages = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get('/api/packages');

        setPackages(response.data.packages || []);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب الباقات');
        console.error('Error fetching packages:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchPackages();
  }, []);

  return { packages, loading, error, refetch: () => setLoading(true) };
};

// Hook للاشتراك في باقة
export const useSubscribe = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const subscribe = async (data: SubscribeData): Promise<Subscription | null> => {
    try {
      setLoading(true);
      setError(null);
      const idempotencyKey = crypto.randomUUID();
      const response = await apiClient.post('/api/subscriptions', {
        ...data,
        idempotency_key: idempotencyKey
      });

      return response.data.subscription;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في الاشتراك في الباقة';
      setError(errorMsg);
      console.error('Error subscribing:', err);
      return null;
    } finally {
      setLoading(false);
    }
  };

  return { subscribe, loading, error };
};

// Hook لجلب الاشتراك الحالي
export const useSubscription = () => {
  const [subscription, setSubscription] = useState<Subscription | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSubscription = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/api/subscriptions/current');

      setSubscription(response.data.subscription);
      setError(null);
    } catch (err: any) {
      if (err.response?.status === 404) {
        setSubscription(null);
        setError(null);
      } else {
        setError(err.response?.data?.error || 'فشل في جلب الاشتراك');
        console.error('Error fetching subscription:', err);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSubscription();
  }, []);

  return { subscription, loading, error, refetch: fetchSubscription };
};

// Hook لجلب استخدام الباقة
export const useUsage = () => {
  const [usage, setUsage] = useState<Usage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsage = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/api/subscriptions/usage');

      setUsage(response.data.usage);
      setError(null);
    } catch (err: any) {
      if (err.response?.status === 404) {
        setUsage(null);
        setError(null);
      } else {
        setError(err.response?.data?.error || 'فشل في جلب الاستخدام');
        console.error('Error fetching usage:', err);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsage();
  }, []);

  return { usage, loading, error, refetch: fetchUsage };
};

// Hook لإلغاء الاشتراك
export const useCancelSubscription = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const cancelSubscription = async (subscriptionId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await apiClient.post(
        `/api/subscriptions/${subscriptionId}/cancel`,
        {}
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في إلغاء الاشتراك';
      setError(errorMsg);
      console.error('Error canceling subscription:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { cancelSubscription, loading, error };
};

// Hook لتحديث إعدادات التجديد التلقائي
export const useUpdateAutoRenew = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateAutoRenew = async (
    subscriptionId: string,
    autoRenew: boolean
  ): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await apiClient.put(
        `/api/subscriptions/${subscriptionId}/auto-renew`,
        { auto_renew: autoRenew }
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تحديث التجديد التلقائي';
      setError(errorMsg);
      console.error('Error updating auto-renew:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { updateAutoRenew, loading, error };
};
