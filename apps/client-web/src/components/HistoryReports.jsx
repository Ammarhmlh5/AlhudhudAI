import React, { useState, useEffect } from 'react';
import { Search, Filter, X, ChevronLeft, ChevronRight, Loader, Download, RotateCcw, Trash2, Loader2 } from 'lucide-react';
import apiClient from '../api/axios';
import { useSocket } from '../contexts/SocketContext';

const STATUSES = [
    { value: '', label: 'الكل' },
    { value: 'DELIVERED', label: 'تم التسليم' },
    { value: 'WAITING_GROUP', label: 'في الانتظار' },
    { value: 'PROCESSING_GROUP', label: 'جاري التسليم' },
    { value: 'FAILED', label: 'الفاشلة' },
];

const SOURCES = [
    { value: '', label: 'الكل' },
    { value: 'WEB_UI', label: 'واجهة الويب' },
    { value: 'SYSTEM', label: 'النظام' },
];

const MSG_TYPES = [
    { value: '', label: 'كل الأنواع' },
    { value: 'INVOICE', label: 'فاتورة' },
    { value: 'PAYMENT', label: 'سداد' },
    { value: 'ALERT', label: 'تنبيه' },
    { value: 'REVERSAL', label: 'عكس تسديد' },
    { value: 'PROMO', label: 'إعلان' },
    { value: 'OTP', label: 'رمز تحقق' },
    { value: 'REMINDER', label: 'تذكير' },
    { value: 'OTHER', label: 'أخرى' },
];

const emptyFilters = {
    status: '', source: '',
    message_type: '', date_from: '', date_to: '',
};

const getStatusBadge = (status) => {
    const map = {
        DELIVERED: 'bg-emerald-500/10 text-emerald-400',
        SENT:      'bg-sky-500/10 text-sky-400',
        PENDING:   'bg-amber-500/10 text-amber-400',
        WAITING:   'bg-amber-500/10 text-amber-400',
        QUEUED:    'bg-amber-500/10 text-amber-400',
        PROCESSING:'bg-sky-500/10 text-sky-400',
        FAILED:    'bg-rose-500/10 text-rose-400',
    };
    const labels = {
        DELIVERED:'تم التسليم', 
        SENT:'جاري التسليم', 
        PROCESSING:'جاري التسليم',
        PENDING:'في الانتظار',
        WAITING:'في الانتظار', 
        QUEUED:'في الانتظار',
        FAILED:'الفاشلة',
    };
    return (
        <span className={`text-xs font-bold px-2 py-1 rounded-lg ${map[status] || 'bg-slate-500/10 text-slate-400'}`}>
            {labels[status] || status}
        </span>
    );
};

const getSourceBadge = (source) => {
    // Categorize sources into Web UI and System
    const isWebUI = source === 'WEB_UI';
    const label = isWebUI ? 'واجهة الويب' : 'النظام';
    const color = isWebUI ? 'text-blue-400' : 'text-emerald-400';
    
    return <span className={`text-xs font-medium ${color}`}>{label}</span>;
};

