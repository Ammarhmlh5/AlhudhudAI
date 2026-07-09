import { createContext, useState, useEffect, useContext } from 'react';
import { API_BASE_URL } from '../config';
import apiClient from '../api/axios';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);
    const [permissions, setPermissions] = useState([]);
    const [tenant, setTenant] = useState(null);
    const [staff, setStaff] = useState(null);

    useEffect(() => {
        checkAuth();
    }, []);

    const checkAuth = async () => {
        try {
            const response = await apiClient.get('/api/v1/auth/me');
            const data = response.data;
            setUser(data.user);
            setPermissions(data.permissions || []);
            setTenant(data.tenant || null);
            setStaff(data.staff || null);
        } catch (error) {
            console.error('Auth check failed:', error);
            // Don't clear user on initial failure if it was just a transient error, 
            // but for now we'll just log it.
        } finally {
            setLoading(false);
        }
    };

    const login = async (email, password) => {
        try {
            const response = await apiClient.post('/api/v1/auth/login', { email, password });
            const data = response.data;

            // Keep token in localStorage for backward compatibility with manual fetch calls
            if (data.token) localStorage.setItem('token', data.token);
            setUser(data.user);
            setPermissions(data.permissions || []);
            setTenant(data.tenant || null);
            setStaff(data.staff || null);

            return data.user;
        } catch (error) {
            console.error('Login error:', error);
            throw error;
        }
    };

    const logout = async () => {
        try {
            await apiClient.post('/api/v1/auth/logout');
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            // Clear token from localStorage
            localStorage.removeItem('token');
            setUser(null);
            setPermissions([]);
            setTenant(null);
            setStaff(null);
        }
    };

    const hasPermission = (permission) => {
        // Super admin has all permissions
        if (user?.user_type === 'SUPER_ADMIN') return true;

        // Check if permission exists in user's permissions
        if (permissions.includes('*')) return true;
        return permissions.includes(permission);
    };

    const isUserType = (...types) => {
        return types.includes(user?.user_type);
    };

    const value = {
        user,
        loading,
        permissions,
        tenant,
        staff,
        login,
        logout,
        hasPermission,
        isUserType,
    };

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
