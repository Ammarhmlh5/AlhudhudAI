import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Wallet, Package, Zap, Star, CreditCard, Clock } from 'lucide-react';

const FinanceDashboard = () => {
    const { tenant } = useAuth();
    const [activeTab, setActiveTab] = useState('balance');

    const staticPackages = [
        { id: 'basic', name: 'الباقة الأساسية', price: 50, quota: 1000, popular: false },
        { id: 'pro', name: 'باقة المحترفين', price: 150, quota: 5000, popular: true },
        { id: 'enterprise', name: 'باقة الشركات', price: 500, quota: 25000, popular: false },
    ];

    return (
        <div className="animate-in fade-in slide-in-from-right-8 duration-500 space-y-6">
            {/* Header */}
            <div>
                <h2 className="text-3xl font-bold">اشتراكاتي</h2>
                <p className="text-slate-400 text-sm mt-1">إدارة رصيدك، باقاتك، وسجل المعاملات المالية.</p>
            </div>

            {/* Tab Switcher */}
            <div className="flex gap-2 p-1 bg-slate-900 border border-slate-800 rounded-xl w-fit">
                <button
                    onClick={() => setActiveTab('balance')}
                    className={`px-5 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'balance' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-white'}`}
                >
                    <Wallet className="w-4 h-4 inline-block ml-2" />
                    رصيدي
                </button>
                <button
                    onClick={() => setActiveTab('subscriptions')}
                    className={`px-5 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'subscriptions' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-white'}`}
                >
                    <Package className="w-4 h-4 inline-block ml-2" />
                    الباقات المتاحة
                </button>
            </div>

            {/* ── Balance View ── */}
            {activeTab === 'balance' && (
                <div className="space-y-6">
                    {/* Info banner from tenant context */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        {/* Cash Balance */}
                        <div className="bg-slate-900 border border-emerald-500/20 rounded-2xl p-6 relative overflow-hidden">
                            <div className="absolute top-0 right-0 p-4 opacity-10">
                                <Wallet className="w-20 h-20 text-emerald-500" />
                            </div>
                            <p className="text-slate-400 text-sm mb-1 flex items-center gap-1">
                                <Wallet className="w-4 h-4 text-emerald-500" /> الرصيد المالي
                            </p>
                            <h3 className="text-4xl font-black text-white">
                                {Number(tenant?.balance || 0).toFixed(2)}
                                <span className="text-lg text-slate-500 mr-1">ر.س</span>
                            </h3>
                        </div>

                        {/* Message Quota */}
                        <div className="bg-slate-900 border border-blue-500/20 rounded-2xl p-6 relative overflow-hidden">
                            <div className="absolute top-0 right-0 p-4 opacity-10">
                                <Zap className="w-20 h-20 text-blue-500" />
                            </div>
                            <p className="text-slate-400 text-sm mb-1 flex items-center gap-1">
                                <Zap className="w-4 h-4 text-blue-500" /> رصيد الرسائل
                            </p>
                            <h3 className="text-4xl font-black text-white">
                                {Number(tenant?.message_limit || 0).toLocaleString()}
                                <span className="text-lg text-slate-500 mr-1">رسالة</span>
                            </h3>
                        </div>

                        {/* Subscription Plan */}
                        <div className="bg-slate-900 border border-rose-500/20 rounded-2xl p-6 relative overflow-hidden">
                            <div className="absolute top-0 right-0 p-4 opacity-10">
                                <Star className="w-20 h-20 text-rose-500" />
                            </div>
                            <p className="text-slate-400 text-sm mb-1 flex items-center gap-1">
                                <Star className="w-4 h-4 text-rose-500" /> الباقة الحالية
                            </p>
                            <h3 className="text-2xl font-black text-white mt-2">
                                {tenant?.subscription_plan || 'الباقة الأساسية'}
                            </h3>
                            <span className={`mt-2 inline-block text-xs font-bold px-2 py-0.5 rounded ${tenant?.status === 'ACTIVE' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-amber-500/10 text-amber-400'}`}>
                                {tenant?.status === 'ACTIVE' ? 'نشط' : tenant?.status || 'غير محدد'}
                            </span>
                        </div>
                    </div>

                    {/* Account Info */}
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
                        <div className="p-5 border-b border-slate-800 bg-slate-900/50 flex items-center gap-2">
                            <Clock className="w-5 h-5 text-slate-400" />
                            <h3 className="font-bold">معلومات الحساب</h3>
                        </div>
                        <div className="divide-y divide-slate-800">
                            <div className="flex items-center justify-between px-6 py-4">
                                <span className="text-slate-400 text-sm">اسم الشركة</span>
                                <span className="font-bold text-white">{tenant?.business_name || '—'}</span>
                            </div>
                            <div className="flex items-center justify-between px-6 py-4">
                                <span className="text-slate-400 text-sm">المسؤول</span>
                                <span className="font-bold text-white">{tenant?.contact_name || '—'}</span>
                            </div>
                            <div className="flex items-center justify-between px-6 py-4">
                                <span className="text-slate-400 text-sm">البريد الإلكتروني</span>
                                <span className="font-mono text-slate-300">{tenant?.email || '—'}</span>
                            </div>
                            <div className="flex items-center justify-between px-6 py-4">
                                <span className="text-slate-400 text-sm">الرقم التسلسلي للشركة</span>
                                <span className="font-mono text-rose-400 text-xs">{tenant?.company_serial || '—'}</span>
                            </div>
                        </div>
                    </div>

                    {/* Contact Support */}
                    <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-4 flex items-start gap-4">
                        <div className="p-3 bg-emerald-500/10 rounded-lg shrink-0">
                            <CreditCard className="w-6 h-6 text-emerald-400" />
                        </div>
                        <div>
                            <h4 className="text-white font-bold text-sm">شحن الرصيد أو ترقية الباقة</h4>
                            <p className="text-slate-400 text-xs mt-1 max-w-2xl leading-relaxed">
                                لشحن رصيدك أو الترقية إلى باقة أعلى، يرجى التواصل مع فريق الدعم الفني مباشرة. سيتم معالجة طلبك خلال أقل من 24 ساعة.
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {/* ── Packages View ── */}
            {activeTab === 'subscriptions' && (
                <div className="space-y-4">
                    <p className="text-slate-500 text-sm">اختر الباقة المناسبة لاحتياجاتك. للترقية، تواصل مع الدعم الفني.</p>
                    {staticPackages.map(pkg => (
                        <div
                            key={pkg.id}
                            className={`bg-slate-900 border rounded-2xl p-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 transition-all hover:border-rose-500/30 ${pkg.popular ? 'border-rose-500/50 shadow-lg shadow-rose-900/10' : 'border-slate-800'}`}
                        >
                            <div className="flex items-center gap-4">
                                <div className={`w-12 h-12 rounded-2xl flex items-center justify-center ${pkg.popular ? 'bg-rose-500/20 text-rose-500' : 'bg-slate-800 text-slate-400'}`}>
                                    <Package className="w-6 h-6" />
                                </div>
                                <div>
                                    <div className="flex items-center gap-2">
                                        <h4 className="text-lg font-bold text-white">{pkg.name}</h4>
                                        {pkg.popular && (
                                            <span className="bg-rose-500 text-white text-[10px] px-2 py-0.5 rounded font-bold">
                                                الأكثر طلباً
                                            </span>
                                        )}
                                        {tenant?.subscription_plan === pkg.name && (
                                            <span className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] px-2 py-0.5 rounded font-bold">
                                                باقتك الحالية
                                            </span>
                                        )}
                                    </div>
                                    <p className="text-slate-400 text-sm mt-1">
                                        <Zap className="w-3 h-3 inline ml-1 text-blue-400" />
                                        {pkg.quota.toLocaleString()} رسالة / شهرياً
                                    </p>
                                </div>
                            </div>
                            <div className="flex items-center gap-6">
                                <div>
                                    <p className="text-2xl font-black text-rose-400">{pkg.price} <span className="text-sm text-slate-500">ر.س</span></p>
                                    <p className="text-[10px] text-slate-500">شهرياً</p>
                                </div>
                                <button
                                    className={`px-5 py-2.5 rounded-xl text-sm font-bold transition-all ${tenant?.subscription_plan === pkg.name ? 'bg-slate-800 text-slate-500 cursor-not-allowed' : pkg.popular ? 'bg-rose-500 hover:bg-rose-600 text-white' : 'bg-slate-800 hover:bg-slate-700 text-slate-300'}`}
                                    disabled={tenant?.subscription_plan === pkg.name}
                                >
                                    {tenant?.subscription_plan === pkg.name ? 'باقتك الحالية' : 'طلب الترقية'}
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default FinanceDashboard;
