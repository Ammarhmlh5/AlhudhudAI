import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Lock, Mail, ShieldCheck, ArrowRight, Loader2, Globe } from 'lucide-react';
import apiClient from '../api/axios';

export default function Login() {
    const { t, i18n } = useTranslation();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const navigate = useNavigate();

    const handleLogin = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');
        try {
            const res = await apiClient.post('/api/v1/auth/login', { email, password });
            
            // Validate if the user is a RESELLER or STAFF of a reseller
            if (res.data.user.user_type !== 'RESELLER' && res.data.user.user_type !== 'SUPER_ADMIN') {
                setError(t('login.access_denied'));
                return;
            }

            localStorage.setItem('reseller_token', res.data.token);
            navigate('/');
        } catch (err) {
            setError(err.response?.data?.error || t('login.invalid'));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-slate-950 flex items-center justify-center p-6 selection:bg-rose-500 selection:text-white relative overflow-hidden">
            {/* Background Decorations */}
            <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-rose-600/10 blur-[120px] rounded-full -mr-64 -mt-64 animate-pulse" />
            <div className="absolute bottom-0 left-0 w-[500px] h-[500px] bg-blue-600/10 blur-[120px] rounded-full -ml-64 -mb-64" />

            <div className="w-full max-w-md relative z-10">
                <div className="text-center mb-10">
                    <div className="inline-flex items-center justify-center w-20 h-20 bg-rose-600 rounded-[2rem] shadow-2xl shadow-rose-900/40 mb-6 group transition-transform hover:scale-110">
                        <ShieldCheck className="w-10 h-10 text-white" />
                    </div>
                    <h1 className="text-4xl font-black text-white tracking-tighter uppercase mb-2">
                        Hudhud<span className="text-rose-500">B2B</span>
                    </h1>
                    <p className="text-slate-500 font-bold uppercase tracking-[0.2em] text-[10px]">{t('brand.tagline')}</p>
                </div>

                <div className="absolute top-4 end-4">
                    <button
                        onClick={() => i18n.changeLanguage(i18n.language === 'ar' ? 'en' : 'ar')}
                        className="flex items-center gap-2 px-3 py-2 bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 rounded-xl transition-colors text-xs font-bold"
                    >
                        <Globe className="w-4 h-4" />
                        {i18n.language === 'ar' ? 'English' : 'عربي'}
                    </button>
                </div>

                <div className="glass-card p-10 border border-slate-800/50 relative overflow-hidden">
                    <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-rose-600 to-rose-400" />
                    
                    <form onSubmit={handleLogin} className="space-y-6">
                        {error && (
                            <div className="p-4 bg-rose-500/10 border border-rose-500/20 rounded-2xl text-rose-500 text-xs font-bold text-center animate-shake">
                                {error}
                            </div>
                        )}

                        <div className="space-y-2">
                            <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('login.email')}</label>
                            <div className="relative group">
                                <Mail className="absolute left-5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500 group-focus-within:text-rose-500 transition-colors" />
                                <input 
                                    type="email" 
                                    required
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-2xl py-4 pl-14 pr-6 text-white focus:outline-none focus:border-rose-500 transition-all font-medium text-sm"
                                    placeholder="name@company.com"
                                />
                            </div>
                        </div>

                        <div className="space-y-2">
                            <div className="flex justify-between items-center px-1">
                                <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('login.password')}</label>
                                <button type="button" className="text-[10px] font-black text-rose-500 uppercase tracking-widest hover:text-rose-400 transition-colors">{t('login.forgot')}</button>
                            </div>
                            <div className="relative group">
                                <Lock className="absolute left-5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500 group-focus-within:text-rose-500 transition-colors" />
                                <input 
                                    type="password" 
                                    required
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-2xl py-4 pl-14 pr-6 text-white focus:outline-none focus:border-rose-500 transition-all font-medium text-sm"
                                    placeholder="••••••••"
                                />
                            </div>
                        </div>

                        <button 
                            type="submit"
                            disabled={loading}
                            className="w-full bg-rose-600 hover:bg-rose-500 disabled:bg-rose-900 disabled:opacity-50 text-white py-4 rounded-2xl font-black uppercase tracking-[0.2em] text-xs transition-all shadow-xl shadow-rose-900/20 flex items-center justify-center gap-3 active:scale-95"
                        >
                            {loading ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                                <>
                                    {t('login.submit')}
                                    <ArrowRight className="w-4 h-4" />
                                </>
                            )}
                        </button>
                    </form>
                </div>

                <p className="text-center mt-8 text-slate-600 text-xs font-bold">
                    {t('brand.footer')}
                </p>
            </div>
        </div>
    );
}
