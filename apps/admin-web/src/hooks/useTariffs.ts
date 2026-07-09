import { useState, useEffect } from 'react';
import apiClient from '../api/axios';

interface Tariff {
  id: string;
  tenant_id: string;
  service_id: string;
  country_code: string;
  operator_name: string;
  cost_per_message: number;
  currency: string;
  effective_from: string;
  effective_to: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface TariffFilters {
  service_id?: string;
  country_code?: string;
  operator_name?: string;
  is_active?: boolean;
}

interface CreateTariffData {
  service_id: string;
  country_code: string;
  operator_name: string;
  cost_per_message: number;
  currency: string;
  effective_from: string;
  effective_to?: string;
}

interface UpdateTariffData {
  cost_per_message?: number;
  effective_to?: string;
  is_active?: boolean;
}

// Hook لجلب جميع التعرفات
export const useTariffs = (filters?: TariffFilters) => {
  const [tariffs, setTariffs] = useState<Tariff[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTariffs = async () => {
      try {
        setLoading(true);
        
        const params = new URLSearchParams();
        if (filters?.service_id) params.append('service_id', filters.service_id);
        if (filters?.country_code) params.append('country_code', filters.country_code);
        if (filters?.operator_name) params.append('operator_name', filters.operator_name);
        if (filters?.is_active !== undefined) params.append('is_active', String(filters.is_active));

        const response = await apiClient.get(`/api/v1/admin/tariffs?${params.toString()}`);

        setTariffs(response.data.tariffs || []);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب التعرفات');
        // Handled by state

      } finally {
        setLoading(false);
      }
    };

    fetchTariffs();
  }, [filters?.service_id, filters?.country_code, filters?.operator_name, filters?.is_active]);

  return { tariffs, loading, error };
};

// Hook لإنشاء تعرفة جديدة
export const useCreateTariff = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createTariff = async (data: CreateTariffData): Promise<Tariff | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiClient.post('/api/v1/admin/tariffs', data);

      return response.data.tariff;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في إنشاء التعرفة';
      setError(errorMsg);
      // Handled by return error

      return null;
    } finally {
      setLoading(false);
    }
  };

  return { createTariff, loading, error };
};

// Hook لتحديث تعرفة موجودة
export const useUpdateTariff = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateTariff = async (tariffId: string, data: UpdateTariffData): Promise<Tariff | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiClient.put(`/api/v1/admin/tariffs/${tariffId}`, data);

      return response.data.tariff;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تحديث التعرفة';
      setError(errorMsg);
      // Handled by return error

      return null;
    } finally {
      setLoading(false);
    }
  };

  return { updateTariff, loading, error };
};

// Hook لحذف تعرفة
export const useDeleteTariff = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteTariff = async (tariffId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await apiClient.delete(`/api/v1/admin/tariffs/${tariffId}`);

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في حذف التعرفة';
      setError(errorMsg);
      // Handled by return error

      return false;
    } finally {
      setLoading(false);
    }
  };

  return { deleteTariff, loading, error };
};

// Hook لجلب تعرفة واحدة
export const useTariff = (tariffId: string | null) => {
  const [tariff, setTariff] = useState<Tariff | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!tariffId) {
      setTariff(null);
      setLoading(false);
      return;
    }

    const fetchTariff = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get(`/api/v1/admin/tariffs/${tariffId}`);

        setTariff(response.data.tariff);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب التعرفة');
        // Handled

      } finally {
        setLoading(false);
      }
    };

    fetchTariff();
  }, [tariffId]);

  return { tariff, loading, error };
};
