import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { MessageSquare, TrendingUp, Clock, AlertTriangle, Package } from 'lucide-react';

const MessageUsage = () => {
    const [usage, setUsage] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchUsage();
    }, []);

    const fetchUsage = async () => {
        try {
            const response = await apiClient.get('/api/v1/finance/usage');
            setUsage(response.data.usage);
        } catch (err) {
            console.error('Error fetching usage:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="glass-card p-6">
                <div className="animate-pulse space-y-4">
                    <div className="h-4 bg-slate-800 rounded w-1/3"></div>
                    <div className="h-24 bg-slate-800 rounded"></div>
                </div>
            </div>
        );
    }

    if (error || !usage) {
        return (
            <div className="glass-card p-6">
                <div className="flex items-center gap-3 text-slate-500">
                    <AlertTriangle className="w-5 h-5" />
                    <p className="text-sm">لا توجد بيانات استخدام متاحة</p>
                </div>
            </div>
        );
    }

    const messagesUsedPercentage = usage.messages_included > 0
        ? Math.min((usage.messages_used / usage.messages_included) * 100, 100)
        : 0;

    const formatDate = (dateString) => {
        if (!dateString) return 'N/A';
        return new Date(dateString).toLocaleDateString('ar-SA', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    };

    return (
        <div className="glass-card p-6 hover:border-rose-500/30 transition-all">
            <div className="flex items-center gap-3 mb-6">
                <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center">
                    <MessageSquare className="w-6 h-6 text-blue-500" />
                </div>
                <div>
                    <h3 className="text-lg font-bold">استخدام الرسائل</h3>
                    <p className="text-xs text-slate-500">تفاصيل الاستهلاك الشهري</p>
                </div>
            </div>

            {/* Messages Progress Bar */}
            <div className="mb-6">
                <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-slate-400">الرسائل المستخدمة</span>
                    <span className="text-sm font-bold text-white">
                        {usage.messages_used || 0} / {usage.messages_included || 0}
                    </span>
                </div>
                <div className="w-full h-3 bg-slate-800 rounded-full overflow-hidden">
                    <div
                        className={`h-full transition-all duration-500 ${messagesUsedPercentage >= 90
                            ? 'bg-gradient-to-r from-rose-500 to-orange-500'
                            : messagesUsedPercentage >= 70
                                ? 'bg-gradient-to-r from-amber-500 to-yellow-500'
                                : 'bg-gradient-to-r from-blue-500 to-cyan-500'
                            }`}
                        style={{ width: `${messagesUsedPercentage}%` }}
                    />
                </div>
                <div className="flex items-center justify-between mt-2">
                    <span className="text-xs text-slate-500">
                        {messagesUsedPercentage.toFixed(1)}% مستخدم
                    </span>
                    <span className="text-xs font-bold text-emerald-400">
                        {usage.messages_remaining || 0} متبقي
                    </span>
                </div>
            </div>

            {/* Rollover Messages */}
            {usage.rollover_messages && usage.rollover_messages.length > 0 && (
                <div className="mb-6 p-4 rounded-xl bg-amber-500/5 border border-amber-500/20">
                    <div className="flex items-center gap-2 mb-3">
                        <Package className="w-4 h-4 text-amber-400" />
                        <h4 className="text-sm font-bold text-amber-400">رسائل مرحّلة</h4>
                    </div>
                    <div className="space-y-2">
                        {usage.rollover_messages.map((rollover, index) => (
                            <div key={index} className="flex items-center justify-between text-xs">
                                <div className="flex items-center gap-2">
                                    <Clock className="w-3 h-3 text-slate-500" />
                                    <span className="text-slate-400">
                                        {rollover.messages_remaining || 0} رسالة
                                    </span>
                                </div>
                                <span className="text-slate-500">
                                    تنتهي: {formatDate(rollover.expiry_date)}
                                </span>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Overage Usage */}
            {usage.overage_messages > 0 && (
                <div className="p-4 rounded-xl bg-rose-500/5 border border-rose-500/20">
                    <div className="flex items-center gap-2 mb-3">
                        <TrendingUp className="w-4 h-4 text-rose-400" />
                        <h4 className="text-sm font-bold text-rose-400">استخدام إضافي</h4>
                    </div>
                    <div className="space-y-2">
                        <div className="flex items-center justify-between">
                            <span className="text-xs text-slate-400">رسائل إضافية</span>
                            <span className="text-sm font-bold text-white">
                                {usage.overage_messages || 0}
                            </span>
                        </div>
                        <div className="flex items-center justify-between">
                            <span className="text-xs text-slate-400">التكلفة الإضافية</span>
                            <span className="text-sm font-bold text-rose-400">
                                ${usage.overage_cost?.toFixed(2) || '0.00'}
                            </span>
                        </div>
                    </div>
                </div>
            )}

            {/* Warning if low messages */}
            {usage.messages_remaining !== undefined && usage.messages_remaining < (usage.messages_included * 0.1) && usage.messages_remaining > 0 && (
                <div className="mt-4 p-3 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center gap-3">
                    <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0" />
                    <p className="text-xs text-amber-400">
                        تحذير: الرسائل المتبقية أقل من 10%
                    </p>
                </div>
            )}
        </div>
    );
};

export default MessageUsage;
