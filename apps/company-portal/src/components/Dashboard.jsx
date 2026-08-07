import { useEffect, useState, useCallback } from 'react';
import {
  Building2,
  Wallet,
  MessageSquare,
  Package,
  RefreshCcw,
  Check,
  X,
  LogOut,
  Receipt,
} from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import apiClient from '../api/axios';

function formatDate(value) {
  if (!value) return '—';
  return new Date(value).toLocaleDateString('ar-EG', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export default function Dashboard() {
  const { user, tenant, logout, checkAuth } = useAuth();
  const [balance, setBalance] = useState(null);
  const [packages, setPackages] = useState([]);
  const [active, setActive] = useState(null);
  const [usage, setUsage] = useState(null);
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [bal, pkg, act, usg, txn] = await Promise.all([
        apiClient.get('/api/v1/balance'),
        apiClient.get('/api/v1/packages'),
        apiClient.get('/api/v1/subscriptions/active').catch(() => null),
        apiClient.get('/api/v1/subscriptions/usage').catch(() => null),
        apiClient.get('/api/v1/transactions').catch(() => ({ data: { transactions: [] } })),
      ]);
      setBalance(bal.data.balance);
      setPackages(pkg.data.packages || []);
      setActive(act?.data?.subscription || null);
      setUsage(usg?.data?.usage || null);
      setTransactions(txn.data.transactions || []);
    } catch (err) {
      setError('تعذّر تحميل البيانات من الخادم.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const subscribe = async (packageId) => {
    setBusy(true);
    setError('');
    try {
      const res = await apiClient.post('/api/v1/subscriptions', {
        package_id: packageId,
        auto_renew: true,
      });
      alert(res.data.message);
      await checkAuth();
      await loadData();
    } catch (err) {
      setError(err.response?.data?.message || err.response?.data?.error || 'فشل الاشتراك.');
    } finally {
      setBusy(false);
    }
  };

  const toggleAutoRenew = async (current) => {
    if (!active) return;
    setBusy(true);
    try {
      const path = current
        ? `/api/v1/subscriptions/${active.id}/cancel`
        : `/api/v1/subscriptions/${active.id}/enable-auto-renew`;
      await apiClient.put(path);
      await loadData();
    } catch (err) {
      setError('تعذّر تحديث التجديد التلقائي.');
    } finally {
      setBusy(false);
    }
  };

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  return (
    <div className="min-h-screen bg-slate-950 pb-16">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 bg-rose-500/10 rounded-xl border border-rose-500/20">
              <Building2 className="w-6 h-6 text-rose-500" />
            </div>
            <div>
              <h1 className="text-lg font-black text-white">بوابة الشركات</h1>
              <p className="text-xs text-slate-400">{tenant?.business_name || user?.email}</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 rounded-xl text-sm text-slate-300 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            خروج
          </button>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 mt-8 space-y-8">
        {error && (
          <div className="px-4 py-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-sm">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-24">
            <div className="animate-spin rounded-full h-12 w-12 border-4 border-rose-500 border-t-transparent" />
          </div>
        ) : (
          <>
            {/* Balance card */}
            <section className="grid md:grid-cols-2 gap-6">
              <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-3 bg-emerald-500/10 rounded-xl border border-emerald-500/20">
                    <Wallet className="w-6 h-6 text-emerald-400" />
                  </div>
                  <h2 className="text-lg font-bold text-white">الرصيد الحالي</h2>
                </div>
                <p className="text-4xl font-black text-emerald-400" dir="ltr">
                  ${balance}
                </p>
                <p className="text-sm text-slate-400 mt-2">متاح للاشتراك في الباقات ودفع رسوم الخدمات</p>
              </div>

              <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-3 bg-sky-500/10 rounded-xl border border-sky-500/20">
                    <MessageSquare className="w-6 h-6 text-sky-400" />
                  </div>
                  <h2 className="text-lg font-bold text-white">استهلاك الباقة</h2>
                </div>
                {usage ? (
                  <>
                    <p className="text-3xl font-black text-white" dir="ltr">
                      {usage.messages_used} / {usage.included_messages ?? '∞'}
                    </p>
                    <p className="text-sm text-slate-400 mt-2">
                      باقة {usage.package_name} • متبقي {usage.messages_remaining ?? '∞'} رسالة
                    </p>
                  </>
                ) : (
                  <p className="text-slate-400">لا توجد اشتراك نشط حالياً.</p>
                )}
              </div>
            </section>

            {/* Active subscription */}
            <section className="bg-slate-900 border border-slate-800 rounded-2xl p-6">
              <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-violet-500/10 rounded-xl border border-violet-500/20">
                    <RefreshCcw className="w-6 h-6 text-violet-400" />
                  </div>
                  <h2 className="text-lg font-bold text-white">الاشتراك الحالي</h2>
                </div>
                {active && (
                  <button
                    onClick={() => toggleAutoRenew(active.auto_renew)}
                    disabled={busy}
                    className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-bold transition-colors disabled:opacity-50 ${
                      active.auto_renew
                        ? 'bg-amber-500/10 text-amber-300 border border-amber-500/30 hover:bg-amber-500/20'
                        : 'bg-emerald-500/10 text-emerald-300 border border-emerald-500/30 hover:bg-emerald-500/20'
                    }`}
                  >
                    {active.auto_renew ? <X className="w-4 h-4" /> : <Check className="w-4 h-4" />}
                    {active.auto_renew ? 'إيقاف التجديد التلقائي' : 'تفعيل التجديد التلقائي'}
                  </button>
                )}
              </div>
              {active ? (
                <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div className="bg-slate-800/50 rounded-xl p-4">
                    <p className="text-xs text-slate-400 mb-1">حالة الاشتراك</p>
                    <p className="font-bold text-white capitalize">{active.status}</p>
                  </div>
                  <div className="bg-slate-800/50 rounded-xl p-4">
                    <p className="text-xs text-slate-400 mb-1">بداية الفترة</p>
                    <p className="font-bold text-white">{formatDate(active.start_date)}</p>
                  </div>
                  <div className="bg-slate-800/50 rounded-xl p-4">
                    <p className="text-xs text-slate-400 mb-1">نهاية الفترة</p>
                    <p className="font-bold text-white">{formatDate(active.end_date)}</p>
                  </div>
                  <div className="bg-slate-800/50 rounded-xl p-4">
                    <p className="text-xs text-slate-400 mb-1">التجديد التلقائي</p>
                    <p className="font-bold text-white">{active.auto_renew ? 'مفعّل' : 'متوقف'}</p>
                  </div>
                </div>
              ) : (
                <p className="text-slate-400">لا يوجد اشتراك نشط. اختر باقة من الأسفل للاشتراك.</p>
              )}
            </section>

            {/* Packages */}
            <section>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-rose-500/10 rounded-xl border border-rose-500/20">
                  <Package className="w-6 h-6 text-rose-400" />
                </div>
                <h2 className="text-lg font-bold text-white">باقات الاشتراك</h2>
              </div>
              <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
                {packages.map((pkg) => (
                  <div
                    key={pkg.id}
                    className="bg-slate-900 border border-slate-800 rounded-2xl p-6 flex flex-col gap-4"
                  >
                    <div className="flex items-start justify-between">
                      <div>
                        <h3 className="text-lg font-black text-white">{pkg.name}</h3>
                        <p className="text-xs text-slate-400">{pkg.code}</p>
                      </div>
                      {active?.package_id === pkg.id && (
                        <span className="px-2.5 py-1 bg-emerald-500/10 border border-emerald-500/30 rounded-lg text-xs font-bold text-emerald-300">
                          الاشتراك الحالي
                        </span>
                      )}
                    </div>
                    <p className="text-3xl font-black text-white" dir="ltr">
                      ${pkg.price_usd}
                      <span className="text-sm text-slate-400 font-normal"> / شهرياً</span>
                    </p>
                    <ul className="text-sm text-slate-300 space-y-2">
                      <li className="flex items-center gap-2">
                        <Check className="w-4 h-4 text-emerald-400" />
                        {pkg.is_unlimited ? 'رسائل غير محدودة' : `${pkg.included_messages.toLocaleString('en')} رسالة`}
                      </li>
                      <li className="flex items-center gap-2">
                        <Check className="w-4 h-4 text-emerald-400" />
                        رسوم إضافية ${pkg.overage_rate} لكل رسالة زائدة
                      </li>
                      {pkg.rollover_enabled && (
                        <li className="flex items-center gap-2">
                          <Check className="w-4 h-4 text-emerald-400" />
                          ترحيل الرسائل المتبقية
                        </li>
                      )}
                    </ul>
                    {active?.package_id !== pkg.id && (
                      <button
                        onClick={() => subscribe(pkg.id)}
                        disabled={busy}
                        className="mt-auto w-full px-4 py-3 bg-rose-600 hover:bg-rose-500 disabled:opacity-50 rounded-xl text-white font-bold transition-colors"
                      >
                        اشترك الآن
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </section>

            {/* Transactions */}
            <section>
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-cyan-500/10 rounded-xl border border-cyan-500/20">
                  <Receipt className="w-6 h-6 text-cyan-400" />
                </div>
                <h2 className="text-lg font-bold text-white">آخر العمليات</h2>
              </div>
              <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
                {transactions.length === 0 ? (
                  <p className="p-6 text-slate-400">لا توجد عمليات بعد.</p>
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-slate-800/50 text-slate-400">
                        <th className="text-right px-6 py-3 font-semibold">النوع</th>
                        <th className="text-right px-6 py-3 font-semibold">المبلغ</th>
                        <th className="text-right px-6 py-3 font-semibold">الحالة</th>
                        <th className="text-right px-6 py-3 font-semibold">التاريخ</th>
                      </tr>
                    </thead>
                    <tbody>
                      {transactions.map((t) => (
                        <tr key={t.id} className="border-t border-slate-800">
                          <td className="px-6 py-3 font-bold text-white">{t.type}</td>
                          <td className="px-6 py-3" dir="ltr">
                            <span className={t.type === 'SUBSCRIPTION_REFUND' ? 'text-emerald-400' : 'text-slate-200'}>
                              ${t.amount}
                            </span>
                          </td>
                          <td className="px-6 py-3">
                            <span className="px-2.5 py-1 bg-slate-800 rounded-lg text-xs text-slate-300 capitalize">
                              {t.status.toLowerCase()}
                            </span>
                          </td>
                          <td className="px-6 py-3 text-slate-400">{formatDate(t.created_at)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </section>
          </>
        )}
      </main>
    </div>
  );
}
