import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { Wallet, Zap, AlertCircle, Loader } from 'lucide-react';

const SubscriptionInfo = () => {
    const [financeData, setFinanceData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchFinanceInfo();
    }, []);

    const fetchFinanceInfo = async () => {
        try {
            const response = await apiClient.get('/api/v1/finance/balance');
            const data = response.data;
            setFinanceData({
                quota: data.quota_balance,
                cash: data.cash_balance,
                currency: data.currency
            });
        } catch (err) {
            console.error('Error fetching info:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="glass-card p-6 border-slate-800/50">
                <div className="animate-pulse space-y-4">
                    <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-slate-800 rounded-lg"></div>
                        <div className="h-4 bg-slate-800 rounded w-1/3"></div>
                    </div>
                    <div className="h-10 bg-slate-800 rounded w-full"></div>
                    <div className="h-10 bg-slate-800 rounded w-full"></div>
                </div>
            </div>
        );
    }

    if (error || !financeData) {
        return (
            <div className="glass-card p-6 border-rose-500/20">
                <div className="flex items-center gap-3 text-slate-500">
                    <AlertCircle className="w-5 h-5" />
                    <p className="text-sm">لا توجد بيانات مالية متاحة حالياً</p>
                </div>
            </div>
        );
    }

    return (
        <div className="glass-card p-6 hover:border-emerald-500/30 transition-all border-emerald-500/10">
            <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 flex items-center justify-center">
                        <Wallet className="w-6 h-6 text-emerald-500" />
                    </div>
                    <div>
                        <h3 className="text-lg font-bold">المحفظة السيادية</h3>
                        <p className="text-xs text-slate-500">جرد الأرصدة المعتمدة</p>
                    </div>
                </div>
                <span className="text-[10px] font-black px-2 py-1 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                    موثق
                </span>
            </div>

            <div className="space-y-4">
                {/* Cash Balance */}
                <div className="flex items-center justify-between p-4 rounded-xl bg-slate-800/30 border border-slate-800/50">
                    <div>
                        <p className="text-xs text-slate-500 mb-1">الرصيد النقدي (Cash)</p>
                        <p className="text-xl font-black text-white">
                            {financeData.cash?.toFixed(2)}{" "}
                            <span className="text-xs text-slate-500">{financeData.currency}</span>
                        </p>
                    </div>
                </div>

                {/* Quota Balance */}
                <div className="flex items-center gap-3 p-4 rounded-xl bg-slate-800/30 border border-slate-800/50">
                    <div className="w-10 h-10 rounded-lg bg-slate-900 flex items-center justify-center">
                        <Zap className="w-5 h-5 text-blue-400" />
                    </div>
                    <div className="flex-1">
                        <p className="text-xs text-slate-500 mb-1">رصيد الرسائل (Quota)</p>
                        <p className="text-sm font-bold text-white">
                            {financeData.quota?.toLocaleString()} رسالة
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SubscriptionInfo;
