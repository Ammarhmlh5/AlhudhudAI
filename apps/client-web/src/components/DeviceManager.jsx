import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { Copy } from 'lucide-react';

const DeviceManager = () => {
    const [activeTab, setActiveTab] = useState('channels'); // 'channels' or 'phones'
    const [channels, setChannels] = useState([]);
    const [devices, setDevices] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [actionResult, setActionResult] = useState(null);
    const [copied, setCopied] = useState(null);

    const handleCopy = (text) => {
        if (!text) return;
        navigator.clipboard.writeText(text);
        setCopied(text);
        setTimeout(() => setCopied(null), 2000);
    };

    const [channelForm, setChannelForm] = useState({ name: '' });

    useEffect(() => {
        refreshData();
    }, [activeTab]);

    const refreshData = async () => {
        setLoading(true);
        if (activeTab === 'channels') {
            await fetchChannels();
        } else {
            await fetchDevices();
        }
        setLoading(false);
    };

    const fetchChannels = async () => {
        try {
            const res = await apiClient.get('/api/v1/channels/list');
            setChannels(res.data || []);
        } catch (err) {
            console.error('Failed to fetch channels:', err);
        }
    };

    const fetchDevices = async () => {
        try {
            const res = await apiClient.get('/api/v1/devices/list');
            setDevices(res.data || []);
        } catch (err) {
            console.error('Failed to fetch devices:', err);
        }
    };

    const handleCreateChannel = async (e) => {
        e.preventDefault();
        try {
            // For phone channels, we call the pairing endpoint
            // If the user just wants a generic channel, they'd use channels/create
            // But here the UI seems to expect a device_pair_key
            const res = await apiClient.post('/api/v1/devices/pair', {
                device_name: channelForm.name || 'Phone Channel',
                hardware_id: 'PENDING_' + Math.random().toString(36).substr(2, 9),
                phone_number: 'PENDING',
                is_primary: true
            });
            if (res.data.device_pair_key) {
                setActionResult({ title: 'تم إنشاء القناة!', key: res.data.device_pair_key });
                fetchChannels();
            }
        } catch (err) {
            console.error('Creation failed:', err);
        }
    };

    const handleDeleteChannel = async (id) => {
        if (!window.confirm('هل أنت متأكد من حذف هذه القناة؟ سيتم فصل الأجهزة المرتبطة بها.')) return;
        try {
            await apiClient.delete(`/api/v1/channels/${id}`);
            fetchChannels();
        } catch (err) {
            console.error('Delete failed:', err);
        }
    };

    const toggleChannelStatus = async (id, currentStatus) => {
        const nextStatus = currentStatus === 'ONLINE' ? 'OFFLINE' : 'ONLINE';
        try {
            await apiClient.put(`/api/v1/channels/${id}`, { status: nextStatus });
            fetchChannels();
        } catch (err) {
            console.error('Status update failed:', err);
        }
    };

    const handleRegenerateKey = async (id) => {
        if (!window.confirm('بعد تجديد المفتاح، ستحتاج لتحديثه في البرنامج المحاسبي. هل تريد الاستمرار؟')) return;
        try {
            // Regeneration endpoint not fully implemented in current backend version 3.2.7
            // We'll simulate it or wait for backend update. For now, let's point to a hypothetical endpoint
            const res = await apiClient.put(`/api/v1/channels/${id}/key`);
            if (res.data.channel_key) {
                setActionResult({ title: 'تم تجديد المفتاح!', key: res.data.channel_key });
            }
            fetchChannels();
        } catch (err) {
            console.error('Regeneration failed:', err);
        }
    };

    const handleDeleteDevice = async (hardwareID) => {
        if (!window.confirm('هل أنت متأكد من حذف هذا الهاتف المسجل؟')) return;
        try {
            await apiClient.delete(`/api/v1/devices/${hardwareID}`);
            fetchDevices();
        } catch (err) {
            console.error('Device delete failed:', err);
        }
    };

    const handleTestConnection = async (hardwareID) => {
        try {
            const res = await apiClient.get(`/api/v1/devices/${hardwareID}/ping`);
            if (res.data.status === 'ONLINE') {
                alert('الهاتف متصل بالسيرفر بنجاح!');
            } else {
                alert('الهاتف غير متصل حالياً.');
            }
            fetchDevices(); // Refresh status
        } catch (err) {
            console.error('Ping failed:', err);
            alert('حدث خطأ أثناء فحص الاتصال');
        }
    };

    return (
        <div className="p-8">
            <div className="flex justify-between items-center mb-8">
                <div>
                    <h2 className="text-2xl font-bold text-white mb-2">إدارة الأجهزة والقنوات</h2>
                    <div className="flex gap-4">
                        <button
                            onClick={() => setActiveTab('channels')}
                            className={`pb-2 px-1 font-bold ${activeTab === 'channels' ? 'text-rose-500 border-b-2 border-rose-500' : 'text-slate-500 hover:text-white transition-colors'}`}
                        >
                            قنوات الهواتف (Channels)
                        </button>
                        <button
                            onClick={() => setActiveTab('phones')}
                            className={`pb-2 px-1 font-bold ${activeTab === 'phones' ? 'text-rose-500 border-b-2 border-rose-500' : 'text-slate-500 hover:text-white transition-colors'}`}
                        >
                            الهواتف المسجلة (Phones)
                        </button>
                    </div>
                </div>
                {activeTab === 'channels' && (
                    <button
                        onClick={() => setShowCreateModal(true)}
                        className="bg-rose-600 hover:bg-rose-700 text-white px-6 py-2 rounded-lg font-bold transition-all shadow-lg shadow-rose-600/20"
                    >
                        + إنشاء قناة هاتف جديدة
                    </button>
                )}
            </div>

            {loading ? (
                <div className="flex items-center justify-center p-20">
                    <div className="w-8 h-8 border-4 border-rose-600 border-t-transparent rounded-full animate-spin"></div>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {activeTab === 'channels' ? (
                        channels.length > 0 ? channels.map(channel => (
                            <div key={channel.id} className="bg-slate-900 border border-slate-800 p-6 rounded-xl relative group overflow-hidden">
                                <div className="flex justify-between items-start mb-4">
                                    <div>
                                        <span className="text-[10px] text-slate-500 block mb-1">مفتاح APK (المحاسبة)</span>
                                        <div 
                                            className="flex items-center gap-2 cursor-pointer group"
                                            onClick={() => handleCopy(channel.channel_key)}
                                            title="اضغط لنسخ المفتاح"
                                        >
                                            <h3 className="text-white font-mono font-bold text-sm tracking-widest group-hover:text-rose-500 transition-colors">
                                                {channel.channel_key}
                                            </h3>
                                            <Copy className="w-4 h-4 text-slate-600 group-hover:text-rose-500" />
                                            {copied === channel.channel_key && <span className="text-emerald-500 text-[10px] font-bold">تم النسخ</span>}
                                        </div>
                                    </div>
                                    <span className={`px-2 py-1 rounded text-[10px] font-bold ${channel.is_default ? 'bg-amber-500/10 text-amber-500' : 'bg-slate-800 text-slate-400'}`}>
                                        {channel.is_default ? 'DEFAULT' : 'EXTRA'}
                                    </span>
                                </div>
                                <div className="space-y-2 mb-6 text-sm">
                                    <div className="flex justify-between">
                                        <span className="text-slate-500">الحالة:</span>
                                        <span className={channel.status === 'ONLINE' ? 'text-emerald-500' : 'text-rose-500'}>{channel.status}</span>
                                    </div>
                                    <div className="flex justify-between">
                                        <span className="text-slate-500">النوع:</span>
                                        <span className="text-slate-300">{channel.type}</span>
                                    </div>
                                </div>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => toggleChannelStatus(channel.id, channel.status)}
                                        className="flex-1 bg-slate-800 hover:bg-slate-700 text-white py-2 rounded text-xs transition-colors"
                                    >
                                        {channel.status === 'ONLINE' ? 'توقيف' : 'تفعيل'}
                                    </button>
                                    <button
                                        onClick={() => handleRegenerateKey(channel.id)}
                                        className="flex-1 bg-slate-800 hover:bg-slate-700 text-white py-2 rounded text-xs transition-colors"
                                    >
                                        تجديد المفتاح
                                    </button>
                                    <button
                                        onClick={() => handleDeleteChannel(channel.id)}
                                        className="flex-1 bg-slate-800 hover:bg-rose-950 hover:text-rose-500 text-slate-400 py-2 rounded text-xs transition-colors"
                                    >
                                        حذف
                                    </button>
                                </div>
                            </div>
                        )) : (
                            <div className="col-span-full text-center py-20 text-slate-500">لا توجد قنوات منشأة حالياً.</div>
                        )
                    ) : (
                        devices.length > 0 ? devices.map(device => (
                            <div key={device.id} className="bg-slate-900 border border-slate-800 p-6 rounded-xl relative group">
                                <div className="flex justify-between items-start mb-4">
                                    <h3 className="text-white font-bold">{device.device_name}</h3>
                                    <span className={`w-2 h-2 rounded-full ${device.status === 'ONLINE' ? 'bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]' : 'bg-slate-600'}`} />
                                </div>
                                <div className="space-y-1 mb-6 text-sm">
                                    <p className="text-slate-400">{device.phone_number}</p>
                                    <p className="text-slate-600 font-mono text-[10px] break-all">{device.hardware_id}</p>
                                </div>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => handleTestConnection(device.hardware_id)}
                                        className="flex-1 bg-slate-800 hover:bg-emerald-900 hover:text-emerald-500 text-slate-400 py-2 rounded text-xs transition-colors"
                                    >
                                        اختبار الاتصال
                                    </button>
                                    <button
                                        onClick={() => handleDeleteDevice(device.hardware_id)}
                                        className="flex-1 bg-slate-800 hover:bg-rose-950 hover:text-rose-500 text-slate-400 py-2 rounded text-xs transition-colors"
                                    >
                                        حذف السجل
                                    </button>
                                </div>
                            </div>
                        )) : (
                            <div className="col-span-full text-center py-20 text-slate-500">لم يتم ربط أي هواتف مادية بعد.</div>
                        )
                    )}
                </div>
            )}

            {showCreateModal && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
                    <div className="bg-slate-900 border border-slate-800 p-8 rounded-2xl max-w-md w-full shadow-2xl">
                        {!actionResult ? (
                            <>
                                <h3 className="text-xl font-bold text-white mb-6">إنشاء قناة هاتف جديدة</h3>
                                <form onSubmit={handleCreateChannel} className="space-y-4">
                                    <div>
                                        <label className="block text-slate-400 text-sm mb-2">اسم أو معرف القناة (اختياري)</label>
                                        <input
                                            type="text"
                                            className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-white focus:border-rose-500 outline-none transition-colors"
                                            value={channelForm.name}
                                            onChange={e => setChannelForm({ ...channelForm, name: e.target.value })}
                                            placeholder="مثال: قناة المبيعات"
                                        />
                                    </div>
                                    <div className="flex gap-4 pt-4">
                                        <button type="submit" className="flex-1 bg-rose-600 hover:bg-rose-700 text-white py-3 rounded-lg font-bold transition-colors">
                                            تأكيد الإنشاء
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => setShowCreateModal(false)}
                                            className="px-6 py-3 text-slate-400 hover:text-white transition-colors"
                                        >
                                            إلغاء
                                        </button>
                                    </div>
                                </form>
                            </>
                        ) : (
                            <div className="text-center">
                                <div className="w-16 h-16 bg-emerald-500/10 text-emerald-500 rounded-full flex items-center justify-center mx-auto mb-6">
                                    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                                    </svg>
                                </div>
                                <h3 className="text-xl font-bold text-white mb-2">{actionResult.title}</h3>
                                <p className="text-slate-400 text-sm mb-6 px-4">استخدم مفتاح الربط التالي لربط هاتف مادي بهذه القناة:</p>
                                <div 
                                    className="bg-slate-950 p-4 rounded-xl border border-slate-800 font-mono text-rose-500 text-lg mb-8 break-all cursor-pointer relative hover:border-rose-500/50 transition-colors flex justify-between items-center group"
                                    onClick={() => handleCopy(actionResult.key)}
                                >
                                    <span>{actionResult.key}</span>
                                    <Copy className="w-5 h-5 text-slate-500 group-hover:text-rose-500" />
                                    {copied === actionResult.key && (
                                        <span className="absolute -top-3 left-1/2 -translate-x-1/2 bg-emerald-500 text-white text-xs py-1 px-3 rounded-full font-bold shadow-lg">
                                            تم النسخ بنجاح!
                                        </span>
                                    )}
                                </div>
                                <button
                                    onClick={() => {
                                        setShowCreateModal(false);
                                        setActionResult(null);
                                        setActiveTab('channels');
                                    }}
                                    className="w-full bg-slate-800 hover:bg-slate-700 text-white py-3 rounded-lg font-bold transition-colors"
                                >
                                    إغلاق
                                </button>
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
};

export default DeviceManager;
