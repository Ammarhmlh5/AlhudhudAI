import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Building2, ArrowLeft, BarChart3, Activity, 
  History, Settings, ShieldCheck, Globe, 
  Zap, TrendingUp, CreditCard, ChevronRight, Trash2,
  Lock, Wallet, Layers, Cpu, Plus
} from 'lucide-react';
import apiClient from '../../api/axios';
import RoutingConfig from '../../components/Companies/RoutingConfig';
import { usePackages } from '../../hooks/usePackages';

export default function CompanyDetails() {
  const { t } = useTranslation();
  const { tenantId } = useParams();
  const navigate = useNavigate();
  const [company, setCompany] = useState(null);
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('overview');
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [paymentAmount, setPaymentAmount] = useState('');
  const [processingPayment, setProcessingPayment] = useState(false);

  useEffect(() => {
    fetchCompanyData();
  }, [tenantId]);

  const fetchCompanyData = async () => {
    try {
      setLoading(true);
      const ratingResponse = await apiClient.get(`/api/v1/admin/companies/${tenantId}/rating`);
      setCompany(ratingResponse.data);

      const metricsResponse = await apiClient.get(`/api/v1/admin/metrics/${tenantId}`);
      setMetrics(metricsResponse.data);

      try {
        const historyResponse = await apiClient.get(`/api/v1/admin/companies/${tenantId}/rating/history`);
        setHistory(historyResponse.data.history || []);
      } catch (e) {
        setHistory([]);
      }
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || t('companyDetails.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteCompany = async () => {
    if (!window.confirm(t('companyDetails.delete_confirm'))) return;
    
    try {
        await apiClient.post('/api/v1/admin/tenants/bulk-delete', { ids: [tenantId] });
        navigate('/companies');
    } catch (err) {
        alert(err.response?.data?.error || t('companyDetails.delete_failed'));
    }
  };

  const handleImpersonate = async (tenantId) => {
    try {
        const res = await apiClient.get(`/api/v1/admin/tenants/${tenantId}/impersonate`);
        const clientUrl = import.meta.env.VITE_CLIENT_WEB_URL || 'http://localhost:5173';
        window.open(`${clientUrl}/login?token=${res.data.token}`, '_blank');
    } catch (err) {
        alert(t('companyDetails.login_failed'));
    }
  };

  const handleToggleStatus = async () => {
    const newStatus = company?.status === 'SUSPENDED' ? true : false;
    try {
        await apiClient.post('/api/v1/admin/tenants/bulk-update', { ids: [tenantId], is_active: newStatus });
        fetchCompanyData();
    } catch (err) {
        alert(t('companyDetails.update_status_failed'));
    }
  };

  const handleAddBalance = async (e) => {
    e.preventDefault();
    if (!paymentAmount) return;
    try {
        setProcessingPayment(true);
        // Assuming there's an endpoint for manual balance update or use a generic one
        await apiClient.post(`/api/v1/admin/tenants/${tenantId}/balance`, { amount: parseFloat(paymentAmount) });
        setShowPaymentModal(false);
        setPaymentAmount('');
        fetchCompanyData();
        alert(t('companyDetails.balance_added'));
    } catch (err) {
        alert(t('companyDetails.balance_add_failed'));
    } finally {
        setProcessingPayment(false);
    }
  };

  const getRatingStyles = (category) => {
    switch (category) {
      case 'excellent': return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20';
      case 'good':      return 'text-blue-400 bg-blue-500/10 border-blue-500/20';
      case 'average':   return 'text-amber-400 bg-amber-500/10 border-amber-500/20';
      case 'poor':      return 'text-rose-400 bg-rose-500/10 border-rose-500/20';
      default:          return 'text-slate-400 bg-slate-500/10 border-slate-500/20';
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] space-y-4">
        <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
        <p className="text-slate-500 font-bold animate-pulse">{t('companyDetails.syncing_identity')}</p>
      </div>
    );
  }

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div className="flex items-center gap-6">
          <button
            onClick={() => navigate('/companies')}
            className="p-3 bg-slate-900 text-slate-500 hover:text-white border border-slate-800 rounded-2xl transition-all group"
          >
            <ArrowLeft className="w-6 h-6 group-hover:-translate-x-1 transition-transform" />
          </button>
          <div>
            <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-3">
              {company?.tenant_name}
              <ShieldCheck className="w-6 h-6 text-emerald-500" />
            </h1>
            <p className="mt-1 text-slate-500 font-mono text-xs">{t('companyDetails.tenant_uid')} {tenantId}</p>
          </div>
        </div>
        
        <div className="flex flex-wrap items-center gap-3">
            <button
                onClick={() => setShowPaymentModal(true)}
                className="px-6 py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-emerald-900/20"
            >
                <Wallet className="w-4 h-4" />
                {t('companyDetails.manual_top_up')}
            </button>
            <button
                onClick={() => handleImpersonate(tenantId)}
                className="px-6 py-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-indigo-900/20"
            >
                <Lock className="w-4 h-4" />
                {t('companyDetails.login')}
            </button>
            <button
                onClick={() => handleToggleStatus()}
                className={`px-6 py-3 border rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 ${
                    company?.status === 'SUSPENDED' 
                    ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-500 hover:bg-emerald-500 hover:text-white' 
                    : 'bg-amber-500/10 border-amber-500/20 text-amber-500 hover:bg-amber-500 hover:text-white'
                }`}
            >
                {company?.status === 'SUSPENDED' ? <ShieldCheck className="w-4 h-4" /> : <Lock className="w-4 h-4" />}
                {company?.status === 'SUSPENDED' ? t('companyDetails.activate_account') : t('companyDetails.suspend_account')}
            </button>
            <div className="w-px h-8 bg-slate-800 mx-2" />
            <button
                onClick={() => navigate(`/signalpush/${tenantId}`)}
                className="px-6 py-3 bg-slate-900 border border-slate-800 text-slate-300 hover:text-white rounded-2xl font-bold uppercase tracking-widest text-xs transition-all flex items-center gap-2 group"
            >
                <Zap className="w-4 h-4 text-amber-500 group-hover:animate-pulse" />
                {t('companyDetails.sending_mode')}
            </button>
            <button
                onClick={handleDeleteCompany}
                className="px-4 py-3 bg-slate-950 border border-rose-900/30 text-rose-500 hover:bg-rose-600 hover:text-white rounded-2xl font-bold uppercase tracking-widest text-[10px] transition-all flex items-center gap-2"
            >
                <Trash2 className="w-3 h-3" />
                {t('companyDetails.delete')}
            </button>
        </div>
      </div>

      {/* Hero Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <HeroMetricCard
          label={t('companyDetails.reputation')}
          value={company?.overall_rating?.toFixed(1) || '—'}
          sub={t(`companies.${company?.rating_category || 'unknown'}`)}
          styles={getRatingStyles(company?.rating_category)}
          icon={ShieldCheck}
        />
        <HeroMetricCard
          label={t('companyDetails.success_rate')}
          value={`${company?.success_rate?.toFixed(1) || 0}%`}
          sub={t('companyDetails.platform_avg')}
          styles="text-white bg-slate-900 border-slate-800"
          icon={Activity}
        />
        <HeroMetricCard
          label={t('companyDetails.total_volume')}
          value={company?.sending_volume?.toLocaleString() || 0}
          sub={t('companyDetails.messages_sent')}
          styles="text-white bg-slate-900 border-slate-800"
          icon={TrendingUp}
        />
        <HeroMetricCard
          label={t('companyDetails.compliance')}
          value={`${company?.payment_compliance?.toFixed(1) || 0}%`}
          sub={t('companyDetails.billed_on_time')}
          styles="text-white bg-slate-900 border-slate-800"
          icon={CreditCard}
        />
      </div>

      {/* Tabs Interface */}
      <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] overflow-hidden shadow-2xl">
        <div className="flex flex-wrap border-b border-slate-800 p-2 bg-slate-950/50 gap-1">
            <TabButton active={activeTab === 'overview'} onClick={() => setActiveTab('overview')} icon={Building2} label={t('companyDetails.tabs.profile')} />
            <TabButton active={activeTab === 'subscription'} onClick={() => setActiveTab('subscription')} icon={Layers} label={t('companyDetails.tabs.subscription')} />
            <TabButton active={activeTab === 'finance'} onClick={() => setActiveTab('finance')} icon={Wallet} label={t('companyDetails.tabs.finance')} />
            <TabButton active={activeTab === 'metrics'} onClick={() => setActiveTab('metrics')} icon={Activity} label={t('companyDetails.tabs.analytics')} />
            <TabButton active={activeTab === 'history'} onClick={() => setActiveTab('history')} icon={History} label={t('companyDetails.tabs.audit_log')} />
            <TabButton active={activeTab === 'routing'} onClick={() => setActiveTab('routing')} icon={Settings} label={t('companyDetails.tabs.routing')} />
            <TabButton active={activeTab === 'gateways'} onClick={() => setActiveTab('gateways')} icon={Cpu} label={t('companyDetails.tabs.private_gateways')} />
        </div>

        <div className="p-10">
          {activeTab === 'overview' && <OverviewTab company={company} />}
          {activeTab === 'subscription' && <SubscriptionTab tenantId={tenantId} />}
          {activeTab === 'finance' && <FinanceTab tenantId={tenantId} />}
          {activeTab === 'metrics' && <MetricsTab metrics={metrics} />}
          {activeTab === 'history' && <HistoryTab history={history} />}
          {activeTab === 'routing' && <RoutingConfig tenantId={tenantId} />}
          {activeTab === 'gateways' && <PrivateGatewaysTab tenantId={tenantId} />}
        </div>
      </div>

      {/* Manual Payment Modal */}
      {showPaymentModal && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-in fade-in duration-300">
            <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 max-w-md w-full shadow-2xl relative overflow-hidden">
                <div className="absolute top-0 left-0 w-full h-1 bg-emerald-500"></div>
                <div className="flex justify-between items-center mb-8">
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-emerald-500/10 rounded-2xl text-emerald-500">
                            <Wallet className="w-6 h-6" />
                        </div>
                        <div>
                            <h2 className="text-2xl font-black text-white tracking-tight">{t('companyDetails.manual_top_up_title')}</h2>
                            <p className="text-slate-500 text-[10px] font-black uppercase tracking-widest mt-1">{t('companyDetails.manual_top_up_desc')}</p>
                        </div>
                    </div>
                    <button onClick={() => setShowPaymentModal(false)} className="p-3 bg-slate-950 border border-slate-800 rounded-2xl text-slate-500 hover:text-white transition-all">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <form onSubmit={handleAddBalance} className="space-y-6">
                    <div className="space-y-2">
                        <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest ml-1">{t('companyDetails.amount_sar')}</label>
                        <div className="relative">
                            <div className="absolute left-5 top-1/2 -translate-y-1/2 text-slate-500 font-bold">$</div>
                            <input
                                type="number"
                                required
                                value={paymentAmount}
                                onChange={e => setPaymentAmount(e.target.value)}
                                className="w-full pl-10 pr-5 py-5 bg-slate-950 border border-slate-800 rounded-2xl text-white focus:outline-none focus:border-emerald-500 transition-all font-mono text-2xl font-black"
                                placeholder="0.00"
                            />
                        </div>
                    </div>

                    <div className="p-4 bg-slate-950/50 border border-slate-800/50 rounded-2xl">
                        <p className="text-[10px] text-slate-500 font-bold uppercase mb-2">{t('companyDetails.target_account')}</p>
                        <p className="text-white font-bold">{company?.tenant_name}</p>
                        <p className="text-xs text-slate-500 font-mono mt-1">{tenantId}</p>
                    </div>

                    <button
                        type="submit"
                        disabled={processingPayment}
                        className="w-full bg-emerald-600 hover:bg-emerald-500 disabled:bg-emerald-900 disabled:opacity-50 text-white py-5 rounded-2xl font-black uppercase tracking-[0.2em] transition-all shadow-xl shadow-emerald-900/20"
                    >
                        {processingPayment ? t('companyDetails.processing') : t('companyDetails.confirm_transaction')}
                    </button>
                </form>
            </div>
        </div>
      )}
    </div>
  );
}