const fmtDate = (d) => {
    if (!d) return '—';
    return new Date(d).toLocaleString('ar-SA', { year:'numeric', month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' });
};

const HistoryReports = () => {
    const [messages, setMessages]   = useState([]);
    const [loading, setLoading]     = useState(false);
    const [page, setPage]           = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const [filters, setFilters]     = useState(emptyFilters);
    const [applied, setApplied]     = useState(emptyFilters);
    const [selectedIds, setSelectedIds] = useState(new Set());
    const [actionLoading, setActionLoading] = useState(false);
    const [rowAction, setRowAction] = useState({}); // {msgId: 'resend'|'delete'}

    const { messageUpdateNonce } = useSocket();

    useEffect(() => { fetchMessages(); }, [page, applied, messageUpdateNonce]);

    const fetchMessages = async () => {
        setLoading(true);
        try {
            const params = { page, page_size: 25 };
            Object.entries(applied).forEach(([k, v]) => { if (v) params[k] = v; });
            const res = await apiClient.get('/api/v1/messaging/list', { params });
            const data = res.data;
            setMessages(data.messages || []);
            setTotalPages(data.total_pages || 1);
            setTotalCount(data.total_count || 0);
        } catch (e) {
            console.error('Error fetching messages:', e);
        } finally {
            setLoading(false);
        }
    };

    const applyFilters = () => { setApplied({ ...filters }); setPage(1); setSelectedIds(new Set()); };
    const clearFilters = () => { setFilters(emptyFilters); setApplied(emptyFilters); setPage(1); setSelectedIds(new Set()); };

    // Auto-apply filters when they change
    useEffect(() => {
        applyFilters();
    }, [filters.status, filters.source, filters.message_type, filters.date_from, filters.date_to]);

    const handleSelectAll = (e) => {
        if (e.target.checked) {
            setSelectedIds(new Set(messages.map(m => m.id)));
        } else {
            setSelectedIds(new Set());
        }
    };

    const toggleSelect = (id) => {
        const next = new Set(selectedIds);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        setSelectedIds(next);
    };

    const handleBulkDelete = async () => {
        if (!window.confirm(`هل أنت متأكد من حذف ${selectedIds.size} رسالة؟`)) return;
        setActionLoading(true);
        try {
            await apiClient.post('/api/v1/messaging/bulk-delete', { ids: Array.from(selectedIds) });
            fetchMessages();
            setSelectedIds(new Set());
        } catch (e) {
            alert('فشل الحذف');
        } finally {
            setActionLoading(false);
        }
    };

    const handleBulkResend = async () => {
        setActionLoading(true);
        try {
            await apiClient.post('/api/v1/messaging/bulk-resend', { ids: Array.from(selectedIds) });
            alert('تم بدء إعادة الإرسال للرسائل الفاشلة المحددة');
            fetchMessages();
            setSelectedIds(new Set());
        } catch (e) {
            alert('فشل إعادة الإرسال');
        } finally {
            setActionLoading(false);
        }
    };

    const handleSingleResend = async (msg) => {
        setRowAction(prev => ({ ...prev, [msg.id]: 'resend' }));
        try {
            await apiClient.post('/api/v1/messaging/bulk-resend', { ids: [msg.id] });
            fetchMessages();
        } catch (e) {
            alert('فشل إعادة الإرسال');
        } finally {
            setRowAction(prev => ({ ...prev, [msg.id]: null }));
        }
    };

    const handleSingleDelete = async (msg) => {
        if (!window.confirm(`هل تريد حذف الرسالة المرسلة إلى ${msg.phone_number}؟`)) return;
        setRowAction(prev => ({ ...prev, [msg.id]: 'delete' }));
        try {
            await apiClient.post('/api/v1/messaging/bulk-delete', { ids: [msg.id] });
            setMessages(prev => prev.filter(m => m.id !== msg.id));
        } catch (e) {
            alert('فشل الحذف');
        } finally {
            setRowAction(prev => ({ ...prev, [msg.id]: null }));
        }
    };

    const hasActive = Object.values(applied).some(v => v !== '');

    return (
        <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
            <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 mb-6">
                <div>
                    <h2 className="text-3xl font-bold text-white">الأرشيف</h2>
                    <p className="text-slate-400 text-sm mt-1">
                        {totalCount > 0 ? `${totalCount.toLocaleString()} رسالة` : 'سجل الرسائل المرسلة'}
                    </p>
                </div>
                
                <div className="flex flex-wrap items-center gap-3">
                    {selectedIds.size > 0 && (
                        <div className="flex items-center gap-2 animate-in fade-in zoom-in duration-200 bg-rose-500/10 p-1.5 rounded-xl border border-rose-500/20 mr-2">
                            <span className="text-xs font-bold text-rose-400 px-2">{selectedIds.size} محدد</span>
                            <button 
                                onClick={handleBulkResend}
                                disabled={actionLoading}
                                className="px-3 py-1.5 bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white text-xs font-bold rounded-lg transition-colors flex items-center gap-1"
                            >
                                إعادة إرسال
                            </button>
                            <button 
                                onClick={handleBulkDelete}
                                disabled={actionLoading}
                                className="px-3 py-1.5 bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white text-xs font-bold rounded-lg transition-colors flex items-center gap-1"
                            >
                                حذف
                            </button>
                        </div>
                    )}

                    <div className="flex flex-wrap items-center gap-3 bg-slate-900/50 p-3 rounded-2xl border border-slate-800">
                        <div className="flex items-center gap-2">
                            <span className="text-xs font-bold text-slate-500 uppercase tracking-wider">الفترة:</span>
                            <input 
                                type="date" 
                                value={filters.date_from}
                                onChange={e => setFilters(f => ({...f, date_from: e.target.value}))}
                                className="bg-slate-950 border border-slate-800 rounded-lg py-1.5 px-3 text-xs text-white focus:outline-none focus:border-rose-500" 
                            />
                            <span className="text-slate-600">—</span>
                            <input 
                                type="date" 
                                value={filters.date_to}
                                onChange={e => setFilters(f => ({...f, date_to: e.target.value}))}
                                className="bg-slate-950 border border-slate-800 rounded-lg py-1.5 px-3 text-xs text-white focus:outline-none focus:border-rose-500" 
                            />
                        </div>
                        {(filters.date_from || filters.date_to || filters.status || filters.source || filters.message_type) && (
                            <button onClick={clearFilters} className="p-2 text-slate-500 hover:text-rose-500 transition-colors" title="مسح الكل">
                                <X className="w-4 h-4" />
                            </button>
                        )}
                    </div>
                </div>
            </div>



            {/* Table */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-right">
                        <thead className="bg-slate-950/50">
                            <tr>
                                <th className="px-4 py-4 w-10">
                                    <input 
                                        type="checkbox" 
                                        onChange={handleSelectAll}
                                        checked={messages.length > 0 && selectedIds.size === messages.length}
                                        className="rounded border-slate-700 bg-slate-900 text-rose-500 focus:ring-rose-500 focus:ring-offset-slate-900" 
                                    />
                                </th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider">رقم الهاتف</th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider">
                                    <div className="flex flex-col gap-1">
                                        <span>الحالة</span>
                                        <select 
                                            value={filters.status} 
                                            onChange={e => setFilters(f => ({...f, status: e.target.value}))}
                                            className="bg-transparent border-none text-[10px] text-rose-500 focus:ring-0 p-0 cursor-pointer"
                                        >
                                            {STATUSES.map(s => <option key={s.value} value={s.value} className="bg-slate-900 text-white">{s.label}</option>)}
                                        </select>
                                    </div>
                                </th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider">
                                    <div className="flex flex-col gap-1">
                                        <span>النوع</span>
                                        <select 
                                            value={filters.message_type} 
                                            onChange={e => setFilters(f => ({...f, message_type: e.target.value}))}
                                            className="bg-transparent border-none text-[10px] text-rose-500 focus:ring-0 p-0 cursor-pointer"
                                        >
                                            {MSG_TYPES.map(s => <option key={s.value} value={s.value} className="bg-slate-900 text-white">{s.label}</option>)}
                                        </select>
                                    </div>
                                </th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider">
                                    <div className="flex flex-col gap-1">
                                        <span>المصدر</span>
                                        <select 
                                            value={filters.source} 
                                            onChange={e => setFilters(f => ({...f, source: e.target.value}))}
                                            className="bg-transparent border-none text-[10px] text-rose-500 focus:ring-0 p-0 cursor-pointer"
                                        >
                                            {SOURCES.map(s => <option key={s.value} value={s.value} className="bg-slate-900 text-white">{s.label}</option>)}
                                        </select>
                                    </div>
                                </th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider text-center">التاريخ</th>
                                <th className="px-4 py-4 text-xs font-black text-slate-400 tracking-wider text-center">إجراء</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800">
                            {loading && messages.length === 0 ? (
                                <tr><td colSpan="7" className="py-16 text-center text-slate-500">
                                    <Loader className="w-8 h-8 animate-spin mx-auto mb-2 text-rose-500" />
                                    جارٍ التحميل...
                                </td></tr>
                            ) : messages.length === 0 ? (
                                <tr><td colSpan="7" className="py-16 text-center text-slate-500">
                                    <Search className="w-10 h-10 mx-auto mb-3 opacity-30" />
                                    لا توجد رسائل تطابق الفلتر
                                </td></tr>
                            ) : messages.map((msg) => {
                                const busy = rowAction[msg.id];
                                return (
                                <tr key={msg.id} className={`hover:bg-slate-800/40 transition-colors ${selectedIds.has(msg.id) ? 'bg-rose-500/5' : ''}`}>
                                    <td className="px-4 py-3">
                                        <input 
                                            type="checkbox" 
                                            checked={selectedIds.has(msg.id)}
                                            onChange={() => toggleSelect(msg.id)}
                                            className="rounded border-slate-700 bg-slate-900 text-rose-500 focus:ring-rose-500 focus:ring-offset-slate-900" 
                                        />
                                    </td>
                                    <td className="px-4 py-3 font-mono text-sm text-slate-200">{msg.phone_number}</td>
                                    <td className="px-4 py-3">{getStatusBadge(msg.status)}</td>
                                    <td className="px-4 py-3 text-xs text-slate-500">{msg.message_type || '—'}</td>
                                    <td className="px-4 py-3">{getSourceBadge(msg.source)}</td>
                                    <td className="px-4 py-3 text-xs text-slate-500 text-center whitespace-nowrap">{fmtDate(msg.created_at)}</td>
                                    <td className="px-4 py-3">
                                        <div className="flex items-center justify-center gap-2">
                                            {['FAILED', 'PENDING', 'WAITING'].includes(msg.status) && (
                                                <button
                                                    onClick={() => handleSingleResend(msg)}
                                                    disabled={!!busy}
                                                    title="إعادة إرسال"
                                                    className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-bold bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20 disabled:opacity-40 transition-colors"
                                                >
                                                    {busy === 'resend' ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RotateCcw className="w-3.5 h-3.5" />}
                                                </button>
                                            )}
                                            <button
                                                onClick={() => handleSingleDelete(msg)}
                                                disabled={!!busy}
                                                title="حذف"
                                                className="flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-bold bg-rose-500/10 text-rose-400 border border-rose-500/20 hover:bg-rose-500/20 disabled:opacity-40 transition-colors"
                                            >
                                                {busy === 'delete' ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            )})}
                        </tbody>
                    </table>
                </div>

                {/* Pagination */}
                <div className="border-t border-slate-800 p-4 flex items-center justify-between bg-slate-950/20">
                    <span className="text-sm text-slate-500">
                        {loading ? '...' : `صفحة ${page} من ${totalPages}`}
                    </span>
                    <div className="flex gap-2">
                        <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1 || loading}
                            className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-40 transition-colors">
                            <ChevronRight className="w-4 h-4 text-white" />
                        </button>
                        <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages || loading}
                            className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-40 transition-colors">
                            <ChevronLeft className="w-4 h-4 text-white" />
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default HistoryReports;

