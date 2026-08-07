import { createContext, useState, useEffect, useContext, useCallback } from 'react';
import apiClient from '../api/axios';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [tenant, setTenant] = useState(null);
  const [loading, setLoading] = useState(true);

  const checkAuth = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/v1/auth/me');
      setUser(response.data.user);
      setTenant(response.data.tenant || null);
    } catch (error) {
      localStorage.removeItem('company_token');
      setUser(null);
      setTenant(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (localStorage.getItem('company_token')) {
      checkAuth();
    } else {
      setLoading(false);
    }
  }, [checkAuth]);

  const login = async (email, password) => {
    const response = await apiClient.post('/api/v1/auth/login', { email, password });
    const data = response.data;
    localStorage.setItem('company_token', data.token);
    setUser(data.user);
    setTenant(data.tenant || null);
    return data;
  };

  const logout = () => {
    localStorage.removeItem('company_token');
    setUser(null);
    setTenant(null);
  };

  const value = { user, tenant, loading, login, logout, checkAuth };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