function HeroMetricCard({ label, value, sub, styles, icon: Icon }) {
    return (
        <div className={`p-8 rounded-3xl border transition-all hover:scale-[1.02] ${styles}`}>
            <div className="flex justify-between items-start mb-4">
                <p className="text-[10px] font-black uppercase tracking-[0.2em] opacity-60">{label}</p>
                <Icon className="w-5 h-5 opacity-40" />
            </div>
            <div className="text-4xl font-black tracking-tighter mb-1">{value}</div>
            <p className="text-xs font-bold opacity-50 uppercase tracking-wider">{sub}</p>
        </div>
    );
}

function TabButton({ active, onClick, icon: Icon, label }) {
    return (
        <button
            onClick={onClick}
            className={`flex items-center gap-3 px-8 py-4 rounded-2xl text-xs font-black uppercase tracking-widest transition-all ${
                active ? 'bg-rose-600 text-white shadow-lg shadow-rose-900/20' : 'text-slate-500 hover:text-slate-300'
            }`}
        >
            <Icon className="w-4 h-4" />
            {label}
        </button>
    );
}

function OverviewTab({ company }) {
    const { t } = useTranslation();
    return (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
            <div className="space-y-8">
                <h3 className="text-xl font-bold text-white border-l-4 border-rose-600 pl-4">{t('companyDetails.identification')}</h3>
                <div className="space-y-4">
                    <DetailRow label={t('companyDetails.legal_name')} value={company?.tenant_name} />
                    <DetailRow label={t('companyDetails.system_id')} value={company?.tenant_id} mono />
                    <DetailRow label={t('companyDetails.created_date')} value={new Date(company?.created_at).toLocaleDateString()} />
                    <DetailRow label={t('companyDetails.last_activity')} value={new Date(company?.updated_at).toLocaleDateString()} />
                </div>
            </div>
            <div className="space-y-8">
                <h3 className="text-xl font-bold text-white border-l-4 border-indigo-600 pl-4">{t('companyDetails.platform_status')}</h3>
                <div className="space-y-4">
                    <DetailRow label={t('companyDetails.current_status')} value={company?.status || t('tenantManager.active')} />
                    <DetailRow label={t('companyDetails.gateway_access')} value={t('companyDetails.enterprise_dedicated')} />
                    <DetailRow label={t('companyDetails.api_access')} value={t('companyDetails.enabled')} />
                    <DetailRow label={t('companyDetails.data_residency')} value={t('companyDetails.global_default')} />
                </div>
            </div>
        </div>
    );
}

