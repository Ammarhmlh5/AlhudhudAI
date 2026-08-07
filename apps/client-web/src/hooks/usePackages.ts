import { useState, useEffect } from 'react';
import apiClient from '../api/axios';

interface Package {
  id: number;
  service_id: number;
  name: string;
  code: string;
  price_usd: number;
  included_messages: number | null;
  overage_rate: number;
  is_unlimited: boolean;
  billing_cycle: string;
  rollover_enabled: boolean;
  rollover_limit: number | null;
  rollover_expiry_months: number | null;
  is_active: boolean;
  display_order: number;
  created_at: string;
  updated_at: string;
}

interface Subscription {
  id: number;
  tenant_id: string;
  package_id: number;
  start_date: string;
  end_date: string;
  messages_used: number;
  messages_remaining: number | null;
  overage_messages: number;
  overage_cost: string;
  status: string;
  auto_renew: boolean;
  grace_period_days: number;
  last_renewed_at?: string | null;
  next_renewal_date?: string | null;
  renewal_attempts: number;
  last_renewal_attempt?: string | null;
  created_at: string;
  updated_at: string;
}

interface SubscribeData {
  package_id: number;
  auto_renew?: boolean;
  idempotency_key?: string;
}

interface SubscribeResult {
  success: boolean;
  subscription_id: number;
  message: string;
  old_balance: string;
  new_balance: string;
}

interface Usage {
  subscription_id: number;
  package_id: number;
  package_name: string;
  messages_used: number;
  messages_remaining: number | null;
  included_messages: number | null;
  overage_messages: number;
  overage_cost: string;
  overage_rate: string;
  status: string;
  start_date: string;
  end_date: string;
  next_renewal_date?: string;
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
        const response = await apiClient.get('/api/v1/packages');

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

  const subscribe = async (data: SubscribeData): Promise<SubscribeResult | null> => {
    try {
      setLoading(true);
      setError(null);
      const idempotencyKey = crypto.randomUUID();
      const response = await apiClient.post('/api/v1/subscriptions', {
        ...data,
        idempotency_key: idempotencyKey
      });

      return response.data as SubscribeResult;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'فشل في الاشتراك في الباقة';
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
      const response = await apiClient.get('/api/v1/subscriptions/active');

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
      const response = await apiClient.get('/api/v1/subscriptions/usage');

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
      await apiClient.put(
        `/api/v1/subscriptions/${subscriptionId}/cancel`,
        {}
      );

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'فشل في إلغاء الاشتراك';
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
      if (autoRenew) {
        await apiClient.put(
          `/api/v1/subscriptions/${subscriptionId}/enable-auto-renew`,
          {}
        );
      } else {
        await apiClient.put(
          `/api/v1/subscriptions/${subscriptionId}/cancel`,
          {}
        );
      }

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'فشل في تحديث التجديد التلقائي';
      setError(errorMsg);
      console.error('Error updating auto-renew:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { updateAutoRenew, loading, error };
};
