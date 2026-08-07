import React, { useState, useEffect, useCallback } from 'react';
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Users, Building2, LayoutDashboard,
  LogOut, Menu, X, Plus,
  Search, ShieldCheck, Globe, Package, Wallet,
  Copy, RefreshCw, Send, CheckCircle2, AlertTriangle, Loader2, XCircle
} from 'lucide-react';
import apiClient from './api/axios';
import Login from './components/Login';
import Packages from './components/Packages';

const ProtectedRoute = ({ children }) => {
    const token = localStorage.getItem('reseller_token');
    if (!token) return <Navigate to="/login" replace />;
    return children;
};

// ── Language Switcher ────────────────────────────────────────────────────────
const LanguageSwitcher = () => {
  const { i18n } = useTranslation();
  return (
    <button
      onClick={() => i18n.changeLanguage(i18n.language === 'ar' ? 'en' : 'ar')}
      className="flex items-center gap-2 px-3 py-2 bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 rounded-xl transition-colors text-xs font-bold"
    >
      <Globe className="w-4 h-4" />
      {i18n.language === 'ar' ? 'English' : 'عربي'}
    </button>
  );
};

const SidebarItem = ({ icon: Icon, label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center gap-4 px-6 py-4 transition-all duration-300 ${
      active
        ? 'text-rose-500 bg-rose-500/5 border-r-4 border-rose-500'
        : 'text-slate-400 hover:text-slate-100 hover:bg-slate-900/50'
    }`}
  >
    <Icon className={`w-5 h-5 ${active ? 'text-rose-500' : 'text-slate-500'}`} />
    <span className="font-bold text-sm tracking-wide">{label}</span>
  </button>
);

// ── Modal ────────────────────────────────────────────────────────────────────
const Modal = ({ title, subtitle, onClose, children }) => (
  <div className="fixed inset-0 z-[100] flex items-center justify-center p-6">
    <div className="absolute inset-0 bg-slate-950/80 backdrop-blur-sm" onClick={onClose} />
    <div className="relative w-full max-w-md glass-card p-8 border border-slate-700/50 max-h-[90vh] overflow-y-auto">
      <div className="flex items-start justify-between mb-6">
        <div>
          <h3 className="text-xl font-black text-white tracking-tight">{title}</h3>
          {subtitle && <p className="text-xs text-slate-500 font-bold mt-1">{subtitle}</p>}
        </div>
        <button onClick={onClose} className="text-slate-500 hover:text-white transition-colors">
          <X className="w-5 h-5" />
        </button>
      </div>
      {children}
    </div>
  </div>
);

const FormField = ({ label, children }) => (
  <div className="space-y-2">
    <label className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{label}</label>
    {children}
  </div>
);

const inputClass = "w-full bg-slate-950 border border-slate-800 rounded-xl py-3 px-4 text-white focus:outline-none focus:border-rose-500 transition-all text-sm";

// ── Dashboard ────────────────────────────────────────────────────────────────
const Dashboard = () => {
  const { t } = useTranslation();
  const [stats, setStats] = useState({ balance: '0', tenants: 0, staff: 0, messagesUsed: null });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    const fetchStats = async () => {
      try {
        const [balanceRes, tenantsRes, staffRes, usageRes] = await Promise.allSettled([
          apiClient.get('/api/v1/reseller/balance'),
          apiClient.get('/api/v1/reseller/tenants'),
          apiClient.get('/api/v1/reseller/staff'),
          apiClient.get('/api/v1/reseller/subscriptions/usage'),
        ]);
        if (!mounted) return;
        setStats({
          balance: balanceRes.status === 'fulfilled' ? balanceRes.value.data?.balance ?? '0' : '0',
          tenants: tenantsRes.status === 'fulfilled' ? (tenantsRes.value.data?.length ?? 0) : 0,
          staff: staffRes.status === 'fulfilled' ? (staffRes.value.data?.length ?? 0) : 0,
          messagesUsed: usageRes.status === 'fulfilled' ? usageRes.value.data?.usage?.messages_used ?? null : null,
        });
      } finally {
        if (mounted) setLoading(false);
      }
    };
    fetchStats();
    return () => { mounted = false; };
  }, []);

  const statCards = [
    { label: t('dash.wallet_balance'), value: Number(stats.balance).toLocaleString('en-US'), icon: Wallet, color: 'text-emerald-500', suffix: true },
    { label: t('dash.total_tenants'), value: stats.tenants.toLocaleString('en-US'), icon: Building2, color: 'text-blue-500', suffix: false },
    { label: t('dash.team_members'), value: stats.staff.toLocaleString('en-US'), icon: Users, color: 'text-rose-500', suffix: false },
    { label: t('dash.messages_used'), value: stats.messagesUsed != null ? stats.messagesUsed.toLocaleString('en-US') : '—', icon: Globe, color: 'text-amber-500', suffix: false },
  ];

  return (
    <div className="space-y-8 animate-fade-in">
      <div>
        <h1 className="text-4xl font-black text-white tracking-tighter">{t('dash.title')}</h1>
        <p className="text-slate-500 font-bold uppercase tracking-widest text-xs mt-2">{t('dash.subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {loading ? (
          <div className="col-span-full glass-card py-20 text-center text-slate-500 font-bold italic">{t('common.loading')}</div>
        ) : statCards.map((stat, i) => (
          <div key={i} className="glass-card p-8 group hover:border-slate-700 transition-all">
            <div className="flex justify-between items-start mb-4">
              <div className={`p-4 rounded-2xl bg-slate-950 border border-slate-800 ${stat.color}`}>
                <stat.icon className="w-6 h-6" />
              </div>
              {stat.suffix && <span className="text-[10px] font-black text-slate-600 uppercase tracking-widest">{t('common.currency')}</span>}
            </div>
            <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mb-1">{stat.label}</p>
            <h3 className="text-3xl font-black text-white">{stat.value}</h3>
          </div>
        ))}
      </div>
    </div>
  );
};

// ── Tenants (Customers) ──────────────────────────────────────────────────────
const TenantsList = () => {
  const { t } = useTranslation();
  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [transferTarget, setTransferTarget] = useState(null);
  const [subscribeTarget, setSubscribeTarget] = useState(null);
  const [packages, setPackages] = useState([]);
  const [form, setForm] = useState({ business_name: '', contact_name: '', email: '', phone: '', password: '' });
  const [formError, setFormError] = useState('');
  const [formLoading, setFormLoading] = useState(false);
  const [toast, setToast] = useState(null);

  const showToast = (msg, ok = true) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 3500);
  };

  const fetchTenants = useCallback(() => {
    apiClient.get('/api/v1/reseller/tenants')
      .then(res => setTenants(res.data))
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchTenants(); }, [fetchTenants]);

  const handleCreate = async (e) => {
    e.preventDefault();
    setFormError('');
    if (form.password.length < 6) {
      setFormError(t('customers.invalid_password'));
      return;
    }
    setFormLoading(true);
    try {
      await apiClient.post('/api/v1/reseller/tenants', form);
      setShowCreate(false);
      setForm({ business_name: '', contact_name: '', email: '', phone: '', password: '' });
      showToast(t('customers.form_success'));
      fetchTenants();
    } catch (err) {
      setFormError(err.response?.data?.error || t('customers.form_error'));
    } finally {
      setFormLoading(false);
    }
  };

  const handleTransfer = async (amount, note) => {
    try {
      await apiClient.post(`/api/v1/reseller/tenants/${transferTarget.id}/transfer`, { amount, reason: note });
      setTransferTarget(null);
      showToast(t('customers.transfer_success'));
      fetchTenants();
    } catch (err) {
      showToast(err.response?.data?.error || t('customers.transfer_error'), false);
    }
  };

  const loadPackages = () => {
    apiClient.get('/api/v1/reseller/packages')
      .then(res => setPackages(res.data.packages || []))
      .catch(() => setPackages([]));
  };

  const openSubscribe = (tenant) => {
    loadPackages();
    setSubscribeTarget(tenant);
  };

  const handleSubscribe = async (packageId, autoRenew) => {
    try {
      await apiClient.post(`/api/v1/reseller/tenants/${subscribeTarget.id}/subscribe`, { package_id: packageId, auto_renew: autoRenew });
      setSubscribeTarget(null);
      showToast(t('customers.subscribe_success'));
      fetchTenants();
    } catch (err) {
      showToast(err.response?.data?.error || t('customers.subscribe_error'), false);
    }
  };

  const handleRegenerateKey = async (tenant) => {
    if (!window.confirm(t('customers.regenerate_confirm'))) return;
    try {
      await apiClient.post(`/api/v1/reseller/tenants/${tenant.id}/api-key`, {
        api_key: `ACC-${crypto.randomUUID()}`,
      });
      showToast(t('customers.regenerate_success'));
      fetchTenants();
    } catch (err) {
      showToast(err.response?.data?.error || t('customers.form_error'), false);
    }
  };

  const copyKey = (key) => {
    navigator.clipboard?.writeText(key).then(() => showToast(t('customers.copied')));
  };

  const filtered = tenants.filter(t =>
    `${t.business_name} ${t.contact_name} ${t.email}`.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6 animate-fade-in">
      {toast && (
        <div className="glass-card flex items-center gap-3 px-6 py-4 border-slate-700/50">
          {toast.ok ? <CheckCircle2 className="w-5 h-5 text-emerald-500" /> : <AlertTriangle className="w-5 h-5 text-rose-500" />}
          <span className={`text-sm font-bold ${toast.ok ? 'text-emerald-500' : 'text-rose-500'}`}>{toast.msg}</span>
        </div>
      )}

      <div className="flex justify-between items-end flex-wrap gap-4">
        <div>
          <h2 className="text-3xl font-black text-white tracking-tight">{t('customers.title')}</h2>
          <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mt-1">{t('customers.subtitle')}</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-500 absolute left-4 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('customers.search')}
              className="bg-slate-900 border border-slate-800 rounded-2xl pl-12 pr-6 py-3 text-sm text-white focus:outline-none focus:border-rose-500 w-64 transition-all"
            />
          </div>
          <button onClick={() => setShowCreate(true)} className="premium-button flex items-center gap-2">
            <Plus className="w-4 h-4" /> {t('customers.add_new')}
          </button>
        </div>
      </div>

      <div className="glass-card overflow-hidden">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-slate-950/50 border-b border-slate-800">
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.business_name')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.contact')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.status')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.balance')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.subscription')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('customers.api_key')}</th>
              <th className="px-8 py-5 text-[10px] font-black text-slate-500 uppercase tracking-widest text-right">{t('customers.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800">
            {loading ? (
              <tr><td colSpan="7" className="px-8 py-20 text-center text-slate-500 font-bold italic">{t('customers.loading')}</td></tr>
            ) : filtered.length === 0 ? (
              <tr><td colSpan="7" className="px-8 py-20 text-center text-slate-500 font-bold italic">{t('customers.empty')}</td></tr>
            ) : filtered.map(tn => (
              <tr key={tn.id} className="hover:bg-slate-900/30 transition-all group">
                <td className="px-8 py-6">
                  <div className="font-bold text-white group-hover:text-rose-500 transition-colors">{tn.business_name}</div>
                  <div className="text-[10px] text-slate-500 font-mono mt-1">{tn.id}</div>
                </td>
                <td className="px-8 py-6">
                  <div className="text-sm text-slate-300">{tn.contact_name}</div>
                  <div className="text-xs text-slate-500">{tn.email}</div>
                </td>
                <td className="px-8 py-6">
                  <span className={`px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest ${tn.is_active ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}`}>
                    {tn.is_active ? t('customers.active') : t('customers.suspended')}
                  </span>
                </td>
                <td className="px-8 py-6">
                  <span className="text-sm font-bold text-white">{Number(tn.balance || 0).toLocaleString('en-US')}</span>
                  <span className="text-xs text-slate-500 ms-1">{t('common.currency')}</span>
                </td>
                <td className="px-8 py-6">
                  <span className={`px-3 py-1 rounded-full text-[10px] font-black uppercase tracking-widest ${
                    tn.subscription === 'active' ? 'bg-emerald-500/10 text-emerald-500' : 'bg-slate-800 text-slate-400'
                  }`}>
                    {tn.subscription === 'active' ? t('customers.active') : tn.subscription === 'grace_period' ? 'Grace' : t('customers.sub_none')}
                  </span>
                </td>
                <td className="px-8 py-6">
                  <div className="flex items-center gap-2">
                    <code className="text-xs text-slate-400 bg-slate-950 px-2 py-1 rounded border border-slate-800 max-w-[140px] truncate">
                      {tn.api_key_accounting ? '••••••••' : '—'}
                    </code>
                    {tn.api_key_accounting && (
                      <>
                        <button onClick={() => copyKey(tn.api_key_accounting)} className="p-2 hover:bg-slate-800 rounded-lg text-slate-500 hover:text-emerald-500 transition-all" title={t('customers.copy')}>
                          <Copy className="w-3 h-3" />
                        </button>
                        <button onClick={() => handleRegenerateKey(tn)} className="p-2 hover:bg-slate-800 rounded-lg text-slate-500 hover:text-white transition-all" title={t('customers.regenerate')}>
                          <RefreshCw className="w-3 h-3" />
                        </button>
                      </>
                    )}
                  </div>
                </td>
                <td className="px-8 py-6 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => setTransferTarget(tn)}
                      className="flex items-center gap-1 px-3 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest bg-slate-800 text-slate-300 hover:text-white transition-all"
                    >
                      <Send className="w-3 h-3" /> {t('customers.transfer')}
                    </button>
                    <button
                      onClick={() => openSubscribe(tn)}
                      className="flex items-center gap-1 px-3 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest bg-rose-600/10 text-rose-500 hover:bg-rose-600/20 transition-all"
                    >
                      <Package className="w-3 h-3" /> {t('customers.subscribe')}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Create Customer Modal */}
      {showCreate && (
        <Modal title={t('customers.add_new')} subtitle={t('customers.add_subtitle')} onClose={() => setShowCreate(false)}>
          <form onSubmit={handleCreate} className="space-y-4">
            {formError && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-rose-500 text-xs font-bold">{formError}</div>
            )}
            <FormField label={t('customers.form_business_name')}>
              <input required className={inputClass} value={form.business_name}
                onChange={(e) => setForm({ ...form, business_name: e.target.value })} />
            </FormField>
            <FormField label={t('customers.form_contact_name')}>
              <input required className={inputClass} value={form.contact_name}
                onChange={(e) => setForm({ ...form, contact_name: e.target.value })} />
            </FormField>
            <FormField label={t('customers.form_email')}>
              <input required type="email" className={inputClass} value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </FormField>
            <FormField label={t('customers.form_phone')}>
              <input className={inputClass} value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })} />
            </FormField>
            <FormField label={t('customers.form_password')}>
              <input required type="password" minLength={6} className={inputClass} value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </FormField>
            <button type="submit" disabled={formLoading}
              className="w-full bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white py-3 rounded-2xl font-black uppercase tracking-widest text-xs transition-all flex items-center justify-center gap-2">
              {formLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : t('customers.form_submit')}
            </button>
          </form>
        </Modal>
      )}

      {/* Transfer Modal */}
      {transferTarget && <TransferModal tenant={transferTarget} onClose={() => setTransferTarget(null)} onConfirm={handleTransfer} />}

      {/* Subscribe Modal */}
      {subscribeTarget && (
        <SubscribeModal tenant={subscribeTarget} packages={packages} onClose={() => setSubscribeTarget(null)} onConfirm={handleSubscribe} />
      )}
    </div>
  );
};

const TransferModal = ({ tenant, onClose, onConfirm }) => {
  const { t } = useTranslation();
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    setError('');
    if (!amount || Number(amount) <= 0) { setError(t('customers.transfer_error')); return; }
    setLoading(true);
    try {
      await onConfirm(amount, note);
    } catch (err) {
      setError(err.message || t('customers.transfer_error'));
      setLoading(false);
    }
  };

  return (
    <Modal title={`${t('customers.transfer_title')} — ${tenant.business_name}`} subtitle={t('customers.transfer_subtitle')} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        {error && <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-rose-500 text-xs font-bold">{error}</div>}
        <FormField label={t('customers.transfer_amount')}>
          <input required type="number" step="0.01" min="0.01" className={inputClass} value={amount}
            onChange={(e) => setAmount(e.target.value)} />
        </FormField>
        <FormField label={t('customers.transfer_note')}>
          <input className={inputClass} value={note} onChange={(e) => setNote(e.target.value)} />
        </FormField>
        <button type="submit" disabled={loading}
          className="w-full bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white py-3 rounded-2xl font-black uppercase tracking-widest text-xs transition-all flex items-center justify-center gap-2">
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />} {t('customers.transfer_confirm')}
        </button>
      </form>
    </Modal>
  );
};

const SubscribeModal = ({ tenant, packages, onClose, onConfirm }) => {
  const { t } = useTranslation();
  const [packageId, setPackageId] = useState('');
  const [autoRenew, setAutoRenew] = useState(true);
  const [loading, setLoading] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    if (!packageId) return;
    setLoading(true);
    await onConfirm(Number(packageId), autoRenew);
  };

  return (
    <Modal title={`${t('customers.subscribe_title')} — ${tenant.business_name}`} subtitle={t('customers.subscribe_subtitle')} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <FormField label={t('customers.subscribe_package')}>
          <select required className={inputClass} value={packageId} onChange={(e) => setPackageId(e.target.value)}>
            <option value="">—</option>
            {packages.map(pkg => (
              <option key={pkg.id} value={pkg.id}>
                {pkg.name} ({Number(pkg.price_usd).toLocaleString('en-US')} {t('common.currency')})
              </option>
            ))}
          </select>
        </FormField>
        <label className="flex items-center gap-3 text-sm text-slate-300">
          <input type="checkbox" checked={autoRenew} onChange={(e) => setAutoRenew(e.target.checked)}
            className="w-4 h-4 accent-rose-500" />
          {t('customers.subscribe_auto_renew')}
        </label>
        <button type="submit" disabled={loading || !packageId}
          className="w-full bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white py-3 rounded-2xl font-black uppercase tracking-widest text-xs transition-all flex items-center justify-center gap-2">
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Package className="w-4 h-4" />} {t('customers.subscribe_confirm')}
        </button>
      </form>
    </Modal>
  );
};

// ── Staff ─────────────────────────────────────────────────────────────────────
const StaffManager = () => {
  const { t } = useTranslation();
  const [staff, setStaff] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiClient.get('/api/v1/reseller/staff')
      .then(res => setStaff(res.data))
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h2 className="text-3xl font-black text-white tracking-tight">{t('team.title')}</h2>
        <p className="text-slate-500 text-xs font-bold uppercase tracking-widest mt-1">{t('team.subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {loading ? (
          <div className="col-span-full py-20 text-center text-slate-500">{t('team.loading')}</div>
        ) : staff.length === 0 ? (
          <div className="col-span-full py-20 text-center text-slate-500 glass-card">{t('team.empty')}</div>
        ) : staff.map(s => (
          <div key={s.id} className="glass-card p-6 flex items-center gap-4 group">
            <div className="w-12 h-12 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-500 font-black text-xl">
              {(s.full_name || s.user?.name || 'S').charAt(0)}
            </div>
            <div className="flex-1">
              <h4 className="text-white font-bold group-hover:text-rose-500 transition-all">{s.full_name || s.user?.name || '—'}</h4>
              <p className="text-[10px] text-slate-500 font-black uppercase tracking-widest">{s.role?.name || t('team.role')}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// ── Main Layout ──────────────────────────────────────────────────────────────
const MainApp = () => {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [walletBalance, setWalletBalance] = useState(null);
  const navigate = useNavigate();
  const { t } = useTranslation();

  useEffect(() => {
    apiClient.get('/api/v1/reseller/balance')
      .then(res => setWalletBalance(res.data?.balance ?? null))
      .catch(() => {});
  }, []);

  const handleTabClick = (tabId) => {
    setActiveTab(tabId);
    setIsMobileMenuOpen(false);
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard': return <Dashboard />;
      case 'tenants': return <TenantsList />;
      case 'staff': return <StaffManager />;
      case 'packages': return <Packages />;
      default: return <Dashboard />;
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex font-sans w-full selection:bg-rose-500 selection:text-white">
      <aside className={`w-72 bg-slate-950 border-r border-slate-900 flex flex-col fixed inset-y-0 start-0 z-50 transition-transform duration-300 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        <div className="h-24 flex items-center justify-between px-8 border-b border-slate-900">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-rose-600 rounded-2xl flex items-center justify-center shadow-lg shadow-rose-900/40">
              <ShieldCheck className="w-6 h-6 text-white" />
            </div>
            <span className="text-xl font-black text-white tracking-tighter uppercase">{t('brand.name')}<span className="text-rose-500">{t('brand.suffix')}</span></span>
          </div>
          <button className="md:hidden text-slate-400 hover:text-white" onClick={() => setIsMobileMenuOpen(false)}>
            <X className="w-6 h-6" />
          </button>
        </div>

        <nav className="flex-1 py-8">
          <SidebarItem icon={LayoutDashboard} label={t('nav.overview')} active={activeTab === 'dashboard'} onClick={() => handleTabClick('dashboard')} />
          <SidebarItem icon={Building2} label={t('nav.customers')} active={activeTab === 'tenants'} onClick={() => handleTabClick('tenants')} />
          <SidebarItem icon={Package} label={t('nav.packages')} active={activeTab === 'packages'} onClick={() => handleTabClick('packages')} />
          <SidebarItem icon={Users} label={t('nav.team')} active={activeTab === 'staff'} onClick={() => handleTabClick('staff')} />
        </nav>

        <div className="p-6 border-t border-slate-900 space-y-4">
          <div className="glass-card p-4 flex items-center gap-3">
            <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
              <Wallet className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest">{t('dash.wallet_balance')}</p>
              <p className="text-lg font-black text-white">
                {walletBalance != null ? Number(walletBalance).toLocaleString('en-US') : '—'}
                <span className="text-xs text-slate-500 ms-1">{t('common.currency')}</span>
              </p>
            </div>
          </div>
          <button
            onClick={() => {
              localStorage.removeItem('reseller_token');
              navigate('/login');
            }}
            className="w-full flex items-center gap-4 px-6 py-4 text-slate-500 hover:text-rose-500 transition-all rounded-2xl hover:bg-rose-500/5 group"
          >
            <LogOut className="w-5 h-5 group-hover:scale-110 transition-transform" />
            <span className="font-bold text-sm">{t('nav.signout')}</span>
          </button>
        </div>
      </aside>

      <main className="flex-1 md:ms-72 p-12 w-full">
        <div className="flex justify-end mb-6">
          <LanguageSwitcher />
        </div>
        {renderContent()}
      </main>
    </div>
  );
};

export default function App() {
    return (
        <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/*" element={
                <ProtectedRoute>
                    <MainApp />
                </ProtectedRoute>
            } />
        </Routes>
    );
}