function DetailRow({ label, value, mono }) {
    return (
        <div className="flex justify-between items-center py-4 border-b border-slate-800/50">
            <span className="text-xs font-black text-slate-500 uppercase tracking-wider">{label}</span>
            <span className={`text-sm font-bold text-slate-300 ${mono ? 'font-mono' : ''}`}>{value}</span>
        </div>
    );
}

function MetricsTab({ metrics }) {
    const { t } = useTranslation();
    if (!metrics) return <div className="text-slate-500 italic">{t('companyDetails.no_analytics')}</div>;

    return (
        <div className="space-y-12">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                <MetricRing label={t('companyDetails.delivered')} value={metrics.successful_messages} total={metrics.total_messages} color="text-emerald-500" />
                <MetricRing label={t('companyDetails.failed')} value={metrics.failed_messages} total={metrics.total_messages} color="text-rose-500" />
                <MetricRing label={t('companyDetails.pending')} value={metrics.total_messages - (metrics.successful_messages + metrics.failed_messages)} total={metrics.total_messages} color="text-amber-500" />
            </div>
        </div>
    );
}

function MetricRing({ label, value, total, color }) {
    const percentage = total > 0 ? (value / total) * 100 : 0;
    return (
        <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-3xl text-center">
            <div className="relative w-24 h-24 mx-auto mb-6 flex items-center justify-center">
                <svg className="w-full h-full transform -rotate-90">
                    <circle cx="48" cy="48" r="40" stroke="currentColor" strokeWidth="8" fill="transparent" className="text-slate-900" />
                    <circle cx="48" cy="48" r="40" stroke="currentColor" strokeWidth="8" fill="transparent" className={color} strokeDasharray={251.2} strokeDashoffset={251.2 - (251.2 * percentage) / 100} />
                </svg>
                <span className="absolute text-xs font-black text-white">{percentage.toFixed(0)}%</span>
            </div>
            <div className="text-2xl font-black text-white mb-1">{value?.toLocaleString()}</div>
            <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{label}</div>
        </div>
    );
}

