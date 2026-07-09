import { useState, useEffect } from 'react';
import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

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

interface PackageFilters {
  service_id?: string;
  is_active?: boolean;
}

interface CreatePackageData {
  service_id: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  messages_included: number;
  overage_rate: number;
  rollover_enabled?: boolean;
  rollover_limit?: number;
  rollover_expiry_months?: number;
}

interface UpdatePackageData {
  name?: string;
  description?: string;
  price?: number;
  overage_rate?: number;
  rollover_enabled?: boolean;
  rollover_limit?: number;
  rollover_expiry_months?: number;
  is_active?: boolean;
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

        const response = await axios.get(`${API_URL}/api/packages?${params.toString()}`, {
          withCredentials: true,
        });

        setPackages(response.data.packages || []);
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
        const response = await axios.get(`${API_URL}/api/packages/${packageId}`, {
          withCredentials: true,
        });

        setPackageData(response.data.package);
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
      const response = await axios.post(`${API_URL}/api/packages`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data.package;
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
      const response = await axios.put(`${API_URL}/api/packages/${packageId}`, data, {
        withCredentials: true,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      return response.data.package;
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
      await axios.delete(`${API_URL}/api/packages/${packageId}`, {
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
