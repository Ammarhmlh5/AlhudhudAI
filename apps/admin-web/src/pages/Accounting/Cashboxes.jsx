import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Wallet, Plus, X, Pencil, RotateCcw, Power, Landmark, Banknote, ShieldAlert, Download } from 'lucide-react';
import { useAuth } from '../../contexts/AuthContext';
import { ACCOUNTING } from '../../api/accounting';

const TYPE_META = {
  CASH: { label: 'نقدي', color: 'emerald', icon: Banknote },
  BANK: { label: 'بنكي', color: 'indigo', icon: Landmark },
  WALLET: { label: 'محفظة', color: 'amber', icon: Wallet },
  OTHER: { label: 'أخرى', color: 'slate', icon: Wallet },
};

const typeOptions = ['CASH', 'BANK', 'WALLET', 'OTHER'];

export default function Cashboxes() {
  const { t } = useTranslation();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('approve_payments');
  const [cashboxes, setCashboxes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState(null);
  const [saving, setSaving] = useState(false);
  const [statement, setStatement] = useState(null);
  const [form, setForm] = useState({
    code: '', name: '', name_en: '', type: 'CASH', currency: 'SAR',
    bank_name: '', account_holder: '', account_number: '', iban: '',
    opening_balance: '0', min_balance: '0', description: '',
  });

  useEffect(() => {
    fetchCashboxes();
  }, []);

  const fetchCashboxes = async () => {
    try {
      setLoading(true);
      const res = await ACCOUNTING.listCashboxes();
      setCashboxes(res.data?.data || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.message || t('cashboxes.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  const openCreate = () => {
    setEditing(null);
    setForm({
      code: '', name: '', name_en: '', type: 'CASH', currency: 'SAR',
      bank_name: '', account_holder: '', account_number: '', iban: '',
      opening_balance: '0', min_balance: '0', description: '',
    });
    setShowModal(true);
  };

  const openEdit = (cb) => {
    setEditing(cb);
    setForm({
      code: cb.code, name: cb.name, name_en: cb.name_en || '', type: cb.type,
      currency: cb.currency || 'SAR', bank_name: cb.bank_name || '',
      account_holder: cb.account_holder || '', account_number: cb.account_number || '',
      iban: cb.iban || '', opening_balance: String(cb.opening_balance || 0),
      min_balance: String(cb.min_balance || 0), description: cb.description || '',
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      setSaving(true);
      if (editing) {
        await ACCOUNTING.updateCashbox(editing.id, { ...form, name: form.name || editing.name, code: form.code || editing.code });
      } else {
        await ACCOUNTING.createCashbox(form);
      }
      setShowModal(false);
      fetchCashboxes();
    } catch (err) {
      alert(err.response?.data?.message || err.response?.data?.error || t('cashboxes.save_failed'));
    } finally {
      setSaving(false);
    }
  };

  const toggleStatus = async (cb) => {
    const next = cb.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE';
    try {
      await ACCOUNTING.setCashboxStatus(cb.id, next);
      fetchCashboxes();
    } catch (err) {
      alert(err.response?.data?.message || t('cashboxes.status_failed'));
    }
  };

  const openStatement = async (cb) => {
    try {
      const res = await ACCOUNTING.cashboxStatement(cb.id);
      setStatement(res.data?.data || null);
    } catch (err) {
      alert(err.response?.data?.message || t('cashboxes.statement_failed'));
    }
  };

  const exportStatementCSV = () => {
    if (!statement) return;
    const header = [t('cashboxes.movement_date'), t('cashboxes.voucher_number'), t('cashboxes.type'), t('cashboxes.description'), t('cashboxes.amount'), t('cashboxes.balance_after')];
    const rows = (statement.movements || []).map(mv => [
      new Date(mv.movement_date).toLocaleString(),
      mv.voucher_number,
      mv.voucher_type === 'RECEIPT' ? t('cashboxes.receipt') : t('cashboxes.payment'),
      (mv.description || '').replace(/"/g, '""'),
      mv.amount,
      mv.balance_after,
    ]);
    const csv = [header, ...rows].map(r => r.map(c => `"${c}"`).join(',')).join('\r\n');
    const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `cashbox_statement_${statement.cashbox?.code || 'box'}_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const formatAmount = (v) =>
    Number(v || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 });

  const inputCls = "w-full px-5 py-4 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-rose-500 transition-all font-bold";
  const labelCls = "text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1";

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700 notranslate">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-emerald-600 rounded-2xl shadow-lg shadow-emerald-900/20">
              <Wallet className="w-8 h-8 text-white" />
            </div>
            {t('cashboxes.title')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">{t('cashboxes.description')}</p>
        </div>
        {canWrite ? (
          <button onClick={openCreate} className="px-6 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-emerald-900/20">
            <Plus className="w-4 h-4" />
            {t('cashboxes.add')}
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
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-3xl w-full shadow-2xl relative overflow-hidden">
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-emerald-600 via-rose-600 to-emerald-600"></div>
            <div className="flex justify-between items-center mb-8">
              <div>
                <h2 className="text-3xl font-black text-white tracking-tight">{editing ? t('cashboxes.edit') : t('cashboxes.add')}</h2>
                <p className="text-slate-500 text-sm font-bold uppercase tracking-widest mt-1">{t('cashboxes.form_subtitle')}</p>
              </div>
              <button onClick={() => setShowModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                <X className="w-6 h-6" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.code')}</label>
                  <input required value={form.code} onChange={e => setForm({ ...form, code: e.target.value })} className={inputCls} placeholder="CASH-01" />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.type')}</label>
                  <select value={form.type} onChange={e => setForm({ ...form, type: e.target.value })} className={inputCls + " appearance-none cursor-pointer"}>
                    {typeOptions.map(opt => (
                      <option key={opt} value={opt}>{TYPE_META[opt].label}</option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.name')}</label>
                  <input required value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} className={inputCls} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.name_en')}</label>
                  <input value={form.name_en} onChange={e => setForm({ ...form, name_en: e.target.value })} className={inputCls} />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.currency')}</label>
                  <input value={form.currency} onChange={e => setForm({ ...form, currency: e.target.value })} className={inputCls} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.opening_balance')}</label>
                  <input disabled={!!editing} value={form.opening_balance} onChange={e => setForm({ ...form, opening_balance: e.target.value })} className={inputCls + " disabled:opacity-40"} />
                </div>
                <div className="space-y-2">
                  <label className={labelCls}>{t('cashboxes.min_balance')}</label>
                  <input value={form.min_balance} onChange={e => setForm({ ...form, min_balance: e.target.value })} className={inputCls} />
                </div>
              </div>

              {form.type === 'BANK' && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <label className={labelCls}>{t('cashboxes.bank_name')}</label>
                    <input value={form.bank_name} onChange={e => setForm({ ...form, bank_name: e.target.value })} className={inputCls} />
                  </div>
                  <div className="space-y-2">
                    <label className={labelCls}>{t('cashboxes.account_holder')}</label>
                    <input value={form.account_holder} onChange={e => setForm({ ...form, account_holder: e.target.value })} className={inputCls} />
                  </div>
                  <div className="space-y-2">
                    <label className={labelCls}>{t('cashboxes.account_number')}</label>
                    <input value={form.account_number} onChange={e => setForm({ ...form, account_number: e.target.value })} className={inputCls} />
                  </div>
                  <div className="space-y-2">
                    <label className={labelCls}>IBAN</label>
                    <input value={form.iban} onChange={e => setForm({ ...form, iban: e.target.value })} className={inputCls} />
                  </div>
                </div>
              )}

              <div className="space-y-2">
                <label className={labelCls}>{t('cashboxes.description')}</label>
                <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} className={inputCls} rows="2" />
              </div>

              <div className="flex gap-4 pt-6">
                <button type="submit" disabled={saving} className="flex-1 bg-emerald-600 hover:bg-emerald-500 disabled:bg-emerald-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-emerald-900/20">
                  {saving ? t('cashboxes.saving') : (editing ? t('cashboxes.update') : t('cashboxes.create'))}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {statement && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
          <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-4xl w-full shadow-2xl relative overflow-hidden max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-center mb-6">
              <div>
                <h2 className="text-3xl font-black text-white tracking-tight">{t('cashboxes.statement_title')}</h2>
                <p className="text-slate-400 text-sm font-bold mt-1">{statement.cashbox?.name} — {t('cashboxes.balance_label')}: <span className="text-emerald-400 font-mono">{formatAmount(statement.cashbox?.current_balance)}</span></p>
              </div>
              <div className="flex items-center gap-3">
                <button onClick={exportStatementCSV} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-emerald-400 hover:text-white hover:bg-emerald-600 transition-all" title={t('cashboxes.export_csv')}>
                  <Download className="w-6 h-6" />
                </button>
                <button onClick={() => setStatement(null)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                  <X className="w-6 h-6" />
                </button>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-2xl">
                <div className="text-xs font-black text-emerald-400 uppercase tracking-widest">{t('cashboxes.total_receipts')}</div>
                <div className="text-xl font-black text-white font-mono mt-1">{formatAmount(statement.totals?.total_receipts)}</div>
              </div>
              <div className="p-4 bg-rose-500/10 border border-rose-500/20 rounded-2xl">
                <div className="text-xs font-black text-rose-400 uppercase tracking-widest">{t('cashboxes.total_payments')}</div>
                <div className="text-xl font-black text-white font-mono mt-1">{formatAmount(statement.totals?.total_payments)}</div>
              </div>
              <div className="p-4 bg-indigo-500/10 border border-indigo-500/20 rounded-2xl">
                <div className="text-xs font-black text-indigo-400 uppercase tracking-widest">{t('cashboxes.net')}</div>
                <div className="text-xl font-black text-white font-mono mt-1">{formatAmount(statement.totals?.net)}</div>
              </div>
            </div>

            <div className="overflow-x-auto max-h-96 overflow-y-auto">
              <table className="w-full text-right border-collapse">
                <thead>
                  <tr className="bg-slate-950/50 border-b border-slate-800">
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.movement_date')}</th>
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.voucher_number')}</th>
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.type')}</th>
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.description')}</th>
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.amount')}</th>
                    <th className="px-6 py-4 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.balance_after')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/50">
                  {(statement.movements || []).map((mv) => (
                    <tr key={mv.id} className="hover:bg-slate-800/30 transition-all">
                      <td className="px-6 py-4 text-sm font-bold text-slate-400">{new Date(mv.movement_date).toLocaleDateString()}</td>
                      <td className="px-6 py-4 text-sm font-mono font-bold text-white">{mv.voucher_number}</td>
                      <td className="px-6 py-4">
                        <span className={`px-2.5 py-1 text-[10px] font-black rounded-lg ${
                          mv.voucher_type === 'RECEIPT' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'
                        }`}>
                          {mv.voucher_type === 'RECEIPT' ? t('cashboxes.receipt') : t('cashboxes.payment')}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-slate-300">{mv.description || '—'}</td>
                      <td className={`px-6 py-4 text-sm font-mono font-black ${mv.amount >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                        {mv.amount >= 0 ? '+' : ''}{formatAmount(mv.amount)}
                      </td>
                      <td className="px-6 py-4 text-sm font-mono font-bold text-white">{formatAmount(mv.balance_after)}</td>
                    </tr>
                  ))}
                  {(statement.movements || []).length === 0 && (
                    <tr><td colSpan="6" className="px-6 py-12 text-center text-slate-500 font-bold">{t('cashboxes.no_movements')}</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-32 space-y-4">
            <div className="w-12 h-12 border-4 border-emerald-500/20 border-t-emerald-500 rounded-full animate-spin"></div>
            <p className="text-slate-400 font-bold animate-pulse">{t('cashboxes.syncing')}</p>
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
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.name')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.type')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.balance')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.min_balance')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest">{t('cashboxes.status')}</th>
                  <th className="px-8 py-5 text-xs font-black text-slate-500 uppercase tracking-widest"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {cashboxes.map((cb) => {
                  const meta = TYPE_META[cb.type] || TYPE_META.OTHER;
                  const Icon = meta.icon;
                  return (
                    <tr key={cb.id} className="hover:bg-slate-800/30 transition-all group">
                      <td className="px-8 py-6">
                        <div className="flex items-center gap-4">
                          <div className={`w-12 h-12 rounded-2xl flex items-center justify-center border bg-slate-950 group-hover:border-rose-500/30 transition-colors`}>
                            <Icon className={`w-6 h-6 text-${meta.color}-500`} />
                          </div>
                          <div>
                            <div className="text-base font-bold text-white">{cb.name}</div>
                            <div className="text-xs text-slate-500 font-mono mt-0.5">{cb.code}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-8 py-6 text-sm font-bold text-slate-300">{meta.label}</td>
                      <td className="px-8 py-6">
                        <span className={`text-sm font-mono font-black ${Number(cb.current_balance) < Number(cb.min_balance) ? 'text-rose-400' : 'text-emerald-400'}`}>
                          {formatAmount(cb.current_balance)} {cb.currency}
                        </span>
                      </td>
                      <td className="px-8 py-6 text-sm font-mono font-bold text-slate-400">{formatAmount(cb.min_balance)}</td>
                      <td className="px-8 py-6">
                        <span className={`px-3 py-1.5 text-[10px] font-black uppercase rounded-xl border flex items-center gap-1.5 w-fit ${
                          cb.status === 'ACTIVE'
                            ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                            : 'bg-slate-500/10 text-slate-400 border-slate-500/20'
                        }`}>
                          {cb.status === 'ACTIVE' ? t('cashboxes.active') : t('cashboxes.inactive')}
                        </span>
                      </td>
                      <td className="px-8 py-6">
                        <div className="flex gap-2 justify-end">
                          <button onClick={() => openStatement(cb)} className="p-2 bg-slate-950 text-slate-400 hover:text-white hover:bg-indigo-600 rounded-xl transition-all border border-slate-800" title={t('cashboxes.statement')}>
                            <RotateCcw className="w-5 h-5" />
                          </button>
                          {canWrite && (
                            <>
                              <button onClick={() => openEdit(cb)} className="p-2 bg-slate-950 text-slate-400 hover:text-white hover:bg-emerald-600 rounded-xl transition-all border border-slate-800" title={t('cashboxes.edit')}>
                                <Pencil className="w-5 h-5" />
                              </button>
                              <button onClick={() => toggleStatus(cb)} className="p-2 bg-slate-950 text-slate-400 hover:text-white hover:bg-amber-600 rounded-xl transition-all border border-slate-800" title={t('cashboxes.toggle_status')}>
                                <Power className="w-5 h-5" />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {cashboxes.length === 0 && !loading && (
        <div className="flex flex-col items-center justify-center py-24 bg-slate-900/50 border border-slate-800 border-dashed rounded-3xl">
          <Wallet className="w-16 h-16 text-slate-800 mb-4" />
          <p className="text-slate-500 font-bold text-lg">{t('cashboxes.no_cashboxes')}</p>
          {canWrite && (
            <button onClick={openCreate} className="mt-4 text-emerald-500 font-bold hover:underline">{t('cashboxes.add_first')}</button>
          )}
        </div>
      )}
    </div>
  );
}