function HistoryTab({ history }) {
    const { t } = useTranslation();
    if (!history || history.length === 0) return <div className="text-slate-500 italic">{t('companyDetails.no_history')}</div>;

    return (
        <div className="space-y-4">
            {history.map((record, index) => (
                <div key={index} className="flex items-center justify-between p-6 bg-slate-950 border border-slate-800 rounded-2xl hover:border-slate-700 transition-colors group">
                    <div className="flex items-center gap-6">
                        <div className="text-center">
                            <div className="text-[10px] font-black text-slate-600 uppercase mb-1">{t('companyDetails.score')}</div>
                            <div className="text-lg font-black text-white">{record.overall_rating?.toFixed(2)}</div>
                        </div>
                        <div className="w-px h-8 bg-slate-800" />
                        <div>
                            <div className="text-sm font-bold text-slate-300 mb-1">{t('companyDetails.rating_adjusted')}</div>
                            <div className="text-[10px] font-mono text-slate-600 uppercase">{new Date(record.created_at).toLocaleString()}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-4">
                        <span className="px-3 py-1.5 text-[10px] font-black uppercase bg-slate-900 border border-slate-800 text-slate-500 rounded-xl">
                            {record.rating_category}
                        </span>
                        <ChevronRight className="w-5 h-5 text-slate-800 group-hover:text-rose-500 transition-colors" />
                    </div>
                </div>
            ))}
        </div>
    );
}

