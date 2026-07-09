import { useState, useEffect } from 'react';
import { ADMIN_API_URL } from '../config';
import { useAuth } from '../contexts/AuthContext';
import { Link, useNavigate } from 'react-router-dom';


export default function Register() {
    const { user, loginWithToken } = useAuth();
    const navigate = useNavigate();

    useEffect(() => {
        if (user) {
            navigate('/', { replace: true });
        }
    }, [user, navigate]);

    const [formData, setFormData] = useState({
        email: '',
        phone: '',
        country_code: '+966',
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
            const response = await fetch(`${ADMIN_API_URL}/api/v1/tenants/register`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(formData),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Registration failed');
            }

            // Auto-login using the token from registration response
            await loginWithToken(data.token);
            navigate('/');

        } catch (err) {
            setError(err.message || 'Registration failed');
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
                    <p className="text-slate-400 text-sm font-medium">Create Business Account</p>
                </div>

                {/* Register Card */}
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 shadow-lg">
                    <form onSubmit={handleSubmit} className="space-y-4">
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
                                name="email"
                                value={formData.email}
                                onChange={handleChange}
                                placeholder="contact@company.com"
                                required
                                disabled={loading}
                                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                            />
                        </div>

                        {/* Phone & Country Code */}
                        <div className="grid grid-cols-3 gap-4">
                            <div className="col-span-1">
                                <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                    Code
                                </label>
                                <input
                                    type="text"
                                    name="country_code"
                                    value={formData.country_code}
                                    onChange={handleChange}
                                    placeholder="+966"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2 py-3 text-center text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                                />
                            </div>
                            <div className="col-span-2">
                                <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                    Phone Number
                                </label>
                                <input
                                    type="tel"
                                    name="phone"
                                    value={formData.phone}
                                    onChange={handleChange}
                                    placeholder="500000000"
                                    required
                                    disabled={loading}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                                />
                            </div>
                        </div>

                        {/* Password Input */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Password
                            </label>
                            <input
                                type="password"
                                name="password"
                                value={formData.password}
                                onChange={handleChange}
                                placeholder="••••••••"
                                required
                                disabled={loading}
                                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-rose-500 transition-all disabled:opacity-50"
                            />
                        </div>

                        {/* Confirm Password */}
                        <div>
                            <label className="text-xs font-bold text-slate-400 uppercase tracking-wider block mb-2">
                                Confirm Password
                            </label>
                            <input
                                type="password"
                                name="confirm_password"
                                value={formData.confirm_password}
                                onChange={handleChange}
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
                            className="w-full bg-rose-600 hover:bg-rose-700 text-white font-bold py-3 rounded-lg transition-all disabled:opacity-50 flex items-center justify-center gap-2 mt-2"
                        >
                            {loading ? 'Creating Account...' : 'Register Now'}
                        </button>
                    </form>

                    {/* Footer Info */}
                    <div className="mt-6 pt-6 border-t border-slate-800 text-center">
                        <p className="text-sm text-slate-500">
                            Already have an account?{' '}
                            <Link to="/login" className="text-rose-400 hover:text-rose-300 font-bold transition-colors">
                                Sign In
                            </Link>
                        </p>
                    </div>
                </div>

                {/* Version */}
                <p className="text-center text-xs text-slate-600 mt-6">
                    ALHudhudAI Portal v1.1.0 &copy; 2026
                </p>
            </div>
        </div>
    );
}
