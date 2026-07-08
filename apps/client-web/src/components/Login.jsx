import { useState, useEffect } from 'react';
import { useNavigate, Link, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';


export default function Login() {
    const { user, login, loginWithToken } = useAuth();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (user) {
            navigate('/', { replace: true });
        }
    }, [user, navigate]);

    // Admin impersonation: auto-login if ?token= is in URL
    useEffect(() => {
        const impersonateToken = searchParams.get('token');
        if (impersonateToken && !user) {
            setLoading(true);
            loginWithToken(impersonateToken)
                .then(() => navigate('/', { replace: true }))
                .catch(() => setError('فشل الدخول التلقائي. يرجى تسجيل الدخول يدوياً.'))
                .finally(() => setLoading(false));
        }
    }, [searchParams, user, loginWithToken, navigate]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        try {
            await login(email, password);
            navigate('/', { replace: true });
            return;
        } catch (err) {
            // Handle rate limiting errors specifically
            if (err.isRateLimitError) {
                setError(`تم تجاوز الحد المسموح من الطلبات. يرجى المحاولة بعد ${err.retryAfter} ثانية.`);
            } else {
                setError(err.message || 'Invalid email or password');
            }
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
            <div className="relative w-full max-w-md">
                {/* Logo/Header */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-rose-600 mb-4 shadow-xl">
                        <span className="text-white text-3xl font-bold">H</span>
                    </div>
                    <h1 className="text-4xl font-black text-white mb-2">ALHudhudAI</h1>
                    <p className="text-slate-400 text-sm font-medium">Tenant Dashboard Login</p>
                </div>

                {/* Login Card */}
                <div className="glass-card p-8 bg-slate-900 border border-slate-800 rounded-xl">
                    <form onSubmit={handleSubmit} className="space-y-6">
                        {/* Error Message */}
                        {error && (
                            <div className="bg-rose-500/10 border border-rose-500/20 rounded-lg p-4">
                                <p className="text-rose-400 text-sm font-medium">{error}</p>
                            </div>
                        )}

                        {/* Email Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Business Email
                            </label>
                            <input
                                type="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="contact@company.com"
                                required
                                disabled={loading}
                                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                            />
                        </div>

                        {/* Password Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Password
                            </label>
                            <input
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="••••••••"
                                required
                                disabled={loading}
                                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                            />
                        </div>

                        {/* Submit Button */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full bg-rose-600 hover:bg-rose-500 text-white font-bold py-3 rounded-lg transition-all disabled:opacity-50"
                        >
                            {loading ? 'Authenticating...' : 'Sign In'}
                        </button>
                    </form>

                    <div className="mt-6 pt-6 border-t border-slate-800">
                        <p className="text-xs text-slate-500 text-center">
                            Don't have an account? <Link to="/register" className="text-rose-500 font-bold hover:text-rose-400">Register Now</Link>
                        </p>
                    </div>
                </div>

                {/* Version */}
                <p className="text-center text-xs text-slate-600 mt-6">
                    ALHudhudAI v1.1.0 &copy; 2026
                </p>
            </div>
        </div>
    );
}
