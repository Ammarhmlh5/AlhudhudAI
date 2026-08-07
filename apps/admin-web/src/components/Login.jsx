import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { LogIn, Mail, Lock, AlertCircle, Loader } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function Login() {
    const { t } = useTranslation();
    const { user, login } = useAuth();
    const navigate = useNavigate();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (user) {
            navigate('/', { replace: true });
        }
    }, [user, navigate]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        setLoading(true);

        try {
            await login(email, password);
            navigate('/', { replace: true });
        } catch (err) {
            setError(err.response?.data?.error || err.message || t('auth.invalid_credentials'));
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-4">
            {/* Background Pattern */}
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(236,72,153,0.05),transparent_50%)]" />

            <div className="relative w-full max-w-md">
                {/* Logo/Header */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-3xl bg-gradient-to-br from-rose-500 to-pink-600 mb-4 shadow-2xl shadow-rose-900/50">
                        <LogIn className="w-10 h-10 text-white" />
                    </div>
                    <h1 className="text-4xl font-black text-white mb-2">ALHudhudAI</h1>
                    <p className="text-slate-400 text-sm font-medium">{t('auth.admin_panel_login')}</p>
                </div>

                {/* Login Card */}
                <div className="bg-slate-900/80 backdrop-blur-xl border border-slate-800 rounded-3xl p-8 shadow-2xl">
                    <form onSubmit={handleSubmit} className="space-y-6">
                        {/* Error Message */}
                        {error && (
                            <div key="login-error" className="bg-rose-500/10 border border-rose-500/20 rounded-xl p-4 flex items-start gap-3 animate-in fade-in slide-in-from-top-2">
                                <AlertCircle className="w-5 h-5 text-rose-500 flex-shrink-0 mt-0.5" />
                                <div className="flex-1">
                                    <p className="text-rose-400 text-sm font-medium">{error}</p>
                                </div>
                            </div>
                        )}

                        {/* Email Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                {t('auth.email_address')}
                            </label>
                            <div className="relative">
                                <Mail className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                <input
                                    type="email"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    placeholder="email@example.com"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                                />
                            </div>
                        </div>

                        {/* Password Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                {t('auth.password')}
                            </label>
                            <div className="relative">
                                <Lock className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    placeholder="••••••••"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                                />
                            </div>
                        </div>

                        {/* Submit Button */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full bg-gradient-to-r from-rose-600 to-pink-600 hover:from-rose-500 hover:to-pink-500 text-white font-bold py-3.5 rounded-xl transition-all shadow-lg shadow-rose-900/30 hover:shadow-rose-900/50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                        >
                            {loading ? (
                                <>
                                    <Loader className="w-5 h-5 animate-spin" />
                                    <span>{t('auth.signing_in')}</span>
                                </>
                            ) : (
                                <>
                                    <LogIn className="w-5 h-5" />
                                    <span>{t('auth.sign_in')}</span>
                                </>
                            )}
                        </button>
                    </form>
                </div>

                {/* Version */}
                <p className="text-center text-xs text-slate-600 mt-6">
                    {t('auth.version', { brand: 'ALHudhudAI', version: 'v2.0.0' })}
                </p>
            </div>
        </div>
    );
}
