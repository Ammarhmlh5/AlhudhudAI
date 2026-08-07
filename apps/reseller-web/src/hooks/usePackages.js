import { useState, useEffect } from 'react';
import apiClient from '../api/axios';

// Fetch the packages catalog (scoped to the reseller's own tenant)
export const usePackages = () => {
  const [packages, setPackages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchPackages = () => {
    apiClient.get('/api/v1/reseller/packages')
      .then((res) => {
        setPackages(res.data.packages || []);
        setError(null);
      })
      .catch((err) => {
        setError(err.response?.data?.error || 'Failed to load packages');
        console.error('Error fetching packages:', err);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchPackages();
  }, []);

  return { packages, loading, error, refetch: fetchPackages };
};

// Subscribe the reseller's own tenant to a package
export const useSubscribe = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const subscribe = async (data) => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiClient.post('/api/v1/reseller/subscriptions', {
        ...data,
        idempotency_key: crypto.randomUUID()
      });
      return response.data;
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'Failed to subscribe';
      setError(errorMsg);
      console.error('Error subscribing:', err);
      return null;
    } finally {
      setLoading(false);
    }
  };

  return { subscribe, loading, error };
};

// Fetch the current active subscription
export const useSubscription = () => {
  const [subscription, setSubscription] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchSubscription = () => {
    apiClient.get('/api/v1/reseller/subscriptions/active')
      .then((res) => {
        setSubscription(res.data.subscription);
        setError(null);
      })
      .catch((err) => {
        if (err.response?.status === 404) {
          setSubscription(null);
          setError(null);
        } else {
          setError(err.response?.data?.error || 'Failed to load subscription');
          console.error('Error fetching subscription:', err);
        }
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchSubscription();
  }, []);

  return { subscription, loading, error, refetch: fetchSubscription };
};

// Fetch usage for the current subscription
export const useUsage = () => {
  const [usage, setUsage] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchUsage = () => {
    apiClient.get('/api/v1/reseller/subscriptions/usage')
      .then((res) => {
        setUsage(res.data.usage);
        setError(null);
      })
      .catch((err) => {
        if (err.response?.status === 404) {
          setUsage(null);
          setError(null);
        } else {
          setError(err.response?.data?.error || 'Failed to load usage');
          console.error('Error fetching usage:', err);
        }
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchUsage();
  }, []);

  return { usage, loading, error, refetch: fetchUsage };
};

// Cancel a subscription or toggle auto-renew off
export const useCancelSubscription = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const cancelSubscription = async (subscriptionId) => {
    try {
      setLoading(true);
      setError(null);
      await apiClient.put(`/api/v1/reseller/subscriptions/${subscriptionId}/cancel`, {});
      return true;
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'Failed to cancel subscription';
      setError(errorMsg);
      console.error('Error canceling subscription:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { cancelSubscription, loading, error };
};

// Toggle auto-renew on/off for a subscription
export const useUpdateAutoRenew = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const updateAutoRenew = async (subscriptionId, autoRenew) => {
    try {
      setLoading(true);
      setError(null);
      if (autoRenew) {
        await apiClient.put(`/api/v1/reseller/subscriptions/${subscriptionId}/enable-auto-renew`, {});
      } else {
        await apiClient.put(`/api/v1/reseller/subscriptions/${subscriptionId}/cancel`, {});
      }
      return true;
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.response?.data?.message || 'Failed to update auto-renew';
      setError(errorMsg);
      console.error('Error updating auto-renew:', err);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return { updateAutoRenew, loading, error };
};
