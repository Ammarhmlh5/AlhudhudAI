import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowUpFromLine, Plus, X, Send, Undo2, Search, ShieldAlert } from 'lucide-react';
import { useAuth } from '../../contexts/AuthContext';
import { ACCOUNTING } from '../../api/accounting';

const STATUS_LABEL = { DRAFT: 'مسودة', POSTED: 'مرحّل', REVERSED: 'معكوس' };
const PAYEE_TYPES = [
  { value: 'OTHER', label: 'أخرى' },
  { value: 'TENANT', label: 'شركة/مستأجر' },
  { value: 'RESELLER', label: 'وكيل' },
  { value: 'SUPPLIER', label: 'مورد' },
];
const PAY_METHODS = [
  { value: 'CASH', label: 'نقدي' },
  { value: 'BANK_TRANSFER', label: 'تحويل بنكي' },
  { value: 'CHEQUE', label: 'شيك' },
  { value: 'CARD', label: 'بطاقة' },
  { value: 'WALLET', label: 'محفظة' },
];

export default function PaymentVouchers() {
  const { t } = useTranslation();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('approve_payments');
  const [vouchers, setVouchers] = useState([]);
  const [cashboxes, setCashboxes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [busyId, setBusyId] = useState(null);
  const [preview, setPreview] = useState('SARF-2026-000001');
  const [statusFilter, setStatusFilter] = useState('');
  const [form, setForm] = useState({
    voucher_date: new Date().toISOString().slice(0, 10),
    cashbox_id: '', amount: '', currency: 'SAR', payee_name: '',
    payee_type: 'OTHER', payment_method: 'CASH', reference_number: '',
    description: '', post_now: true,
  });

  useEffect(() => {
    fetchAll();
  }, []);

  useEffect(() => {
    if (!showModal) return;
    ACCOUNTING.previewNumber('PAYMENT').then(res => {
      setPreview(res.data?.data?.number || preview);
    }).catch(() => {});
  }, [showModal]);

  const fetchAll = async () => {
    try {
      setLoading(true);
      const [vRes, cbRes] = await Promise.all([
        ACCOUNTING.listPayments({ limit: 200 }),
        ACCOUNTING.listCashboxes(),
      ]);
      setVouchers(vRes.data?.data?.vouchers || []);
      setCashboxes(cbRes.data?.data || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.message || t('vouchers.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  const openCreate = () => {
    setForm({
      voucher_date: new Date().toISOString().slice(0, 10),
      cashbox_id: cashboxes[0]?.id || '', amount: '', currency: 'SAR',
      payee_name: '', payee_type: 'OTHER', payment_method: 'CASH',
      reference_number: '', description: '', post_now: true,
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      setSaving(true);
      await ACCOUNTING.createPayment(form);
      setShowModal(false);
      fetchAll();
    } catch (err) {
      alert(err.response?.data?.message || err.response?.data?.error || t('vouchers.save_failed'));
    } finally {
      setSaving(false);
    }
  };

  const post = async (id) => {
    try {
      setBusyId(id);
      await ACCOUNTING.postPayment(id);
      fetchAll();
    } catch (err) {
      alert(err.response?.data?.message || t('vouchers.post_failed'));
    } finally {
      setBusyId(null);
    }
  };

  const reverse = async (id) => {
    if (!window.confirm(t('vouchers.reverse_confirm'))) return;
    try {
      setBusyId(id);
      await ACCOUNTING.reversePayment(id);
      fetchAll();
    } catch (err) {
      alert(err.response?.data?.message || t('vouchers.reverse_failed'));
    } finally {
      setBusyId(null);
    }
  };

  const filtered = vouchers.filter(v => !statusFilter || v.status === statusFilter);
  const inputCls = "w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold";
  const labelCls = "text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1";
  const fmt = (v) => Number(v || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 });
  const cashboxName = (id) => cashboxes.find(c => c.id === id)?.name || '—';

  const statusBadge = (status) => {
    const cls = status === 'POSTED' ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
      : status === 'REVERSED' ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
      : 'bg-amber-500/10 text-amber-400 border-amber-500/20';
    return (
      <span className={`px-3 py-1.5 text-[10px] font-black rounded-xl border w-fit ${cls}`}>
        {STATUS_LABEL[status] || status}
      </span>
    );
  };

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700 notranslate">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-rose-600 rounded-2xl shadow-lg shadow-rose-900/20">
              <ArrowUpFromLine className="w-8 h-8 text-white" />
            </div>
            {t('vouchers.payments_title')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">{t('vouchers.payments_description')}</p>
        </div>
        {canWrite ? (
          <button onClick={openCreate} className="px-6 py-3 bg-rose-600 hover:bg-rose-700 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-rose-900/20">
            <Plus className="w-4 h-4" />
            {t('vouchers.add_payment')}
          </button>
        ) : (
          <div className="flex items-center gap-2 px-4 py-3 bg-amber-500/10 border border-amber-500/20 rounded-2xl text-amber-400 text-xs font-bold">
            <ShieldAlert className="w-4 h-4" />
            {t('permissions.read_only')}
          </div>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-3xl w-full shadow-2xl relative overflow-hidden max-h-[90vh] overflow-y-auto">
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-rose-600 via-emerald-600 to-rose-600"></div>
            <div className="flex justify-between items-center mb-8">
              <div>
                <h2 className="text-3xl font-black text-white tracking-tight">{t('vouchers.add_payment')}</h2>
                <p className="text-slate-500 text-sm font-bold uppercase tracking-widest mt-1">
                  {t('vouchers.next_number')}: <span className="text-rose-400 font-mono">{preview}</span>
                </p>
              </div>
              <button onClick={() => setShowModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                <X className="w-6 h-6" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.date')}</label>
                  <input type="date" required value={form.voucher_date} onChange={e => setForm({ ...form, voucher_date: e.target.value })} className={inputCls} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.cashbox')}</label>
                  <select required value={form.cashbox_id} onChange={e => setForm({ ...form, cashbox_id: e.target.value })} className={inputCls + " appearance-none cursor-pointer"}>
                    <option value="">{t('vouchers.select_cashbox')}</option>
                    {cashboxes.filter(c => c.status === 'ACTIVE').map(cb => (
                      <option key={cb.id} value={cb.id}>{cb.name} ({cb.code})</option>
                    ))}
                  </select>
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.amount')}</label>
                  <input type="number" step="0.0001" min="0.0001" required value={form.amount} onChange={e => setForm({ ...form, amount: e.target.value })} className={inputCls} placeholder="0.00" />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.payee_name')}</label>
                  <input required value={form.payee_name} onChange={e => setForm({ ...form, payee_name: e.target.value })} className={inputCls} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.payee_type')}</label>
                  <select value={form.payee_type} onChange={e => setForm({ ...form, payee_type: e.target.value })} className={inputCls + " appearance-none cursor-pointer"}>
                    {PAYEE_TYPES.map(pt => <option key={pt.value} value={pt.value}>{pt.label}</option>)}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.payment_method')}</label>
                  <select value={form.payment_method} onChange={e => setForm({ ...form, payment_method: e.target.value })} className={inputCls + " appearance-none cursor-pointer"}>
                    {PAY_METHODS.map(pm => <option key={pm.value} value={pm.value}>{pm.label}</option>)}
                  </select>
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.currency')}</label>
                  <input value={form.currency} onChange={e => setForm({ ...form, currency: e.target.value })} className={inputCls} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('vouchers.reference_number')}</label>
                  <input value={form.reference_number} onChange={e => setForm({ ...form, reference_number: e.target.value })} className={inputCls} />
                </div>
              </div>

              <div className="space-y-2">
                <label className={labelCls}>{t('vouchers.description')}</label>
                <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} className={inputCls} rows="2" />
              </div>

              <label className="flex items-center gap-3 cursor-pointer">
                <input type="checkbox" checked={form.post_now} onChange={e => setForm({ ...form, post_now: e.target.checked })} className="w-5 h-5 accent-rose-600" />
                <span className="text-sm font-bold text-slate-300">{t('vouchers.post_now')}</span>
              </label>

              <div className="flex gap-4 pt-6">
                <button type="submit" disabled={saving} className="flex-1 bg-rose-600 hover:bg-rose-500 disabled:bg-rose-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-rose-900/20">
                  {saving ? t('vouchers.saving') : t('vouchers.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        <div className="flex gap-3 flex-wrap">
          <div className="flex-1 min-w-[240px] relative">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
            <input type="text" placeholder={t('vouchers.search_placeholder')} className="w-full pl-12 pr-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white placeholder:text-slate-600 focus:outline-none focus:border-rose-500 transition-all shadow-inner" />
          </div>
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)} className="px-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold appearance-none cursor-pointer">
            <option value="">{t('vouchers.all_status')}</option>
            <option value="DRAFT">مسودة</option>
            <option value="POSTED">مرحّل</option>
            <option value="REVERSED">معكوس</option>
          </select>
        </div>
      </div>

      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-32 space-y-4">
            <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
            <p className="text-slate-400 font-bold animate-pulse">{t('vouchers.syncing')}</p>
          </div>
        ) : error ? (
          <div className="p-12 text-center">
            <div className="bg-rose-500/10 text-rose-400 border border-rose-500/20 rounded-2xl p-6 inline-block">
              <p className="font-bold">{error}</p>
            </div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-right border-collapse">
              <thead>
                <tr className="bg-slate-950/50 border-b border-slate-800">
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.voucher_number')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.date')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.cashbox')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.payee')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.amount')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('vouchers.status')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {filtered.map((v) => (
                  <tr key={v.id} className="hover:bg-slate-800/30 transition-all group">
                    <td className="px-8 py-6">
                      <div className="font-mono font-black text-white">{v.voucher_number}</div>
                      <div className="text-xs text-slate-500 mt-0.5">{v.description || '—'}</div>
                    </td>
                    <td className="px-8 py-6 text-sm font-bold text-slate-400">{new Date(v.voucher_date).toLocaleDateString()}</td>
                    <td className="px-8 py-6 text-sm font-bold text-slate-300">{cashboxName(v.cashbox_id)}</td>
                    <td className="px-8 py-6">
                      <div className="text-sm font-bold text-slate-300">{v.party_name}</div>
                      <div className="text-xs text-slate-500">{PAYEE_TYPES.find(p => p.value === v.party_type)?.label || v.party_type}</div>
                    </td>
                    <td className="px-8 py-6 text-sm font-mono font-black text-rose-400">{fmt(v.amount)} {v.currency}</td>
                    <td className="px-8 py-6">{statusBadge(v.status)}</td>
                    <td className="px-8 py-6">
                      <div className="flex gap-2 justify-end">
                        {canWrite && v.status === 'DRAFT' && (
                          <button onClick={() => post(v.id)} disabled={busyId === v.id} className="p-2 bg-slate-950 text-emerald-500 hover:text-white hover:bg-emerald-600 rounded-xl transition-all border border-slate-800 disabled:opacity-40" title={t('vouchers.post')}>
                            <Send className="w-5 h-5" />
                          </button>
                        )}
                        {canWrite && v.status === 'POSTED' && (
                          <button onClick={() => reverse(v.id)} disabled={busyId === v.id} className="p-2 bg-slate-950 text-rose-500 hover:text-white hover:bg-rose-600 rounded-xl transition-all border border-slate-800 disabled:opacity-40" title={t('vouchers.reverse')}>
                            <Undo2 className="w-5 h-5" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {filtered.length === 0 && !loading && (
        <div className="flex flex-col items-center justify-center py-24 bg-slate-900/50 border border-slate-800 border-dashed rounded-3xl">
          <ArrowUpFromLine className="w-16 h-16 text-slate-800 mb-4" />
          <p className="text-slate-500 font-bold text-lg">{t('vouchers.no_vouchers')}</p>
          {canWrite && (
            <button onClick={openCreate} className="mt-4 text-rose-500 font-bold hover:underline">{t('vouchers.add_first_payment')}</button>
          )}
        </div>
      )}
    </div>
  );
}
