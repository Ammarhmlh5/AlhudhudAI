import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import {
    MessageSquare, Filter, Download, ChevronLeft, ChevronRight, Loader,
    Search, LayoutList, Columns, FileText, CheckCircle2, XCircle, Clock
} from 'lucide-react';
import { useSocket } from '../contexts/SocketContext';

const MessagesReport = () => {
    const [viewMode, setViewMode] = useState('flat'); // 'flat' or 'grouped' (batch view)
    const [data, setData] = useState([]);
    const [summary, setSummary] = useState({ total_messages: 0, delivered: 0, failed: 0, pending: 0, total_cost: 0 });
    const [loading, setLoading] = useState(false);
    
    // Pagination
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    
    // Filters
    const [filters, setFilters] = useState({
        batch_ref: '',
        msg_ref: '',
        ext_ref: '',
        status: '',
        phone_number: ''
    });

    // We store expanded batches simply as an array of batch refs
    const [expandedBatches, setExpandedBatches] = useState([]);
    const [batchDetails, setBatchDetails] = useState({}); // Cache for batch details
    const [loadingBatches, setLoadingBatches] = useState({});

    const { messageUpdateNonce } = useSocket();

    useEffect(() => {
        fetchData();
        // Reset expansions when mode changes or filters change
        if (viewMode === 'flat') {
            setExpandedBatches([]);
        }
    }, [viewMode, currentPage, filters, messageUpdateNonce]);

    const params = new URLSearchParams({
        page: currentPage,
        view: viewMode,
        ...Object.fromEntries(Object.entries(filters).filter(([_, v]) => v !== ''))
    }).toString();

    const fetchData = async () => {
        setLoading(true);
        try {
            const response = await apiClient.get(`/api/v1/tenant/messages/dashboard-report?${params}`);
            const result = response.data;
            setData(result.data || []);
            setTotalPages(result.meta.total_pages || 1);
            if (result.summary) {
                setSummary(result.summary);
            }
        } catch (error) {
            console.error('Error fetching messages report:', error);
        } finally {
            setLoading(false);
        }
    };

    const fetchBatchMessages = async (batchRef) => {
        if (batchDetails[batchRef] || loadingBatches[batchRef]) return; // Already have it or loading
        
        setLoadingBatches(prev => ({ ...prev, [batchRef]: true }));
        try {
            const response = await apiClient.get(`/api/v1/tenant/messages/dashboard-report?${params}`);
            setBatchDetails(prev => ({ ...prev, [batchRef]: response.data.data || [] }));
        } catch (err) {
            console.error('Failed to load batch details', err);
        } finally {
            setLoadingBatches(prev => ({ ...prev, [batchRef]: false }));
        }
    };

    const toggleBatchExpansion = (batchRef) => {
        if (expandedBatches.includes(batchRef)) {
            setExpandedBatches(prev => prev.filter(r => r !== batchRef));
        } else {
            setExpandedBatches(prev => [...prev, batchRef]);
            fetchBatchMessages(batchRef);
        }
    };

    const handleFilterChange = (e) => {
        const { name, value } = e.target;
        setFilters(prev => ({ ...prev, [name]: value }));
        setCurrentPage(1); // Reset to first page
    };

    const clearFilters = () => {
        setFilters({ batch_ref: '', msg_ref: '', ext_ref: '', status: '', phone_number: '' });
        setCurrentPage(1);
    };

    const handleExport = () => {
        alert('سيتم توفير خيار تصدير JSON للأنظمة قريباً');
    };

    const formatDate = (dateString) => {
        if (!dateString) return 'N/A';
        return new Date(dateString).toLocaleString('ar-SA', {
            year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
        });
    };

    const getStatusStyle = (status) => {
        switch (status) {
            case 'DELIVERED':
            case 'DONE': return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
            case 'SENT': return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
            case 'FAILED': return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
            case 'PARTIAL': return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
            default: return 'bg-slate-500/10 text-slate-400 border-slate-500/20';
        }
    };

    const StatusBadge = ({ status }) => (
        <span className={`px-2 py-1 rounded-md text-xs font-medium border ${getStatusStyle(status)}`}>
            {status}
        </span>
    );

    // Render single message row (used in flat mode or inside expanded batches)
    const renderMessageRow = (msg, isSubRow = false) => (
        <tr key={msg.id} className={`${isSubRow ? 'bg-slate-900/50' : 'hover:bg-slate-800/30'} border-b border-slate-800 transition-colors`}>
            <td className={`py-4 ${isSubRow ? 'pl-6 pr-10' : 'px-6'}`}>
                <div className="flex flex-col">
                    <span className="font-mono text-xs text-slate-300">{msg.msg_ref}</span>
                    {msg.ext_ref && <span className="font-mono text-[10px] text-rose-400">EXT: {msg.ext_ref}</span>}
                    {!isSubRow && viewMode === 'flat' && <span className="font-mono text-[10px] text-slate-500 mt-1">Batch: {msg.batch_ref}</span>}
                </div>
            </td>
            <td className="px-6 py-4 font-mono text-sm">{msg.phone_number}</td>
            <td className="px-6 py-4 text-sm text-slate-400 truncate max-w-xs" title={msg.content}>
                {msg.content}
            </td>
            <td className="px-6 py-4"><StatusBadge status={msg.status} /></td>
            <td className="px-6 py-4 text-sm text-emerald-400 font-medium">${Number(msg.cost || 0).toFixed(3)}</td>
            <td className="px-6 py-4 text-xs text-slate-500 whitespace-nowrap">{formatDate(msg.created_at)}</td>
        </tr>
    );

    return (
        <div className="animate-in fade-in space-y-6">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                    <h2 className="text-3xl font-bold flex items-center gap-2">
                        <FileText className="w-8 h-8 text-rose-500" />
                        التقارير الموحدة
                    </h2>
                    <p className="text-slate-400 text-sm mt-1">عرض ومتابعة عمليات الإرسال الفردية والجماعية مع الأرقام المرجعية للأنظمة.</p>
                </div>
                
                <div className="flex items-center gap-3">
                    <div className="flex bg-slate-900 rounded-lg p-1 border border-slate-800">
                        <button
                            onClick={() => { setViewMode('flat'); setCurrentPage(1); }}
                            className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all ${viewMode === 'flat' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-white'}`}
                        >
                            <LayoutList className="w-4 h-4" />
                            رسائل فردية
                        </button>
                        <button
                            onClick={() => { setViewMode('grouped'); setCurrentPage(1); }}
                            className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all ${viewMode === 'grouped' ? 'bg-slate-800 text-white' : 'text-slate-400 hover:text-white'}`}
                        >
                            <Columns className="w-4 h-4" />
                            دفعات الإرسال (Batches)
                        </button>
                    </div>

                    <button onClick={handleExport} className="flex items-center gap-2 px-4 py-2 bg-rose-500/10 text-rose-400 hover:bg-rose-500/20 rounded-lg font-medium transition-colors border border-rose-500/20">
                        <Download className="w-4 h-4" /> تصدير
                    </button>
                </div>
            </div>

            {/* Summary Cards */}
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col justify-center">
                    <span className="text-slate-400 text-xs font-bold mb-1">الرصيد المستهلك (التكلفة)</span>
                    <span className="text-2xl font-black text-white">${Number(summary.total_cost || 0).toFixed(2)}</span>
                </div>
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col justify-center">
                    <span className="text-slate-400 text-xs font-bold mb-1">إجمالي الرسائل</span>
                    <span className="text-2xl font-black text-blue-400">{Number(summary.total_messages || 0).toLocaleString()}</span>
                </div>
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col justify-center">
                    <span className="flex items-center gap-1 text-emerald-400/80 text-xs font-bold mb-1">
                        <CheckCircle2 className="w-3 h-3" /> تم التوصيل
                    </span>
                    <span className="text-2xl font-black text-emerald-400">{Number(summary.delivered || 0).toLocaleString()}</span>
                </div>
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col justify-center">
                    <span className="flex items-center gap-1 text-slate-400 text-xs font-bold mb-1">
                        <Clock className="w-3 h-3" /> قيد المراسل
                    </span>
                    <span className="text-2xl font-black text-slate-200">{Number(summary.pending || 0).toLocaleString()}</span>
                </div>
                <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col justify-center">
                    <span className="flex items-center gap-1 text-rose-400/80 text-xs font-bold mb-1">
                        <XCircle className="w-3 h-3" /> فشل الإرسال
                    </span>
                    <span className="text-2xl font-black text-rose-400">{Number(summary.failed || 0).toLocaleString()}</span>
                </div>
            </div>

            {/* Smart Filters Bar */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
                <div className="flex flex-wrap items-end gap-4">
                    <div className="flex-1 min-w-[200px]">
                        <label className="block text-xs text-slate-400 font-bold mb-1">رقم عملية الإرسال (Batch Ref)</label>
                        <div className="relative">
                            <Search className="w-4 h-4 text-slate-500 absolute top-2.5 right-3" />
                            <input type="text" name="batch_ref" placeholder="بحث..." value={filters.batch_ref} onChange={handleFilterChange}
                                className="w-full bg-slate-950 border border-slate-800 rounded-lg py-2 pr-9 pl-3 text-sm focus:border-rose-500 focus:outline-none" />
                        </div>
                    </div>
                    {viewMode === 'flat' && (
                        <>
                            <div className="flex-1 min-w-[200px]">
                                <label className="block text-xs text-slate-400 font-bold mb-1">مرجع الرسالة (Msg / Ext Ref)</label>
                                <div className="relative">
                                    <Search className="w-4 h-4 text-slate-500 absolute top-2.5 right-3" />
                                    <input type="text" name={filters.msg_ref ? "msg_ref" : "ext_ref"} placeholder="رقم رسالة المنصة او رقم المحاسبة" value={filters.msg_ref || filters.ext_ref} onChange={(e) => {
                                        // A simple heuristic rule: if it starts with the company prefix it's msg_ref, otherwise ext_ref
                                        const val = e.target.value;
                                        if (val.includes('-M-')) {
                                            setFilters(p => ({ ...p, msg_ref: val, ext_ref: '' }));
                                        } else {
                                            setFilters(p => ({ ...p, ext_ref: val, msg_ref: '' }));
                                        }
                                    }}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-lg py-2 pr-9 pl-3 text-sm focus:border-rose-500 focus:outline-none" />
                                </div>
                            </div>
                            <div className="flex-1 min-w-[150px]">
                                <label className="block text-xs text-slate-400 font-bold mb-1">رقم المستلم</label>
                                <input type="text" name="phone_number" placeholder="مثال: +966..." value={filters.phone_number} onChange={handleFilterChange}
                                    className="w-full bg-slate-950 border border-slate-800 rounded-lg py-2 px-3 text-sm focus:border-rose-500 focus:outline-none" />
                            </div>
                        </>
                    )}
                    <div className="w-32">
                        <label className="block text-xs text-slate-400 font-bold mb-1">حالة الإرسال</label>
                        <select name="status" value={filters.status} onChange={handleFilterChange}
                            className="w-full bg-slate-950 border border-slate-800 rounded-lg py-2 px-2 text-sm focus:border-rose-500 focus:outline-none">
                            <option value="">جميع الحالات</option>
                            <option value="DELIVERED">تم التوصيل ✔️</option>
                            <option value="SENT">أرسل للشبكة</option>
                            <option value="FAILED">فشل ❌</option>
                            <option value="PENDING">قيد الانتظار 🕐</option>
                        </select>
                    </div>
                    <button onClick={clearFilters} className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium rounded-lg transition-colors">
                        إزالة الفلاتر
                    </button>
                </div>
            </div>

            {/* Main Data Table */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-right divide-y divide-slate-800">
                        <thead className="bg-slate-950/50">
                            {viewMode === 'flat' ? (
                                <tr>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">الأرقام المرجعية</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">المستلم</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">المحتوى</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">الحالة</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">التكلفة</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">التاريخ</th>
                                </tr>
                            ) : (
                                <tr>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider w-10"></th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">بيانات الدفعة (Batch)</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">النوع</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">الكمية</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">الحالة العامة</th>
                                    <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">تاريخ البدء</th>
                                </tr>
                            )}
                        </thead>
                        <tbody className="divide-y divide-slate-800">
                            {loading && data.length === 0 ? (
                                <tr><td colSpan="6" className="py-12 text-center text-slate-500"><Loader className="w-8 h-8 animate-spin mx-auto mb-2 text-rose-500" />جاري سحب البيانات...</td></tr>
                            ) : data.length === 0 ? (
                                <tr><td colSpan="6" className="py-12 text-center text-slate-500">لا يوجد بيانات تطابق البحث 📭</td></tr>
                            ) : viewMode === 'flat' ? (
                                data.map(msg => renderMessageRow(msg))
                            ) : (
                                data.map(batch => (
                                    <React.Fragment key={batch.id}>
                                        <tr className="hover:bg-slate-800/30 transition-colors cursor-pointer group" onClick={() => toggleBatchExpansion(batch.batch_ref)}>
                                            <td className="px-4 py-4 content-center opacity-50 group-hover:opacity-100 transition-opacity">
                                                <div className={`transform transition-transform ${expandedBatches.includes(batch.batch_ref) ? '-rotate-90' : 'rotate-180'}`}>
                                                    <ChevronLeft className="w-5 h-5 text-slate-400" />
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col">
                                                    <span className="font-mono text-sm text-white">{batch.batch_ref}</span>
                                                    <span className="text-xs text-slate-500 mt-0.5">Device: {batch.sent_from_device || 'System'}</span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <span className={`px-2 py-1 bg-slate-800 rounded-md text-xs font-bold ${batch.send_type === 'BULK' ? 'text-purple-400' : 'text-blue-400'}`}>
                                                    {batch.send_type}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 text-sm font-bold text-slate-300">{batch.total_messages}</td>
                                            <td className="px-6 py-4"><StatusBadge status={batch.status} /></td>
                                            <td className="px-6 py-4 text-xs text-slate-500">{formatDate(batch.created_at)}</td>
                                        </tr>
                                        {/* EXPANDED SECTION */}
                                        {expandedBatches.includes(batch.batch_ref) && (
                                            <tr className="bg-slate-950/30">
                                                <td colSpan="6" className="p-0 border-b-4 border-slate-900">
                                                    <div className="pr-12 pl-6 py-4 border-r-2 border-rose-500/30 bg-slate-900/20">
                                                        <span className="text-xs font-bold text-rose-500/80 mb-3 block">الرسائل داخل هذه الدفعة (أول 100 رسالة):</span>
                                                        {loadingBatches[batch.batch_ref] ? (
                                                            <div className="flex items-center gap-2 text-slate-500 text-sm py-4"><Loader className="w-4 h-4 animate-spin" /> جاري تحميل تفاصيل الدفعة...</div>
                                                        ) : (
                                                            <div className="overflow-hidden rounded-lg border border-slate-800 bg-slate-950">
                                                                <table className="w-full text-right divide-y divide-slate-800">
                                                                    <thead className="bg-slate-900/80">
                                                                        <tr>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">مرجع الرسالة</th>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">المستلم</th>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">المحتوى</th>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">الحالة</th>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">التקلفة</th>
                                                                            <th className="px-6 py-2 text-[10px] font-black text-slate-500 tracking-wider">الوقت</th>
                                                                        </tr>
                                                                    </thead>
                                                                    <tbody className="divide-y divide-slate-800/50">
                                                                        {batchDetails[batch.batch_ref]?.map(msg => renderMessageRow(msg, true))}
                                                                    </tbody>
                                                                </table>
                                                            </div>
                                                        )}
                                                    </div>
                                                </td>
                                            </tr>
                                        )}
                                    </React.Fragment>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>

                {/* Pagination Controls */}
                <div className="border-t border-slate-800 p-4 flex items-center justify-between bg-slate-950/20">
                    <span className="text-sm text-slate-500 font-medium">
                        إظهار الصفحة {currentPage} من {totalPages}
                    </span>
                    <div className="flex items-center gap-2">
                        <button onClick={() => setCurrentPage(p => Math.max(1, p - 1))} disabled={currentPage === 1 || loading}
                            className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-50 transition-colors">
                            <ChevronRight className="w-4 h-4 text-white" />
                        </button>
                        <button onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))} disabled={currentPage === totalPages || loading || totalPages === 0}
                            className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-50 transition-colors">
                            <ChevronLeft className="w-4 h-4 text-white" />
                        </button>
                    </div>
                </div>
            </div>
            
            {/* Developer API Hint Card */}
            <div className="bg-slate-900/80 border border-slate-800 rounded-xl p-4 flex items-start gap-4 mt-8">
                <div className="p-3 bg-blue-500/10 rounded-lg">
                    <MessageSquare className="w-6 h-6 text-blue-400" />
                </div>
                <div>
                    <h4 className="text-white font-bold text-sm">مزامنة الأنظمة المحاسبية (API)</h4>
                    <p className="text-slate-400 text-xs mt-1 max-w-2xl leading-relaxed">
                        نظام "الهدهد" يدعم الآن استعلامات مباشرة للأنظمة المحاسبية (ERP) بربط فوري لمعرفة حالة الرسائل.
                        يمكن لبرنامجكم المحاسبي التواصل مباشرة مع <code className="text-rose-400 bg-rose-500/10 px-1 py-0.5 rounded mx-1">/api/v1/accounting/messages</code> 
                        باستخدام <code className="text-blue-400 bg-blue-500/10 px-1 py-0.5 rounded mx-1">X-API-Key</code> لعكس حالة الإرسال والتكاليف تلقائياً في القيود المالية.
                    </p>
                </div>
            </div>
        </div>
    );
};

export default MessagesReport;
