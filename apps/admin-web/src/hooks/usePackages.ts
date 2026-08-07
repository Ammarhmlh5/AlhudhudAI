import { useState, useEffect } from 'react';
import axios from 'axios';
import { API_BASE_URL } from '../config';

interface Package {
  id: string;
  tenant_id: string;
  name: string;
  code: string;
  price_usd: number;
  included_messages: number | null;
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

interface PackageFilters {
  service_id?: string;
  is_active?: boolean;
}

interface CreatePackageData {
  service_id: number;
  name: string;
  code: string;
  price_usd: number;
  included_messages?: number | null;
  overage_rate?: number;
  is_unlimited?: boolean;
  billing_cycle: 'monthly' | 'yearly';
  rollover_enabled?: boolean;
  rollover_limit?: number | null;
  rollover_expiry_months?: number | null;
  display_order?: number;
}

interface UpdatePackageData {
  name?: string;
  code?: string;
  price_usd?: number;
  included_messages?: number | null;
  overage_rate?: number;
  is_unlimited?: boolean;
  billing_cycle?: 'monthly' | 'yearly';
  rollover_enabled?: boolean;
  rollover_limit?: number | null;
  rollover_expiry_months?: number | null;
  is_active?: boolean;
  display_order?: number;
}

// Hook لجلب جميع الباقات
export const usePackages = (filters?: PackageFilters) => {
  const [packages, setPackages] = useState<Package[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPackages = async () => {
      try {
        setLoading(true);
        
        const params = new URLSearchParams();
        if (filters?.service_id) params.append('service_id', filters.service_id);
        if (filters?.is_active !== undefined) params.append('is_active', String(filters.is_active));

        const response = await axios.get(`${API_BASE_URL}/api/v1/admin/packages?${params.toString()}`, {
          withCredentials: true,
        });

        setPackages(response.data || []);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب الباقات');
        // Handled by state

      } finally {
        setLoading(false);
      }
    };

    fetchPackages();
  }, [filters?.service_id, filters?.is_active]);

  return { packages, loading, error };
};

// Hook لجلب باقة واحدة
export const usePackage = (packageId: string | null) => {
  const [packageData, setPackageData] = useState<Package | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!packageId) {
      setPackageData(null);
      setLoading(false);
      return;
    }

    const fetchPackage = async () => {
      try {
        setLoading(true);
        const response = await axios.get(`${API_BASE_URL}/api/v1/admin/packages/${packageId}`, {
          withCredentials: true,
        });

        setPackageData(response.data);
        setError(null);
      } catch (err: any) {
        setError(err.response?.data?.error || 'فشل في جلب الباقة');
        // Handled

      } finally {
        setLoading(false);
      }
    };

    fetchPackage();
  }, [packageId]);

  return { package: packageData, loading, error };
};

// Hook لإنشاء باقة جديدة
export const useCreatePackage = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createPackage = async (data: CreatePackageData): Promise<Package | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.post(`${API_BASE_URL}/api/v1/admin/packages`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في إنشاء الباقة';
      setError(errorMsg);
      // Handled by return error

      return null;
    } finally {
      setLoading(false);
    }
  };

  return { createPackage, loading, error };
};

// Hook لتحديث باقة موجودة
export const useUpdatePackage = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updatePackage = async (packageId: string, data: UpdatePackageData): Promise<Package | null> => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.put(`${API_BASE_URL}/api/v1/admin/packages/${packageId}`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في تحديث الباقة';
      setError(errorMsg);
      // Handled by return error

      return null;
    } finally {
      setLoading(false);
    }
  };

  return { updatePackage, loading, error };
};

// Hook لحذف باقة
export const useDeletePackage = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deletePackage = async (packageId: string): Promise<boolean> => {
    try {
      setLoading(true);
      setError(null);
      await axios.delete(`${API_BASE_URL}/api/v1/admin/packages/${packageId}`, {
        withCredentials: true,
      });

      return true;
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'فشل في حذف الباقة';
      setError(errorMsg);
      // Handled by return error

      return false;
    } finally {
      setLoading(false);
    }
  };

  return { deletePackage, loading, error };
};
