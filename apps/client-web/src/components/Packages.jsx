import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { Package, Check, Zap, TrendingUp, Star, Loader, AlertCircle } from 'lucide-react';

const Packages = () => {
    const [packages, setPackages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [subscribing, setSubscribing] = useState(null);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchPackages();
    }, []);

    const fetchPackages = async () => {
        try {
            const response = await apiClient.get('/api/v1/client/packages');
            setPackages(response.data.packages || []);
        } catch (err) {
            console.error('Error fetching packages:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const handleSubscribe = async (packageId) => {
        if (!window.confirm('هل أنت متأكد من الاشتراك في هذه الباقة؟')) {
            return;
        }

        setSubscribing(packageId);
        try {
            const response = await apiClient.post('/api/v1/client/subscribe', { package_id: packageId });

            alert('تم الاشتراك بنجاح!');
            // Optionally refresh the page or update state
            window.location.reload();
        } catch (err) {
            console.error('Error subscribing:', err);
            alert(err.message || 'فشل الاشتراك. الرجاء المحاولة مرة أخرى.');
        } finally {
            setSubscribing(null);
        }
    };

    const getPackageIcon = (name) => {
        if (name.toLowerCase().includes('enterprise')) return Star;
        if (name.toLowerCase().includes('business')) return TrendingUp;
        if (name.toLowerCase().includes('professional')) return Zap;
        return Package;
    };

    const getPackageColor = (index) => {
        const colors = [
            'from-blue-500 to-cyan-500',
            'from-purple-500 to-pink-500',
            'from-orange-500 to-rose-500',
            'from-emerald-500 to-teal-500',
            'from-amber-500 to-yellow-500'
        ];
        return colors[index % colors.length];
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-center">
                    <Loader className="w-12 h-12 text-rose-500 animate-spin mx-auto mb-4" />
                    <p className="text-slate-400">جاري تحميل الباقات...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="text-center">
                    <AlertCircle className="w-12 h-12 text-rose-500 mx-auto mb-4" />
                    <p className="text-slate-400">فشل تحميل الباقات</p>
                    <button
                        onClick={fetchPackages}
                        className="mt-4 px-6 py-2 bg-rose-500 hover:bg-rose-600 rounded-xl text-white font-bold transition-colors"
                    >
                        إعادة المحاولة
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
            <div className="mb-8 text-center">
                <h2 className="text-4xl font-bold mb-3">اختر الباقة المناسبة لك</h2>
                <p className="text-slate-400 text-lg">باقات شهرية مرنة تناسب جميع احتياجاتك</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {packages.map((pkg, index) => {
                    const Icon = getPackageIcon(pkg.name);
                    const isUnlimited = pkg.messages_included === null || pkg.messages_included === 0;
                    const isPopular = pkg.name.toLowerCase().includes('professional') || pkg.name.toLowerCase().includes('business');

                    return (
                        <div
                            key={pkg.id}
                            className={`bg-slate-900 border border-slate-800 p-6 rounded-xl hover:border-rose-500/30 transition-all relative overflow-hidden group ${isPopular ? 'border-rose-500/50' : ''
                                }`}
                        >
                            {isPopular && (
                                <div className="absolute top-0 right-0 bg-rose-600 text-white text-xs font-bold px-4 py-1 rounded-bl-lg">
                                    الأكثر شعبية
                                </div>
                            )}

                            {/* Package Icon */}
                            <div className={`w-16 h-16 rounded-xl bg-slate-800 flex items-center justify-center mb-6 group-hover:scale-105 transition-transform`}>
                                <Icon className="w-8 h-8 text-rose-500" />
                            </div>

                            {/* Package Name */}
                            <h3 className="text-2xl font-bold mb-2">{pkg.name}</h3>

                            {/* Package Description */}
                            {pkg.description && (
                                <p className="text-sm text-slate-400 mb-6 leading-relaxed">
                                    {pkg.description}
                                </p>
                            )}

                            {/* Price */}
                            <div className="mb-6">
                                <div className="flex items-baseline gap-2">
                                    <span className="text-4xl font-black text-white">
                                        ${pkg.price}
                                    </span>
                                    <span className="text-slate-500 text-sm">/شهرياً</span>
                                </div>
                                {!isUnlimited && pkg.messages_included > 0 && (
                                    <p className="text-xs text-slate-500 mt-2">
                                        ${(pkg.price / pkg.messages_included).toFixed(4)} لكل رسالة
                                    </p>
                                )}
                            </div>

                            {/* Features */}
                            <div className="space-y-3 mb-6">
                                <div className="flex items-center gap-3">
                                    <div className="w-5 h-5 rounded-full bg-emerald-500/20 flex items-center justify-center flex-shrink-0">
                                        <Check className="w-3 h-3 text-emerald-400" />
                                    </div>
                                    <span className="text-sm text-slate-300">
                                        {isUnlimited ? 'رسائل غير محدودة' : `${pkg.messages_included.toLocaleString()} رسالة شهرياً`}
                                    </span>
                                </div>

                                {pkg.rollover_enabled && (
                                    <div className="flex items-center gap-3">
                                        <div className="w-5 h-5 rounded-full bg-blue-500/20 flex items-center justify-center flex-shrink-0">
                                            <Check className="w-3 h-3 text-blue-400" />
                                        </div>
                                        <span className="text-sm text-slate-300">
                                            ترحيل الرسائل المتبقية
                                        </span>
                                    </div>
                                )}

                                {pkg.overage_allowed && (
                                    <div className="flex items-center gap-3">
                                        <div className="w-5 h-5 rounded-full bg-amber-500/20 flex items-center justify-center flex-shrink-0">
                                            <Check className="w-3 h-3 text-amber-400" />
                                        </div>
                                        <span className="text-sm text-slate-300">
                                            استخدام إضافي متاح
                                        </span>
                                    </div>
                                )}

                                <div className="flex items-center gap-3">
                                    <div className="w-5 h-5 rounded-full bg-purple-500/20 flex items-center justify-center flex-shrink-0">
                                        <Check className="w-3 h-3 text-purple-400" />
                                    </div>
                                    <span className="text-sm text-slate-300">
                                        دعم فني على مدار الساعة
                                    </span>
                                </div>

                                {isUnlimited && (
                                    <div className="flex items-center gap-3">
                                        <div className="w-5 h-5 rounded-full bg-rose-500/20 flex items-center justify-center flex-shrink-0">
                                            <Check className="w-3 h-3 text-rose-400" />
                                        </div>
                                        <span className="text-sm text-slate-300">
                                            أولوية في الإرسال
                                        </span>
                                    </div>
                                )}
                            </div>

                            {/* Subscribe Button */}
                            <button
                                onClick={() => handleSubscribe(pkg.id)}
                                disabled={subscribing === pkg.id || !pkg.is_active}
                                className={`w-full py-3 rounded-lg font-bold transition-all flex items-center justify-center gap-2 ${isPopular
                                    ? 'bg-rose-600 hover:bg-rose-700 text-white'
                                    : 'bg-slate-800 hover:bg-slate-700 text-white'
                                    } disabled:opacity-50 disabled:cursor-not-allowed`}
                            >
                                {subscribing === pkg.id ? (
                                    <>
                                        <Loader className="w-5 h-5 animate-spin" />
                                        جاري الاشتراك...
                                    </>
                                ) : !pkg.is_active ? (
                                    'غير متاح حالياً'
                                ) : (
                                    <>
                                        <Package className="w-5 h-5" />
                                        اشترك الآن
                                    </>
                                )}
                            </button>

                            {/* Billing Cycle */}
                            {pkg.billing_cycle && (
                                <p className="text-xs text-slate-500 text-center mt-3">
                                    دورة الفوترة: {pkg.billing_cycle === 'monthly' ? 'شهرية' : pkg.billing_cycle}
                                </p>
                            )}
                        </div>
                    );
                })}
            </div>

            {/* Additional Info */}
            <div className="mt-12 bg-slate-900 border border-slate-800 p-6 rounded-xl">
                <h3 className="text-lg font-bold mb-4">معلومات إضافية</h3>
                <div className="grid md:grid-cols-2 gap-6 text-sm text-slate-400">
                    <div>
                        <h4 className="font-bold text-white mb-2">التجديد التلقائي</h4>
                        <p className="leading-relaxed">
                            يتم تجديد اشتراكك تلقائياً في نهاية كل شهر. يمكنك إلغاء التجديد التلقائي في أي وقت من الإعدادات.
                        </p>
                    </div>
                    <div>
                        <h4 className="font-bold text-white mb-2">الاستخدام الإضافي</h4>
                        <p className="leading-relaxed">
                            عند نفاد رسائل الباقة، يمكنك الاستمرار في الإرسال بنفس سعر الباقة. سيتم خصم التكلفة من رصيدك.
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Packages;
