import { createContext, useState, useEffect, useContext } from 'react';
import apiClient from '../api/axios';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);
    const [tenant, setTenant] = useState(null);
    const [authCheckAttempted, setAuthCheckAttempted] = useState(false);

    useEffect(() => {
        if (!authCheckAttempted) {
            checkAuth();
        }
    }, [authCheckAttempted]);

    const checkAuth = async () => {
        try {
            const response = await apiClient.get('/api/v1/auth/me');
            const data = response.data;

            // For tenant panel, we only allow TENANT or STAFF user types
            if (data.user.user_type !== 'TENANT' && data.user.user_type !== 'STAFF') {
                throw new Error('Unauthorized access to tenant panel');
            }

            setUser(data.user);
            setTenant(data.tenant || null);
        } catch (error) {
            console.error('Auth check failed:', error);
            // Token is in httpOnly cookie, no need to clear localStorage
            
            // If rate limited, don't retry immediately
            if (error.isRateLimitError) {
                console.warn('Rate limited during auth check. User will need to refresh manually.');
            }
        } finally {
            setLoading(false);
            setAuthCheckAttempted(true);
        }
    };

    const login = async (email, password) => {
        try {
            const response = await apiClient.post('/api/v1/auth/login', { email, password });
            const data = response.data;

            // Validate user type for tenant panel
            if (data.user.user_type !== 'TENANT' && data.user.user_type !== 'STAFF') {
                throw new Error('Use admin panel for this account type');
            }

            // Keep token in localStorage for backward compatibility
            if (data.token) localStorage.setItem('token', data.token);
            setUser(data.user);
            setTenant(data.tenant || null);

            return data.user;
        } catch (error) {
            throw error;
        }
    };

    // loginWithToken is used by admin impersonation — receives a pre-issued JWT
    const loginWithToken = async (token) => {
        localStorage.setItem('token', token);
        try {
            const response = await apiClient.get('/api/v1/auth/me', {
                headers: { Authorization: `Bearer ${token}` }
            });
            const data = response.data;
            if (data.user.user_type !== 'TENANT' && data.user.user_type !== 'STAFF') {
                localStorage.removeItem('token');
                throw new Error('Invalid account type for tenant panel');
            }
            setUser(data.user);
            setTenant(data.tenant || null);
            return data.user;
        } catch (error) {
            localStorage.removeItem('token');
            throw error;
        }
    };

    const logout = async () => {
        try {
            await apiClient.post('/api/v1/auth/logout');
        } catch (error) {
            // Logout fail handled
        } finally {
            // Clear token from localStorage
            localStorage.removeItem('token');
            setUser(null);
            setTenant(null);
        }
    };

    const isUserType = (...types) => {
        return types.includes(user?.user_type);
    };

    const value = {
        user,
        loading,
        tenant,
        login,
        loginWithToken,
        logout,
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
