import { useState, useEffect, useCallback } from 'react';
import {
  Wallet, TrendingUp, Download, Search, CheckCircle, Clock,
  Building, CreditCard, Settings, Plus, Trash2, AlertCircle
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { financeAPI } from '../api/finance';

export default function FinanceManager() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Data states
  const [transactions, setTransactions] = useState([]);
  const [banks, setBanks] = useState([]);
  const [gateways, setGateways] = useState([]);
  const [stats, setStats] = useState({
    totalRevenue: 0,
    pendingApprovals: 0,
    avgRevenue: 0,
    weeklyGrowth: '+12%'
  });

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    try {
      const response = await financeAPI.getTransactions({ limit: 20 });
      setTransactions(response.data || []);
    } catch (err) {
      console.error('Failed to fetch transactions:', err);
      // Use mock data on error for development
      setTransactions([
        { id: 'TRX-9921', tenant: 'مؤسسة النور للتقنية', amount: 5000, type: 'حوالة بنكية', status: 'Pending', date: 'منذ ١٠ دقائق' },
        { id: 'TRX-9920', tenant: 'بوتيك سارة', amount: 150, type: 'بطاقة ائتمانية', status: 'Success', date: 'منذ ساعتين' },
        { id: 'TRX-9919', tenant: 'مستوصف الحياة', amount: 2500, type: 'حوالة بنكية', status: 'Success', date: 'بالأمس' },
      ]);
    }
  }, []);

  // Fetch cashboxes (banks)
  const fetchCashboxes = useCallback(async () => {
    try {
      const response = await financeAPI.getCashboxes();
      setBanks(response.data || []);
    } catch (err) {
      console.error('Failed to fetch cashboxes:', err);
      // Use mock data on error
      setBanks([
        { id: 1, bank: 'مصرف الراجحي', iban: 'SA12 8000 0000 6080 1010 1234', name: 'مؤسسة الهدهد للتقنية', status: 'Active' },
        { id: 2, bank: 'البنك الأهلي', iban: 'SA55 1000 0000 4444 3333 2211', name: 'مؤسسة الهدهد للتقنية', status: 'Active' },
      ]);
    }
  }, []);

  // Fetch data on mount and tab change
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        await Promise.all([
          fetchTransactions(),
          fetchCashboxes()
        ]);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [fetchTransactions, fetchCashboxes]);

  // Handle transaction approval
  const handleApprove = async (transactionId) => {
    try {
      await financeAPI.reverseTransaction({ id: transactionId, action: 'approve' });
      await fetchTransactions();
    } catch (err) {
      console.error('Failed to approve transaction:', err);
    }
  };

  // Create cashbox handler
  const handleCreateCashbox = async (data) => {
    try {
      await financeAPI.createCashbox(data);
      await fetchCashboxes();
    } catch (err) {
      console.error('Failed to create cashbox:', err);
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <div className="w-12 h-12 border-4 border-rose-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
            <p className="text-slate-400">{t('loading')}</p>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="space-y-6">
        <div className="bg-rose-500/10 border border-rose-500/50 rounded-3xl p-6 text-center">
          <AlertCircle className="w-12 h-12 text-rose-500 mx-auto mb-4" />
          <h3 className="text-xl font-bold text-rose-400 mb-2">حدث خطأ</h3>
          <p className="text-slate-400">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">

      {/* Module Header & Tabs */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white flex items-center gap-2">
            <Wallet className="w-6 h-6 text-rose-500" />
            {t('financeManager.title')}
          </h2>
          <p className="text-slate-400 text-sm mt-1">{t('financeManager.description')}</p>
        </div>
        <div className="flex bg-slate-900 p-1 rounded-xl border border-slate-800">
          <button
            onClick={() => setActiveTab('overview')}
            className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'overview' ? 'bg-slate-800 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}
          >
            {t('financeManager.tab_overview')}
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${activeTab === 'settings' ? 'bg-slate-800 text-white shadow-lg' : 'text-slate-500 hover:text-slate-300'}`}
          >
            {t('financeManager.tab_settings')}
          </button>
        </div>
      </div>

      {activeTab === 'overview' ? (
        <>
          {/* Stats Cards */}
          <div className="flex flex-col md:flex-row gap-4 mb-8">
            <div className="flex-1 bg-gradient-to-br from-emerald-600 to-teal-600 rounded-3xl p-6 text-white shadow-lg shadow-emerald-900/20">
              <div className="flex justify-between items-start mb-8">
                <div className="p-3 bg-white/20 rounded-2xl">
                  <Wallet className="w-6 h-6 text-white" />
                </div>
                <span className="text-xs font-bold bg-black/20 px-2 py-1 rounded-lg">{stats.weeklyGrowth} {t('financeManager.weekly_growth')}</span>
              </div>
              <p className="text-emerald-100 text-sm font-medium">{t('financeManager.total_revenue')}</p>
              <h3 className="text-4xl font-bold mt-1">${stats.totalRevenue.toLocaleString()}</h3>
            </div>

            <div className="flex-1 bg-slate-900 border border-slate-800 rounded-3xl p-6">
              <div className="flex justify-between items-start mb-8">
                <div className="p-3 bg-rose-500/10 rounded-2xl">
                  <Clock className="w-6 h-6 text-rose-500" />
                </div>
              </div>
              <p className="text-slate-400 text-sm font-medium">{t('financeManager.pending_approvals')}</p>
              <h3 className="text-4xl font-bold mt-1 text-white">{t('financeManager.requests', { count: stats.pendingApprovals })}</h3>
            </div>

            <div className="flex-1 bg-slate-900 border border-slate-800 rounded-3xl p-6">
              <div className="flex justify-between items-start mb-8">
                <div className="p-3 bg-blue-500/10 rounded-2xl">
                  <TrendingUp className="w-6 h-6 text-blue-500" />
                </div>
              </div>
              <p className="text-slate-400 text-sm font-medium">{t('financeManager.avg_revenue_per_user')}</p>
              <h3 className="text-4xl font-bold mt-1 text-white">${stats.avgRevenue.toFixed(2)}</h3>
            </div>
          </div>

          {/* Transactions Table */}
          <div className="glass-card border border-slate-800 bg-slate-900/40 rounded-3xl overflow-hidden">
            <div className="p-6 border-b border-slate-800 flex justify-between items-center">
              <h3 className="text-lg font-bold text-white">{t('financeManager.recent_transactions')}</h3>
              <div className="flex gap-2">
                <button className="p-2 bg-slate-800 text-slate-400 hover:text-white rounded-lg transition-colors">
                  <Search className="w-4 h-4" />
                </button>
                <button
                  onClick={() => financeAPI.getTransactions({ format: 'csv' })}
                  className="p-2 bg-slate-800 text-slate-400 hover:text-white rounded-lg transition-colors"
                >
                  <Download className="w-4 h-4" />
                </button>
              </div>
            </div>
            <div className="overflow-x-auto w-full">
              <table className="w-full text-left">
                <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                  <tr>
                    <th className="px-6 py-4">{t('financeManager.col_transaction_id')}</th>
                    <th className="px-6 py-4">{t('financeManager.col_tenant')}</th>
                    <th className="px-6 py-4">{t('financeManager.col_amount')}</th>
                    <th className="px-6 py-4">{t('financeManager.col_method')}</th>
                    <th className="px-6 py-4">{t('financeManager.col_status')}</th>
                    <th className="px-6 py-4 text-right">{t('financeManager.col_action')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800 text-slate-300">
                  {transactions.map((trx) => (
                    <tr key={trx.id} className="hover:bg-slate-800/30 transition-colors">
                      <td className="px-6 py-4 font-mono text-xs text-slate-500">{trx.id}</td>
                      <td className="px-6 py-4 font-bold text-white">{trx.tenant || trx.payer_name || trx.customer_name}</td>
                      <td className="px-6 py-4 text-emerald-400 font-mono font-bold">+${Number(trx.amount).toFixed(2)}</td>
                      <td className="px-6 py-4 text-sm">{trx.type || trx.payment_method || '-'}</td>
                      <td className="px-6 py-4">
                        <span className={`flex items-center gap-1.5 text-xs font-bold ${trx.status === 'Success' || trx.status === 'POSTED' ? 'text-emerald-400' : 'text-amber-400'}`}>
                          {(trx.status === 'Success' || trx.status === 'POSTED') ? <CheckCircle className="w-3 h-3" /> : <Clock className="w-3 h-3" />}
                          {trx.status === 'Success' || trx.status === 'POSTED' ? t('financeManager.status_success') : t('financeManager.status_pending')}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-right">
                        {(trx.status === 'Pending' || trx.status === 'PENDING') && (
                          <button
                            onClick={() => handleApprove(trx.id)}
                            className="px-3 py-1 bg-rose-600 hover:bg-rose-500 text-white text-xs font-bold rounded-lg transition-colors"
                          >
                            {t('financeManager.approve')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                  {transactions.length === 0 && (
                    <tr>
                      <td colSpan="6" className="px-6 py-8 text-center text-slate-500">
                        لا توجد معاملات
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 animate-in fade-in slide-in-from-right-4">

          {/* Bank Accounts Section */}
          <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden p-6">
            <div className="flex justify-between items-center mb-6">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Building className="w-5 h-5 text-rose-500" />
                {t('financeManager.bank_accounts')}
              </h3>
              <button className="flex items-center gap-2 text-xs font-bold bg-slate-800 hover:bg-slate-700 text-white px-3 py-2 rounded-lg transition-colors">
                <Plus className="w-3 h-3" /> {t('financeManager.add_bank')}
              </button>
            </div>
            <div className="space-y-4">
              {banks.map((bank) => (
                <div key={bank.id} className="p-4 rounded-2xl bg-slate-950 border border-slate-800 flex justify-between items-center group">
                  <div className="flex gap-4">
                    <div className="w-10 h-10 rounded-full bg-slate-800 flex items-center justify-center text-slate-400">
                      <Building className="w-5 h-5" />
                    </div>
                    <div>
                      <h4 className="font-bold text-white text-sm">{bank.bank || bank.bank_name}</h4>
                      <p className="text-[10px] sm:text-xs text-slate-500 font-mono mt-0.5">{bank.iban}</p>
                      <p className="text-[10px] text-slate-600 mt-1">{bank.name}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] bg-emerald-500/10 text-emerald-400 px-2 py-0.5 rounded font-bold">
                      {bank.status === 'Active' ? t('financeManager.active') : bank.status}
                    </span>
                    <button className="p-2 text-slate-600 hover:text-rose-500 transition-colors opacity-0 group-hover:opacity-100">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
              {banks.length === 0 && (
                <p className="text-slate-500 text-center py-4">لا توجد صناديق</p>
              )}
            </div>
            <div className="mt-6 bg-slate-950/50 p-4 rounded-xl text-xs text-slate-500 border border-slate-800/50">
              <span className="text-rose-400 font-bold block mb-1">{t('financeManager.important')}</span>
              {t('financeManager.bank_note')}
            </div>
          </div>

          {/* Electronic Wallets & Gateways */}
          <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden p-6">
            <div className="flex justify-between items-center mb-6">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <CreditCard className="w-5 h-5 text-rose-500" />
                {t('financeManager.payment_gateways')}
              </h3>
            </div>
            <div className="space-y-4">
              {/* Mock gateway data - replace with API when available */}
              {[
                { id: 'stc', name: 'STC Pay', type: 'Wallet', active: true },
                { id: 'urpay', name: 'UrPay', type: 'Wallet', active: false },
                { id: 'moyasar', name: 'Moyasar', type: 'Gateway', active: true },
                { id: 'stripe', name: 'Stripe', type: 'Gateway', active: false },
              ].map((gw) => (
                <div key={gw.id} className="p-4 rounded-2xl bg-slate-950 border border-slate-800 flex justify-between items-center">
                  <div className="flex gap-4 items-center">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center border ${gw.active ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-500' : 'bg-slate-800 border-slate-700 text-slate-500'}`}>
                      <CreditCard className="w-5 h-5" />
                    </div>
                    <div>
                      <h4 className="font-bold text-white text-sm">{gw.name}</h4>
                      <p className="text-[10px] text-slate-500 uppercase font-bold">{gw.type}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <button className="text-xs text-slate-400 hover:text-white underline">{t('financeManager.configure')}</button>
                    <div
                      className={`w-12 h-6 rounded-full p-1 cursor-pointer transition-colors ${gw.active ? 'bg-emerald-500' : 'bg-slate-700'}`}
                    >
                      <div className={`w-4 h-4 bg-white rounded-full shadow-sm transition-transform ${gw.active ? 'translate-x-6' : 'translate-x-0'}`} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-6">
              <button className="w-full py-3 rounded-xl border border-dashed border-slate-700 text-slate-500 text-sm font-bold hover:bg-slate-800 hover:text-white transition-colors">
                + {t('financeManager.add_provider')}
              </button>
            </div>
          </div>

        </div>
      )}

    </div>
  );
}