import { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Link, useNavigate } from 'react-router-dom';
import { UserPlus, Mail, Lock, AlertCircle, Loader, Smartphone, Globe } from 'lucide-react';
import { API_BASE_URL } from '../config';

export default function Register() {
    const { login } = useAuth(); // We'll assume auto-login or just redirect to login
    const navigate = useNavigate();

    const [formData, setFormData] = useState({
        email: '',
        phone: '',
        country_code: '+966', // Default
        password: '',
        confirm_password: ''
    });

    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleChange = (e) => {
        setFormData({
            ...formData,
            [e.target.name]: e.target.value
        });
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');

        if (formData.password !== formData.confirm_password) {
            setError("Passwords do not match");
            return;
        }

        setLoading(true);

        try {
            const response = await fetch(`${API_BASE_URL}/api/v1/tenants/register`, {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(formData),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Registration failed');
            }

            // Auto-login or redirect
            // If the response contains a token (which we implemented it to), we can use the login function logic 
            // from AuthContext if we exposed a "setSession" method, but 'login' method in context does a fetch.
            // For now, let's just redirect to login with a success message or if we want auto-login we need to handle tokens.
            // Our backend returns tokens. Let's try to manually set them or just redirect to login for simplicity/safety first.

            // Actually, best UX is auto-login. But AuthContext doesn't export a "setUserData" method easily.
            // Let's redirect to Login for now to avoid AuthContext refactoring, or just call login() with the credentials?
            // Calling login(email, password) is easiest!

            await login(formData.email, formData.password);
            navigate('/');

        } catch (err) {
            setError(err.message || 'Registration failed');
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-4">
            {/* Background Pattern */}
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(14,165,233,0.05),transparent_50%)]" />

            <div className="relative w-full max-w-md">
                {/* Logo/Header */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-20 h-20 rounded-3xl bg-gradient-to-br from-blue-500 to-cyan-600 mb-4 shadow-2xl shadow-blue-900/50">
                        <UserPlus className="w-10 h-10 text-white" />
                    </div>
                    <h1 className="text-4xl font-black text-white mb-2">ALHudhudAI</h1>
                    <p className="text-slate-400 text-sm font-medium">Create New Tenant Account</p>
                </div>

                {/* Register Card */}
                <div className="bg-slate-900/80 backdrop-blur-xl border border-slate-800 rounded-3xl p-8 shadow-2xl">
                    <form onSubmit={handleSubmit} className="space-y-4">
                        {/* Error Message */}
                        {error && (
                            <div className="bg-rose-500/10 border border-rose-500/20 rounded-xl p-4 flex items-start gap-3 animate-in fade-in slide-in-from-top-2">
                                <AlertCircle className="w-5 h-5 text-rose-500 flex-shrink-0 mt-0.5" />
                                <div>
                                    <p className="text-rose-400 text-sm font-medium">{error}</p>
                                </div>
                            </div>
                        )}

                        {/* Email Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Email Address
                            </label>
                            <div className="relative">
                                <Mail className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                <input
                                    type="email"
                                    name="email"
                                    value={formData.email}
                                    onChange={handleChange}
                                    placeholder="name@company.com"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all disabled:opacity-50"
                                />
                            </div>
                        </div>

                        {/* Phone & Country Code */}
                        <div className="grid grid-cols-3 gap-4">
                            <div className="col-span-1">
                                <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                    Code
                                </label>
                                <div className="relative">
                                    <Globe className="w-5 h-5 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
                                    <input
                                        type="text"
                                        name="country_code"
                                        value={formData.country_code}
                                        onChange={handleChange}
                                        placeholder="+966"
                                        required
                                        disabled={loading}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-10 pr-2 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all disabled:opacity-50"
                                    />
                                </div>
                            </div>
                            <div className="col-span-2">
                                <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                    Phone Number
                                </label>
                                <div className="relative">
                                    <Smartphone className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                    <input
                                        type="tel"
                                        name="phone"
                                        value={formData.phone}
                                        onChange={handleChange}
                                        placeholder="55xxxxxxx"
                                        required
                                        disabled={loading}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all disabled:opacity-50"
                                    />
                                </div>
                            </div>
                        </div>

                        {/* Password Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Password
                            </label>
                            <div className="relative">
                                <Lock className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                <input
                                    type="password"
                                    name="password"
                                    value={formData.password}
                                    onChange={handleChange}
                                    placeholder="••••••••"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all disabled:opacity-50"
                                />
                            </div>
                        </div>

                        {/* Confirm Password */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Confirm Password
                            </label>
                            <div className="relative">
                                <Lock className="w-5 h-5 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
                                <input
                                    type="password"
                                    name="confirm_password"
                                    value={formData.confirm_password}
                                    onChange={handleChange}
                                    placeholder="••••••••"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-12 pr-4 py-3.5 text-white placeholder-slate-600 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all disabled:opacity-50"
                                />
                            </div>
                        </div>

                        {/* Submit Button */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-500 hover:to-cyan-500 text-white font-bold py-3.5 rounded-xl transition-all shadow-lg shadow-blue-900/30 hover:shadow-blue-900/50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 mt-2"
                        >
                            {loading ? (
                                <>
                                    <Loader className="w-5 h-5 animate-spin" />
                                    Creating Account...
                                </>
                            ) : (
                                <>
                                    <UserPlus className="w-5 h-5" />
                                    Register Now
                                </>
                            )}
                        </button>
                    </form>

                    {/* Footer Info */}
                    <div className="mt-6 pt-6 border-t border-slate-800 text-center">
                        <p className="text-sm text-slate-500">
                            Already have an account?{' '}
                            <Link to="/login" className="text-blue-400 hover:text-blue-300 font-bold transition-colors">
                                Sign In
                            </Link>
                        </p>
                    </div>
                </div>

                {/* Version */}
                <p className="text-center text-xs text-slate-600 mt-6">
                    ALHudhudAI Admin Panel v2.0.0
                </p>
            </div>
        </div>
    );
}
