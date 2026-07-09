import React, { useState, useEffect } from 'react';
import { Building, ShieldCheck, Wallet, Bell, RefreshCcw, Copy, CreditCard, AlertCircle, Plus } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import apiClient from '../api/axios';

const SettingsManager = () => {
    const [activeSection, setActiveSection] = useState('profile');
    const { user, tenant } = useAuth();
    const [paymentMethods, setPaymentMethods] = useState([]);
    const [loadingBilling, setLoadingBilling] = useState(false);

    // Change Password State
    const [passData, setPassData] = useState({ oldPassword: '', newPassword: '' });
    const [passStatus, setPassStatus] = useState(null);
    const [copied, setCopied] = useState(false);

    // Regenerate API Key State
    const [showRegenModal, setShowRegenModal] = useState(false);
    const [regenPassword, setRegenPassword] = useState('');
    const [regenStatus, setRegenStatus] = useState(null);
    const [currentApiKey, setCurrentApiKey] = useState('');

    useEffect(() => {
        if (tenant?.api_key_accounting) {
            setCurrentApiKey(tenant.api_key_accounting);
        }
    }, [tenant]);

    const handleCopy = (text) => {
        if (!text) return;
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const fetchPaymentMethods = async () => {
        setLoadingBilling(true);
        try {
            const response = await apiClient.get('/api/v1/finance/payment-methods');
            setPaymentMethods(response.data.payment_methods || []);
        } catch (error) {
            console.error('Failed to fetch payment methods', error);
        } finally {
            setLoadingBilling(false);
        }
    };

    useEffect(() => {
        if (activeSection === 'billing') {
            fetchPaymentMethods();
        }
    }, [activeSection]);

    const handleChangePassword = async () => {
        try {
            setPassStatus('loading');
            const response = await apiClient.post('/api/v1/auth/change-password', {
                old_password: passData.oldPassword,
                new_password: passData.newPassword
            });

            setPassStatus('success');
            setPassData({ oldPassword: '', newPassword: '' });
        } catch (error) {
            setPassStatus('Error connecting to server');
        }
    };

    const handleRegenerateKey = async () => {
        if (!regenPassword) return;
        setRegenStatus('loading');
        try {
            const response = await apiClient.post('/api/v1/settings/api-key/regenerate', {
                password: regenPassword
            });
            setCurrentApiKey(response.data.api_key);
            setRegenStatus('success');
            setTimeout(() => {
                setShowRegenModal(false);
                setRegenPassword('');
                setRegenStatus(null);
            }, 2000);
        } catch (error) {
            setRegenStatus(error.response?.data?.error || 'حدث خطأ غير معروف');
        }
    };

    return (
        <div className="animate-in fade-in slide-in-from-bottom-8 duration-500">
            <h2 className="text-3xl font-bold mb-8">Advanced Settings</h2>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
                {/* Sidebar Navigation */}
                <div className="md:col-span-1 space-y-2">
                    {[
                        { id: 'profile', label: 'Company Profile', icon: Building },
                        { id: 'security', label: 'Security & API', icon: ShieldCheck },
                        { id: 'billing', label: 'Financial Hub', icon: Wallet },
                        { id: 'notifications', label: 'Alerts', icon: Bell }
                    ].map(item => (
                        <button
                            key={item.id}
                            onClick={() => setActiveSection(item.id)}
                            className={`w-full flex items-center gap-3 px-4 py-3 rounded-2xl transition-all ${activeSection === item.id
                                ? 'bg-rose-600 text-white font-bold shadow-lg'
                                : 'text-slate-400 hover:bg-slate-900'
                                }`}
                        >
                            <item.icon className="w-4 h-4" />
                            <span className="text-sm">{item.label}</span>
                        </button>
                    ))}
                </div>

                {/* Content Area */}
                <div className="md:col-span-3 space-y-6">
                    {activeSection === 'profile' && (
                        <div className="glass-card p-8 space-y-6 animate-in fade-in slide-in-from-right-4">
                            <h3 className="text-xl font-bold">Company Profile</h3>
                            <div className="grid grid-cols-2 gap-6">
                                <div className="space-y-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase">Business Name</label>
                                    <input
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 text-slate-300"
                                        defaultValue={tenant?.business_name || 'Loading...'}
                                        readOnly
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase">Contact Email</label>
                                    <input
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 text-slate-300"
                                        defaultValue={tenant?.email || user?.email}
                                        readOnly
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase">Serial Number</label>
                                    <input
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 font-mono text-amber-500"
                                        defaultValue={tenant?.company_serial || 'N/A'}
                                        readOnly
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase">Subscription Plan</label>
                                    <input
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 text-emerald-400 font-bold"
                                        defaultValue={tenant?.subscription_plan || 'Basic Plan'}
                                        readOnly
                                    />
                                </div>
                            </div>
                            <div className="text-xs text-slate-500 italic mt-4">
                                * To update company details, please contact support.
                            </div>
                        </div>
                    )}

                    {activeSection === 'security' && (
                        <div className="space-y-6 animate-in fade-in slide-in-from-right-4">
                            <div className="glass-card p-8 space-y-6">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h3 className="text-xl font-bold">Accounting System API Key</h3>
                                        <p className="text-slate-500 text-sm">Use this key to link your ERP/Accounting software.</p>
                                    </div>
                                    <button 
                                        onClick={() => setShowRegenModal(true)}
                                        className="flex items-center gap-2 text-rose-500 text-sm font-bold bg-rose-500/10 px-4 py-2 rounded-xl border border-rose-500/20 hover:bg-rose-500/20 transition-all shadow-[0_0_15px_rgba(244,63,94,0.3)]"
                                    >
                                        <RefreshCcw className="w-4 h-4" /> Regenerate
                                    </button>
                                </div>
                                <div className="bg-slate-950 p-4 rounded-2xl flex items-center justify-between border border-slate-800">
                                    <code className="text-rose-400 font-mono text-sm break-all select-all">
                                        {currentApiKey || 'hh_live_xxxxxxxxxxxxxxxx'}
                                    </code>
                                    <button
                                        className="p-2 hover:bg-slate-800 rounded-lg group relative cursor-pointer"
                                        onClick={() => handleCopy(currentApiKey)}
                                        title="نسخ المفتاح"
                                    >
                                        <Copy className="w-5 h-5 text-slate-400 group-hover:text-emerald-500 transition-colors" />
                                        {copied && (
                                            <span className="absolute -top-8 left-1/2 -translate-x-1/2 bg-emerald-500 text-white text-xs py-1 px-2 rounded font-bold whitespace-nowrap shadow-lg">
                                                تم النسخ!
                                            </span>
                                        )}
                                    </button>
                                </div>
                            </div>

                            {/* Regenerate API Key Modal */}
                            {showRegenModal && (
                                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
                                    <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 max-w-md w-full shadow-2xl animate-in zoom-in-95">
                                        <div className="flex items-center gap-3 mb-4 text-amber-500">
                                            <AlertCircle className="w-6 h-6" />
                                            <h3 className="text-xl font-bold">تجديد مفتاح الربط (API Key)</h3>
                                        </div>
                                        <p className="text-slate-400 text-sm mb-6 leading-relaxed">
                                            تحذير: تجديد المفتاح سيؤدي إلى إيقاف عمل المفتاح القديم فوراً. جميع الأنظمة المرتبطة يجب تحديثها بالمفتاح الجديد لتستمر في العمل. يرجى إدخال كلمة مرور حسابك للتأكيد.
                                        </p>
                                        <input
                                            type="password"
                                            placeholder="أدخل كلمة المرور الخاصة بك..."
                                            className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 mb-4 focus:ring-1 ring-amber-500 text-white"
                                            value={regenPassword}
                                            onChange={(e) => setRegenPassword(e.target.value)}
                                        />
                                        
                                        {regenStatus && (
                                            <div className={`text-sm font-bold mb-4 ${regenStatus === 'success' ? 'text-emerald-500' : 'text-rose-500'}`}>
                                                {regenStatus === 'success' ? 'تم التجديد بنجاح!' : regenStatus === 'loading' ? 'جاري التحقق...' : regenStatus}
                                            </div>
                                        )}

                                        <div className="flex justify-end gap-3 mt-2">
                                            <button
                                                onClick={() => {
                                                    setShowRegenModal(false);
                                                    setRegenPassword('');
                                                    setRegenStatus(null);
                                                }}
                                                className="px-4 py-2 text-slate-400 hover:text-white transition-colors"
                                            >
                                                إلغاء
                                            </button>
                                            <button
                                                onClick={handleRegenerateKey}
                                                disabled={!regenPassword || regenStatus === 'loading'}
                                                className="bg-amber-500 hover:bg-amber-600 text-black font-bold px-6 py-2 rounded-lg transition-colors disabled:opacity-50"
                                            >
                                                تأكيد التجديد
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            <div className="glass-card p-8 space-y-6">
                                <h3 className="text-xl font-bold">Change Password</h3>
                                <div className="space-y-4 max-w-sm">
                                    <input
                                        type="password"
                                        placeholder="Current Password"
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 focus:ring-1 ring-rose-500"
                                        value={passData.oldPassword}
                                        onChange={(e) => setPassData({ ...passData, oldPassword: e.target.value })}
                                    />
                                    <input
                                        type="password"
                                        placeholder="New Password (min 8 chars)"
                                        className="w-full bg-slate-800 border-none rounded-xl py-3 px-4 focus:ring-1 ring-rose-500"
                                        value={passData.newPassword}
                                        onChange={(e) => setPassData({ ...passData, newPassword: e.target.value })}
                                    />

                                    {passStatus && (
                                        <div className={`text-sm font-bold ${passStatus === 'success' ? 'text-emerald-500' : 'text-rose-500'}`}>
                                            {passStatus === 'success' ? 'Password changed successfully!' : passStatus === 'loading' ? 'Updating...' : passStatus}
                                        </div>
                                    )}

                                    <button
                                        onClick={handleChangePassword}
                                        disabled={passStatus === 'loading' || !passData.oldPassword || !passData.newPassword}
                                        className="bg-slate-100 text-slate-950 font-bold py-3 px-8 rounded-xl hover:bg-white transition-all disabled:opacity-50"
                                    >
                                        Save Changes
                                    </button>
                                </div>
                            </div>
                        </div>
                    )}

                    {activeSection === 'billing' && (
                        <div className="space-y-6 animate-in fade-in slide-in-from-right-4">
                            <div className="glass-card p-8 space-y-6">
                                <h3 className="text-xl font-bold">Payment Methods</h3>
                                <div className="space-y-4">
                                    {loadingBilling ? (
                                        <div className="text-center py-4 text-slate-500">Loading payment methods...</div>
                                    ) : paymentMethods.map(method => (
                                        <div key={method.id} className="flex items-center justify-between p-4 bg-slate-800/30 rounded-2xl border border-slate-700/50">
                                            <div className="flex items-center gap-4">
                                                <div className="bg-slate-700 p-3 rounded-xl"><CreditCard className="w-6 h-6 text-amber-500" /></div>
                                                <div>
                                                    <p className="font-bold">{method.card_brand} Ending in {method.last4}</p>
                                                    <p className="text-xs text-slate-500 uppercase">Expires {method.exp_month}/{method.exp_year}</p>
                                                </div>
                                            </div>
                                            {method.is_primary && (
                                                <span className="text-[10px] font-bold text-emerald-400 bg-emerald-500/10 px-2 py-1 rounded-md uppercase">Primary</span>
                                            )}
                                        </div>
                                    ))}

                                    {paymentMethods.length === 0 && !loadingBilling && (
                                        <div className="text-center py-10 bg-slate-900/20 rounded-3xl border border-slate-800/50">
                                            <CreditCard className="w-12 h-12 text-slate-700 mx-auto mb-4" />
                                            <p className="text-slate-500 mb-2">No payment methods stored.</p>
                                        </div>
                                    )}
                                    <button className="w-full border-2 border-dashed border-slate-800 py-6 rounded-3xl text-slate-500 font-bold hover:border-rose-500/30 hover:text-rose-500 transition-all flex items-center justify-center gap-2">
                                        <Plus className="w-4 h-4" /> Add Payment Service
                                    </button>
                                </div>
                            </div>

                            <div className="glass-card p-6 border-amber-500/20 bg-amber-500/5 flex items-center justify-between">
                                <div className="flex items-center gap-4">
                                    <AlertCircle className="w-6 h-6 text-amber-500" />
                                    <div>
                                        <p className="text-sm font-bold text-amber-200">Auto-Recharge Active</p>
                                        <p className="text-xs text-amber-500/80">Recharge $50 when balance falls below $10.</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}

                    {activeSection === 'notifications' && (
                        <div className="glass-card p-8 text-center text-slate-500 italic">
                            Notification settings are managed by your administrator.
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default SettingsManager;
