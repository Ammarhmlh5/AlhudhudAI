import React, { useState, useEffect } from 'react';
import { CheckCircle, XCircle, Clock, RefreshCw, Download, ChevronRight, ChevronLeft, RotateCcw, Trash2, Loader2 } from 'lucide-react';
import apiClient from '../api/axios';
import { useSocket } from '../contexts/SocketContext';

// Helper: show toast (will disappear after 3s)
const useToast = () => {
  const [toast, setToast] = useState(null);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(null), 3000);
    return () => clearTimeout(t);
  }, [toast]);
  const show = (type, text) => setToast({ type, text });
  return { toast, show };
};

const MessageHistory = () => {
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('ALL');
  const [page, setPage] = useState(1);
  const [pageSize] = useState(25);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [rowAction, setRowAction] = useState({}); // {msgId: 'resend'|'delete'}
  const { toast, show } = useToast();
  const { messageUpdateNonce } = useSocket();

  useEffect(() => {
    fetchHistory();
  }, [filter, page, messageUpdateNonce]);

  const fetchHistory = async () => {
    setLoading(true);
    try {
      const statusQuery = filter !== 'ALL' ? `&status=${filter}` : '';
      const res = await apiClient.get(`/api/v1/messaging/list?page=${page}&page_size=${pageSize}${statusQuery}`);
      const data = res.data;
      setMessages(data.messages || []);
      setTotalCount(data.total_count || 0);
      setTotalPages(data.total_pages || 1);
    } catch (err) {
      console.error('Error fetching history:', err);
      show('error', 'فشل تحميل السجل');
    } finally {
      setLoading(false);
    }
  };

  const handleExportCSV = () => {
    if (!messages || messages.length === 0) { alert('لا توجد سجلات للتصدير'); return; }
    const headers = ['الحالة', 'رقم الهاتف', 'المحتوى', 'التاريخ'];
    const rows = messages.map(m => {
      const date = new Date(m.created_at).toLocaleString('ar-SA');
      const content = `"${(m.content || '').replace(/"/g, '""')}"`;
      return [m.status, m.phone_number, content, date].join(',');
    });
    const csv = 'data:text/csv;charset=utf-8,\uFEFF' + [headers.join(','), ...rows].join('\n');
    const a = document.createElement('a');
    a.href = encodeURI(csv);
    a.download = `message_history_${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const setAction = (id, act) => setRowAction(prev => ({ ...prev, [id]: act }));

  const handleResend = async (msg) => {
    setAction(msg.id, 'resend');
    try {
      await apiClient.post('/api/v1/messaging/bulk-resend', { ids: [msg.id] });
      show('success', `تمت إعادة الإرسال إلى ${msg.phone_number}`);
      fetchHistory();
    } catch (err) {
      console.error('Resend error:', err);
      show('error', 'فشل إعادة الإرسال');
    } finally {
      setAction(msg.id, null);
    }
  };

  const handleDelete = async (msg) => {
    if (!window.confirm(`هل تريد حذف الرسالة إلى ${msg.phone_number}?`)) return;
    setAction(msg.id, 'delete');
    try {
      await apiClient.post('/api/v1/messaging/bulk-delete', { ids: [msg.id] });
      show('success', 'تم حذف الرسالة');
      setMessages(prev => prev.filter(m => m.id !== msg.id));
    } catch (err) {
      console.error('Delete error:', err);
      show('error', 'فشل الحذف');
    } finally {
      setAction(msg.id, null);
    }
  };

  return (
    <div className="relative animate-in fade-in slide-in-from-right-8 duration-500">
      {/* Toast */}
      {toast && (
        <div className={`fixed top-6 left-1/2 -translate-x-1/2 z-50 px-5 py-3 rounded-2xl text-sm font-bold flex items-center gap-2 transition-all ${toast.type === 'success' ? 'bg-emerald-500/90 text-white' : 'bg-rose-500/90 text-white'}`}>
          {toast.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
          {toast.text}
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-8">
        <div>
          <h2 className="text-3xl font-bold">الأرشيف</h2>
          <p className="text-slate-400 text-sm mt-1">سجل جميع الرسائل المرسلة</p>
        </div>
        <div className="flex gap-2 flex-wrap">
          <select className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-2 text-sm focus:outline-none focus:border-rose-500" value={filter} onChange={e => { setFilter(e.target.value); setPage(1); }}>
            <option value="ALL">جميع الحالات</option>
            <option value="DELIVERED">تم التوصيل</option>
            <option value="SENT">مُرسَل</option>
            <option value="FAILED">فشل</option>
            <option value="WAITING">انتظار</option>
            <option value="PENDING">قيد الانتظار</option>
          </select>
          <button onClick={handleExportCSV} disabled={loading || messages.length === 0} className="px-4 py-2 bg-slate-800 rounded-xl hover:bg-slate-700 transition-colors flex items-center gap-2 text-sm font-bold text-slate-300 disabled:opacity-50">
            <Download className="w-4 h-4 text-emerald-400" /> تصدير CSV
          </button>
          <button onClick={fetchHistory} className="p-2 bg-slate-800 rounded-xl hover:bg-slate-700 transition-colors" title="تحديث">
            <RefreshCw className={`w-5 h-5 text-slate-400 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-right">
            <thead className="bg-slate-950/50">
              <tr>
                <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">الحالة</th>
                <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">المستلم</th>
                <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">المحتوى</th>
                <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider">التاريخ</th>
                <th className="px-6 py-4 text-xs font-black text-slate-400 tracking-wider text-center">إجراء</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {loading && messages.length === 0 ? (
                <tr><td colSpan="5" className="px-6 py-10 text-center text-slate-500">جاري التحميل...</td></tr>
              ) : messages.length === 0 ? (
                <tr><td colSpan="5" className="px-6 py-10 text-center text-slate-500">لا توجد سجلات</td></tr>
              ) : (
                messages.map(msg => {
                  const busy = rowAction[msg.id];
                  return (
                    <tr key={msg.id} className="hover:bg-slate-800/30 transition-colors">
                      <td className="px-6 py-5">
                        <div className="flex items-center gap-2">
                          {msg.status === 'DELIVERED' && <CheckCircle className="w-4 h-4 text-emerald-500" />}
                          {msg.status === 'FAILED' && <XCircle className="w-4 h-4 text-rose-500" />}
                          {msg.status === 'WAITING' && <Clock className="w-4 h-4 text-blue-500 animate-pulse" />}
                          {msg.status === 'PENDING' && <Clock className="w-4 h-4 text-amber-500" />}
                          <span className={`text-xs font-bold ${
                            msg.status === 'DELIVERED' ? 'text-emerald-500' : 
                            msg.status === 'FAILED' ? 'text-rose-500' : 
                            msg.status === 'WAITING' ? 'text-blue-500 animate-pulse' : 
                            'text-amber-500'
                          }`}>{msg.status}</span>
                        </div>
                      </td>
                      <td className="px-6 py-5 font-mono text-sm text-slate-200">{msg.phone_number}</td>
                      <td className="px-6 py-5 text-sm text-slate-400 max-w-xs truncate" title={msg.content}>{msg.content}</td>
                      <td className="px-6 py-5 text-xs text-slate-500 font-mono">{new Date(msg.created_at).toLocaleString('ar-SA')}</td>
                      <td className="px-6 py-5 text-center">
                        <div className="flex items-center justify-center gap-2">
                          {['FAILED', 'PENDING', 'WAITING'].includes(msg.status) && (
                            <button onClick={() => handleResend(msg)} disabled={!!busy} className="flex items-center gap-1 px-3 py-1.5 rounded-xl text-xs font-bold bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20 hover:border-blue-400/40 disabled:opacity-40 disabled:cursor-not-allowed transition-all" title="إعادة إرسال">
                              {busy === 'resend' ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RotateCcw className="w-3.5 h-3.5" />} إعادة إرسال
                            </button>
                          )}
                          <button onClick={() => handleDelete(msg)} disabled={!!busy} className="flex items-center gap-1 px-3 py-1.5 rounded-xl text-xs font-bold bg-rose-500/10 text-rose-400 border border-rose-500/20 hover:bg-rose-500/20 hover:border-rose-400/40 disabled:opacity-40 disabled:cursor-not-allowed transition-all" title="حذف">
                            {busy === 'delete' ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />} حذف
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        {/* Pagination */}
        <div className="border-t border-slate-800 p-4 flex items-center justify-between bg-slate-950/20">
          <span className="text-sm text-slate-500">{loading ? '...' : `صفحة ${page} من ${totalPages} (${totalCount.toLocaleString()} رسالة)`}</span>
          <div className="flex gap-2">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1 || loading} className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-colors" title="الصفحة السابقة">
              <ChevronRight className="w-4 h-4 text-white" />
            </button>
            <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages || loading} className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-colors" title="الصفحة التالية">
              <ChevronLeft className="w-4 h-4 text-white" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default MessageHistory;