function SubscriptionTab({ tenantId }) {
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [actionError, setActionError] = useState(null);
    const [busy, setBusy] = useState(false);
    const [selectedPackageId, setSelectedPackageId] = useState('');
    const { packages, loading: packagesLoading } = usePackages({ is_active: true });
    const { t } = useTranslation();

    const fetchSubscription = useCallback(async () => {
        try {
            setLoading(true);
            const res = await apiClient.get(`/api/v1/admin/tenants/${tenantId}/subscription`);
            setData(res.data);
            setError(null);
        } catch (err) {
            setData(null);
            setError(err.response?.data?.error || t('companyDetails.load_failed'));
        } finally {
            setLoading(false);
        }
    }, [tenantId]);

    useEffect(() => {
        fetchSubscription();
    }, [fetchSubscription]);

    const handleChangePackage = async (e) => {
        e.preventDefault();
        if (!selectedPackageId) return;
        setBusy(true);
        setActionError(null);
        try {
            await apiClient.post(`/api/v1/admin/tenants/${tenantId}/subscribe`, {
                package_id: Number(selectedPackageId),
                auto_renew: data?.subscription?.auto_renew ?? true,
            });
            setSelectedPackageId('');
            await fetchSubscription();
            alert(t('companyDetails.package_updated'));
        } catch (err) {
            setActionError(err.response?.data?.message || err.response?.data?.error || t('companyDetails.update_package_failed'));
        } finally {
            setBusy(false);
        }
    };

    const handleToggleAutoRenew = async () => {
        if (!data?.subscription) return;
        setBusy(true);
        setActionError(null);
        try {
            await apiClient.put(`/api/v1/admin/tenants/${tenantId}/subscription/auto-renew`, {
                enabled: !data.subscription.auto_renew,
            });
            await fetchSubscription();
        } catch (err) {
            setActionError(err.response?.data?.error || t('companyDetails.auto_renew_failed'));
        } finally {
            setBusy(false);
        }
    };

    const sub = data?.subscription;
    const pkg = data?.package;
    const statusLabel = ({
        active: t('companyDetails.status_active'),
        grace_period: t('companyDetails.status_grace'),
        suspended: t('companyDetails.status_suspended'),
        expired: t('companyDetails.status_expired'),
        cancelled: t('companyDetails.status_cancelled')
    })[sub?.status] || sub?.status || t('companyDetails.no_subscription');

    const statusStyle = sub?.status === 'active'
        ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
        : sub?.status === 'grace_period'
            ? 'text-amber-400 bg-amber-500/10 border-amber-500/20'
            : 'text-rose-400 bg-rose-500/10 border-rose-500/20';

    const formatNumber = (value) => {
        const num = Number(value);
        return Number.isFinite(num) ? num.toLocaleString('en-US') : '—';
    };

    return (
        <div className="space-y-8 animate-in slide-in-from-right duration-500">
            <div className="flex justify-between items-center">
                <h3 className="text-xl font-bold text-white border-l-4 border-indigo-600 pl-4">{t('companyDetails.subscription_plan')}</h3>
                {sub && (
                    <button
                        onClick={handleToggleAutoRenew}
                        disabled={busy}
                        className={`px-4 py-2 rounded-xl text-xs font-black uppercase tracking-widest transition-all ${
                            sub.auto_renew
                                ? 'bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 hover:bg-emerald-500 hover:text-white'
                                : 'bg-slate-900 border border-slate-800 text-slate-400 hover:bg-slate-800 hover:text-white'
                        }`}
                    >
                        {t('companyDetails.auto_renew')} {sub.auto_renew ? t('companyDetails.on') : t('companyDetails.off')}
                    </button>
                )}
            </div>

            {actionError && (
                <div className="bg-rose-500/10 border border-rose-500/20 text-rose-400 rounded-2xl px-5 py-3 text-sm font-bold">
                    {actionError}
                </div>
            )}

            {loading && (
                <div className="text-slate-500 font-bold py-12 text-center">{t('companyDetails.loading_subscription')}</div>
            )}

            {!loading && error && (
                <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem]">
                    <div className="text-sm text-slate-400 mb-4">{error}</div>
                    <div className="text-xs font-black text-slate-500 uppercase tracking-widest mb-3">{t('companyDetails.assign_package')}</div>
                    <form onSubmit={handleChangePackage} className="flex flex-col sm:flex-row gap-3">
                        <select
                            value={selectedPackageId}
                            onChange={(e) => setSelectedPackageId(e.target.value)}
                            disabled={packagesLoading || busy}
                            className="flex-1 bg-slate-900 border border-slate-800 text-white rounded-xl px-4 py-3 text-sm disabled:opacity-50"
                        >
                            <option value="">{packagesLoading ? t('companyDetails.loading_packages') : t('companyDetails.select_package')}</option>
                            {packages.map((p) => (
                                <option key={p.id} value={p.id}>
                                    {p.name} — {formatNumber(p.price_usd)} TND / {p.billing_cycle}
                                </option>
                            ))}
                        </select>
                        <button
                            type="submit"
                            disabled={!selectedPackageId || busy}
                            className="px-6 py-3 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl text-xs font-black uppercase tracking-widest transition-all"
                        >
                            {busy ? t('companyDetails.processing') : t('companyDetails.subscribe')}
                        </button>
                    </form>
                </div>
            )}

            {!loading && data && (
                <>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                        <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem]">
                            <div className="flex items-center gap-4 mb-6">
                                <div className="p-3 bg-indigo-500/10 rounded-2xl text-indigo-500"><Layers className="w-6 h-6" /></div>
                                <div>
                                    <div className="text-xs font-black text-slate-500 uppercase">{t('companyDetails.current_tier')}</div>
                                    <div className="text-xl font-black text-white">{pkg?.name || '—'}</div>
                                </div>
                            </div>
                            <div className="space-y-4">
                                <DetailRow label={t('companyDetails.price')} value={`${formatNumber(pkg?.price_usd)} TND / ${pkg?.billing_cycle || '—'}`} />
                                <DetailRow
                                    label={t('companyDetails.included_messages')}
                                    value={pkg?.is_unlimited ? t('companyDetails.unlimited') : `${formatNumber(pkg?.included_messages)} Msg`}
                                />
                                <DetailRow label={t('companyDetails.renewal_date')} value={sub?.end_date ? new Date(sub.end_date).toLocaleDateString() : '—'} />
                                <DetailRow label={t('companyDetails.started')} value={sub?.start_date ? new Date(sub.start_date).toLocaleDateString() : '—'} />
                                <div className="flex items-center justify-between">
                                    <span className="text-xs font-black text-slate-500 uppercase">{t('companyDetails.status')}</span>
                                    <span className={`px-3 py-1 text-xs font-black uppercase rounded-xl border ${statusStyle}`}>
                                        {statusLabel}
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem]">
                            <div className="text-xs font-black text-slate-500 uppercase mb-6">{t('companyDetails.usage')}</div>
                            <div className="space-y-4">
                                <DetailRow label={t('companyDetails.messages_used')} value={`${formatNumber(sub?.messages_used)} Msg`} />
                                <DetailRow label={t('companyDetails.messages_remaining')} value={sub?.messages_remaining == null ? t('companyDetails.unlimited') : `${formatNumber(sub?.messages_remaining)} Msg`} />
                                <DetailRow label={t('companyDetails.overage_messages')} value={`${formatNumber(sub?.overage_messages)} Msg`} />
                                <DetailRow label={t('companyDetails.overage_cost')} value={`${formatNumber(sub?.overage_cost)} TND`} />
                                <DetailRow label={t('companyDetails.rollover')} value={pkg?.rollover_enabled ? t('companyDetails.enabled') : t('tenantManager.disabled')} />
                            </div>
                        </div>
                    </div>

                    <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem]">
                        <div className="text-xs font-black text-slate-500 uppercase tracking-widest mb-4">{t('companyDetails.change_package')}</div>
                        <form onSubmit={handleChangePackage} className="flex flex-col sm:flex-row gap-3">
                            <select
                                value={selectedPackageId}
                                onChange={(e) => setSelectedPackageId(e.target.value)}
                                disabled={packagesLoading || busy}
                                className="flex-1 bg-slate-900 border border-slate-800 text-white rounded-xl px-4 py-3 text-sm disabled:opacity-50"
                            >
                                <option value="">{packagesLoading ? t('companyDetails.loading_packages') : t('companyDetails.select_package')}</option>
                                {packages.map((p) => (
                                    <option key={p.id} value={p.id}>
                                        {p.name} — {formatNumber(p.price_usd)} TND / {p.billing_cycle}
                                    </option>
                                ))}
                            </select>
                            <button
                                type="submit"
                                disabled={!selectedPackageId || busy}
                                className="px-6 py-3 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl text-xs font-black uppercase tracking-widest transition-all"
                            >
                                {busy ? t('companyDetails.processing') : t('companyDetails.change_package')}
                            </button>
                        </form>
                        <p className="text-xs text-slate-600 mt-3">
                            {t('companyDetails.change_package_note')}
                        </p>
                    </div>
                </>
            )}
        </div>
    );
}

