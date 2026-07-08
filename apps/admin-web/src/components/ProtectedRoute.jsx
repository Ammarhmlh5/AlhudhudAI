import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Loader } from 'lucide-react';

export default function ProtectedRoute({ children, allowedTypes = [] }) {
    const { user, loading } = useAuth();

    if (loading) {
        return (
            <div className="min-h-screen bg-slate-950 flex items-center justify-center">
                <div className="text-center">
                    <Loader className="w-12 h-12 text-rose-500 animate-spin mx-auto mb-4" />
                    <p className="text-slate-400 font-medium">Loading...</p>
                </div>
            </div>
        );
    }

    if (!user) {
        return <Navigate to="/login" replace />;
    }

    if (allowedTypes.length > 0 && !allowedTypes.includes(user.user_type)) {
        return (
            <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
                <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-md text-center">
                    <div className="w-16 h-16 rounded-2xl bg-rose-500/10 flex items-center justify-center mx-auto mb-4">
                        <svg className="w-8 h-8 text-rose-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                    </div>
                    <h2 className="text-2xl font-bold text-white mb-2">Access Denied</h2>
                    <p className="text-slate-400 mb-6">You don't have permission to access this page.</p>
                    <button
                        onClick={async () => {
                            const { logout } = require('../contexts/AuthContext');
                            // We can use the logout function from useAuth hook
                        }}
                        className="hidden"
                    >
                        Go Back
                    </button>
                    {/* Proper way to handle logout in React component */}
                    <LogoutButton />
                </div>
            </div>
        );
    }

    return children;
}

// Extract logout to a small component to easily use the hook
function LogoutButton() {
    const { logout } = useAuth();
    return (
        <button
            onClick={async () => {
                await logout();
                window.location.href = '/login';
            }}
            className="px-6 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold transition-colors"
        >
            Sign Out & Switch Account
        </button>
    );
}
