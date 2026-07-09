import React, { useState, useEffect, useCallback } from 'react';
import {
    Globe, Plus, Shield, Server, Zap, Trash2, RefreshCw,
    ChevronUp, ChevronDown, ToggleLeft, ToggleRight, AlertCircle, CheckCircle
} from 'lucide-react';
import apiClient from '../api/axios';

const GatewayManager = () => {
    const [gateways, setGateways] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [notification, setNotification] = useState(null);

    const showNotif = (msg, type = 'success') => {
        setNotification({ msg, type });
        setTimeout(() => setNotification(null), 3000);
    };

    const fetchGateways = useCallback(async () => {
        try {
            setLoading(true);
            const res = await apiClient.get('/api/v1/channels/list');
            setGateways(Array.isArray(res.data) ? res.data : []);
        } catch (err) {
            console.error('Failed to fetch gateways:', err);
            setGateways([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { fetchGateways(); }, [fetchGateways]);

    const handleAddGateway = async (gatewayData) => {
        try {
            await apiClient.post('/api/v1/channels/create', gatewayData);
            showNotif('تم إضافة البوابة بنجاح');
            fetchGateways();
            setShowAddModal(false);
        } catch (err) {
            showNotif('خطأ في الاتصال بالخادم', 'error');
        }
    };

    const handleUpdatePriority = async (id, newPriority) => {
        try {
            await apiClient.put(`/api/v1/channels/${id}`, { priority: newPriority });
            setGateways(prev => prev.map(g => g.id === id ? { ...g, priority: newPriority } : g)
                .sort((a, b) => b.priority - a.priority));
        } catch (err) {
            showNotif('خطأ في الاتصال بالخادم', 'error');
        }
    };

    const handleToggleStatus = async (id, currentStatus) => {
        const newStatus = currentStatus === 'ONLINE' ? 'OFFLINE' : 'ONLINE';
        try {
            await apiClient.put(`/api/v1/channels/${id}`, { status: newStatus });
            setGateways(prev => prev.map(g => g.id === id ? { ...g, status: newStatus } : g));
            showNotif(`تم تغيير الحالة إلى ${newStatus}`);
        } catch (err) {
            showNotif('خطأ في الاتصال بالخادم', 'error');
        }
    };

    const handleDelete = async (id, key) => {
        if (!window.confirm(`هل أنت متأكد من حذف البوابة "${key}"؟`)) return;
        try {
            await apiClient.delete(`/api/v1/channels/${id}`);
            showNotif('تم حذف البوابة');
            setGateways(prev => prev.filter(g => g.id !== id));
        } catch (err) {
            showNotif('خطأ في الاتصال بالخادم', 'error');
        }
    };

    const getTypeLabel = (type) => {
        switch (type) {
            case 'GATEWAY_HUDHUD': return 'بوابة ALHudhudAI العالمية';
            case 'GATEWAY_PRIVATE': return 'بوابة API خاصة';
            case 'PHONE': return 'جهاز هاتف';
            default: return type;
        }
    };

    const getTypeColor = (type) => {
        switch (type) {
            case 'GATEWAY_HUDHUD': return 'from-blue-600 to-indigo-700';
            case 'GATEWAY_PRIVATE': return 'from-violet-600 to-purple-700';
            case 'PHONE': return 'from-emerald-600 to-teal-700';
            default: return 'from-slate-600 to-slate-700';
        }
    };

    return (
        <div className="animate-in slide-in-from-right-8 duration-500">
            {/* Notification */}
            {notification && (
                <div className={`fixed top-6 right-6 z-50 flex items-center gap-3 px-5 py-3 rounded-2xl shadow-2xl text-sm font-bold animate-in slide-in-from-top-4 duration-300 ${notification.type === 'error' ? 'bg-rose-600 text-white' : 'bg-emerald-600 text-white'}`}>
                    {notification.type === 'error' ? <AlertCircle className="w-4 h-4" /> : <CheckCircle className="w-4 h-4" />}
                    {notification.msg}
                </div>
            )}

            {/* Header */}
            <div className="flex flex-col md:flex-row md:items-center justify-between mb-8 gap-4">
                <div>
                    <h2 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-white to-slate-400">
                        بوابات الإرسال
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">
                        إدارة مسارات الإرسال وضبط أولوياتها. الأعلى أولوية يُستخدم أولاً.
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <button onClick={fetchGateways} className="p-2.5 bg-slate-800 hover:bg-slate-700 rounded-xl transition-colors" title="تحديث">
                        <RefreshCw className="w-4 h-4 text-slate-400" />
                    </button>
                    <button
                        onClick={() => setShowAddModal(true)}
                        className="btn-premium py-2.5 px-6 text-sm flex items-center justify-center gap-2 group whitespace-nowrap"
                    >
                        <Plus className="w-4 h-4 group-hover:rotate-90 transition-transform" />
                        ربط بوابة جديدة
                    </button>
                </div>
            </div>

            {/* Priority Legend */}
            <div className="mb-6 p-4 rounded-2xl bg-slate-900/50 border border-slate-800/50 flex items-center gap-4 text-xs text-slate-400">
                <Zap className="w-4 h-4 text-amber-400 flex-shrink-0" />
                <span>
                    <strong className="text-slate-200">كيف يعمل التوجيه:</strong> عند إرسال رسالة، يختار النظام تلقائياً البوابة <strong className="text-emerald-400">ONLINE</strong> ذات الأولوية الأعلى. إذا كانت معطلة، ينتقل للتالية.
                </span>
            </div>

            {/* Gateway Cards */}
            <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
                {loading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="glass-card p-6 animate-pulse">
                            <div className="h-4 bg-slate-800 rounded mb-4 w-3/4" />
                            <div className="h-3 bg-slate-800 rounded mb-2 w-1/2" />
                            <div className="h-3 bg-slate-800 rounded w-2/3" />
                        </div>
                    ))
                ) : gateways.length === 0 ? (
                    <div className="col-span-3 glass-card p-12 flex flex-col items-center justify-center text-center">
                        <Server className="w-12 h-12 text-slate-700 mb-4" />
                        <h4 className="font-bold text-slate-400 mb-2">لا توجد بوابات مضافة</h4>
                        <p className="text-sm text-slate-600 mb-6">أضف بوابة إرسال لبدء استخدام النظام</p>
                        <button onClick={() => setShowAddModal(true)} className="btn-premium py-2 px-6 text-sm">
                            إضافة أول بوابة
                        </button>
                    </div>
                ) : (
                    gateways.map((gw, index) => (
                        <div key={gw.id} className={`glass-card group overflow-hidden relative transition-all duration-300 ${gw.status === 'ONLINE' ? 'hover:border-emerald-500/30' : 'hover:border-slate-600/50 opacity-75'}`}>
                            {/* Priority Badge */}
                            <div className="absolute top-4 left-4 z-10">
                                <span className="bg-slate-900 border border-slate-700 text-[10px] font-black uppercase tracking-widest px-2 py-1 rounded-lg text-slate-400">
                                    #{index + 1} أولوية
                                </span>
                            </div>

                            {/* Status Indicator */}
                            <div className={`absolute top-0 right-0 w-1 h-full ${gw.status === 'ONLINE' ? 'bg-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.4)]' : 'bg-slate-700'}`} />

                            <div className="p-6 pt-12">
                                {/* Type Icon & Status */}
                                <div className="flex justify-between items-start mb-4">
                                    <div className={`bg-gradient-to-br ${getTypeColor(gw.type)} p-3 rounded-2xl shadow-lg`}>
                                        <Globe className="w-5 h-5 text-white" />
                                    </div>
                                    <button
                                        onClick={() => handleToggleStatus(gw.id, gw.status)}
                                        className="flex items-center gap-2 text-xs font-bold transition-colors"
                                        title={gw.status === 'ONLINE' ? 'إيقاف' : 'تشغيل'}
                                    >
                                        {gw.status === 'ONLINE' ? (
                                            <><ToggleRight className="w-6 h-6 text-emerald-400" /><span className="text-emerald-400">نشط</span></>
                                        ) : (
                                            <><ToggleLeft className="w-6 h-6 text-slate-500" /><span className="text-slate-500">معطل</span></>
                                        )}
                                    </button>
                                </div>

                                {/* Name & Type */}
                                <h4 className="text-base font-bold text-slate-100 mb-1 font-mono truncate">
                                    {gw.channel_key}
                                </h4>
                                <p className="text-xs text-slate-500 mb-5">{getTypeLabel(gw.type)}</p>

                                {/* Priority Control */}
                                <div className="border-t border-slate-800/80 pt-4">
                                    <div className="flex items-center justify-between mb-2">
                                        <span className="text-xs text-slate-500 font-bold uppercase tracking-wider">الأولوية</span>
                                        <span className="text-lg font-black text-amber-400">{gw.priority}</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="range"
                                            min="1"
                                            max="100"
                                            value={gw.priority}
                                            onChange={(e) => setGateways(prev => prev.map(g => g.id === gw.id ? { ...g, priority: parseInt(e.target.value) } : g))}
                                            onMouseUp={(e) => handleUpdatePriority(gw.id, parseInt(e.target.value))}
                                            onTouchEnd={(e) => handleUpdatePriority(gw.id, parseInt(e.target.value))}
                                            className="flex-1 h-1.5 bg-slate-800 rounded-full appearance-none cursor-pointer accent-amber-400"
                                        />
                                        <div className="flex flex-col gap-0.5">
                                            <button
                                                onClick={() => handleUpdatePriority(gw.id, Math.min(100, gw.priority + 1))}
                                                className="p-0.5 hover:bg-slate-800 rounded transition-colors"
                                            >
                                                <ChevronUp className="w-3 h-3 text-slate-400" />
                                            </button>
                                            <button
                                                onClick={() => handleUpdatePriority(gw.id, Math.max(1, gw.priority - 1))}
                                                className="p-0.5 hover:bg-slate-800 rounded transition-colors"
                                            >
                                                <ChevronDown className="w-3 h-3 text-slate-400" />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {/* Footer */}
                            <div className="px-6 py-3 bg-slate-900/50 border-t border-slate-800/50 flex justify-between items-center">
                                <span className="text-[10px] text-slate-600 font-mono">
                                    {new Date(gw.created_at).toLocaleDateString('ar-SA')}
                                </span>
                                <button
                                    onClick={() => handleDelete(gw.id, gw.channel_key)}
                                    className="p-1.5 hover:bg-rose-500/10 rounded-lg transition-colors group/del"
                                    title="حذف"
                                >
                                    <Trash2 className="w-3.5 h-3.5 text-slate-600 group-hover/del:text-rose-500 transition-colors" />
                                </button>
                            </div>
                        </div>
                    ))
                )}
            </div>

            {/* Security Note */}
            <div className="mt-8 glass-card p-5 border-slate-800/50 flex items-start gap-4">
                <Shield className="w-5 h-5 text-emerald-500 flex-shrink-0 mt-0.5" />
                <div>
                    <h3 className="font-bold text-sm mb-1">عزل تام بين الشركات</h3>
                    <p className="text-xs text-slate-500">بوابات كل شركة معزولة تماماً. لا يمكن لأي شركة رؤية أو استخدام بوابات شركة أخرى. جميع التغييرات تسري فوراً على الرسائل القادمة.</p>
                </div>
            </div>

            {/* Add Gateway Modal */}
            {showAddModal && (
                <AddGatewayModal
                    onClose={() => setShowAddModal(false)}
                    onAdd={handleAddGateway}
                />
            )}
        </div>
    );
};

const AddGatewayModal = ({ onClose, onAdd }) => {
    const [formData, setFormData] = useState({
        type: 'GATEWAY_PRIVATE',
        channel_key: '',
        config: '{}',
        priority: 10
    });
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        await onAdd(formData);
        setLoading(false);
    };

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="glass-card max-w-md w-full p-8 animate-in zoom-in-95 duration-300">
                <h3 className="text-2xl font-bold mb-2">ربط بوابة جديدة</h3>
                <p className="text-sm text-slate-500 mb-6">ستُستخدم هذه البوابة لإرسال الرسائل وفق الأولوية المحددة.</p>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-xs font-bold text-slate-400 mb-2 uppercase tracking-wider">نوع البوابة</label>
                        <select
                            value={formData.type}
                            onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                            className="w-full bg-slate-900/50 border border-slate-800 rounded-xl py-3 px-4 text-sm focus:outline-none focus:border-blue-500/50 text-slate-300"
                        >
                            <option value="GATEWAY_PRIVATE">بوابة API خاصة</option>
                            <option value="GATEWAY_HUDHUD">بوابة ALHudhudAI العالمية</option>
                            <option value="PHONE">جهاز هاتف</option>
                        </select>
                    </div>

                    <div>
                        <label className="block text-xs font-bold text-slate-400 mb-2 uppercase tracking-wider">معرف البوابة (Channel Key)</label>
                        <input
                            type="text"
                            value={formData.channel_key}
                            onChange={(e) => setFormData({ ...formData, channel_key: e.target.value })}
                            className="w-full bg-slate-900/50 border border-slate-800 rounded-xl py-3 px-4 text-sm focus:outline-none focus:border-blue-500/50 font-mono"
                            placeholder="my_private_gateway"
                            required
                        />
                    </div>

                    <div>
                        <label className="block text-xs font-bold text-slate-400 mb-2 uppercase tracking-wider">الإعدادات (JSON)</label>
                        <textarea
                            value={formData.config}
                            onChange={(e) => setFormData({ ...formData, config: e.target.value })}
                            className="w-full bg-slate-900/50 border border-slate-800 rounded-xl py-3 px-4 text-sm focus:outline-none focus:border-blue-500/50 font-mono h-20 resize-none"
                            placeholder='{"api_key": "..."}'
                        />
                    </div>

                    <div>
                        <label className="block text-xs font-bold text-slate-400 mb-2 uppercase tracking-wider">
                            الأولوية: <span className="text-amber-400 font-black">{formData.priority}</span>
                        </label>
                        <input
                            type="range"
                            min="1"
                            max="100"
                            value={formData.priority}
                            onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                            className="w-full h-1.5 bg-slate-800 rounded-full appearance-none cursor-pointer accent-amber-400"
                        />
                        <div className="flex justify-between text-[10px] text-slate-600 mt-1">
                            <span>1 (أدنى)</span>
                            <span>100 (أعلى)</span>
                        </div>
                    </div>

                    <div className="flex gap-3 pt-2">
                        <button type="button" onClick={onClose} className="flex-1 py-3 bg-slate-800 hover:bg-slate-700 rounded-xl transition-colors text-sm">
                            إلغاء
                        </button>
                        <button type="submit" disabled={loading} className="flex-1 btn-premium py-3 text-sm disabled:opacity-50">
                            {loading ? 'جاري الإضافة...' : 'ربط البوابة'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default GatewayManager;