function FinanceTab({ tenantId }) {
    const { t } = useTranslation();
    return (
        <div className="space-y-8 animate-in slide-in-from-right duration-500">
            <div className="flex justify-between items-center">
                <h3 className="text-xl font-bold text-white border-l-4 border-emerald-600 pl-4">{t('companyDetails.financial_status')}</h3>
                <button className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-black uppercase tracking-widest transition-all">
                    {t('companyDetails.add_funds')}
                </button>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem] text-center">
                    <div className="text-xs font-black text-slate-500 uppercase mb-2">{t('companyDetails.available_balance')}</div>
                    <div className="text-3xl font-black text-white">$12,450.00</div>
                </div>
                <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem] text-center">
                    <div className="text-xs font-black text-slate-500 uppercase mb-2">{t('companyDetails.pending_invoices')}</div>
                    <div className="text-3xl font-black text-rose-500">$0.00</div>
                </div>
                <div className="bg-slate-950/50 border border-slate-800 p-8 rounded-[2rem] text-center text-emerald-500">
                    <div className="text-xs font-black text-slate-500 uppercase mb-2">{t('companyDetails.total_credit')}</div>
                    <div className="text-3xl font-black">$50,000.00</div>
                </div>
            </div>
        </div>
    );
}

function PrivateGatewaysTab({ tenantId }) {
    const { t } = useTranslation();
    return (
        <div className="space-y-8 animate-in slide-in-from-right duration-500">
            <div className="flex justify-between items-center">
                <h3 className="text-xl font-bold text-white border-l-4 border-rose-600 pl-4">{t('companyDetails.private_gateways')}</h3>
                <button className="px-6 py-3 bg-rose-600 hover:bg-rose-500 text-white rounded-2xl font-black uppercase tracking-widest text-xs transition-all flex items-center gap-2 shadow-lg shadow-rose-900/20">
                    <Plus className="w-4 h-4" />
                    {t('companyDetails.create_private_gateway')}
                </button>
            </div>
            <div className="flex flex-col items-center justify-center py-20 bg-slate-950/30 border-2 border-dashed border-slate-800 rounded-[2.5rem]">
                <Cpu className="w-16 h-16 text-slate-800 mb-4" />
                <p className="text-slate-500 font-bold">{t('companyDetails.no_gateways')}</p>
                <p className="text-xs text-slate-600 uppercase tracking-widest mt-1">{t('companyDetails.create_bridge')}</p>
            </div>
        </div>
    );
}

