import React, { useState, useEffect, useCallback } from 'react';
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { Building2, Globe, Activity, Zap } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useAuth } from './contexts/AuthContext';
import apiClient, { initializeAppCSRF } from './api/axios';
import LanguageSwitcher from './components/LanguageSwitcher';
import Login from './components/Login';
import ProtectedRoute from './components/ProtectedRoute';
import StaffManager from './components/StaffManager';
import TariffManager from './components/TariffManager';
import FinanceManager from './components/FinanceManager';
import SupportCenter from './components/SupportCenter';
import Gateways from './components/Gateways';
import TenantManager from './components/TenantManager';
import Reports from './components/Reports';
import AuditLogManager from './components/AuditLogManager';
import RoleManager from './components/RoleManager';
import Overview from './pages/Overview';
import CompaniesList from './pages/Companies/CompaniesList';
import ResellersList from './pages/Resellers/ResellersList';
import ResellerDetails from './pages/Resellers/ResellerDetails';
import CompanyDetails from './pages/Companies/CompanyDetails';
import ServersList from './pages/Servers/ServersList';
import AlertsList from './pages/Alerts/AlertsList';
import AlertConfig from './pages/Alerts/AlertConfig';
import DistributionConfig from './pages/Distribution/DistributionConfig';
import SignalPushMonitor from './pages/SignalPush/Monitor';
import NotificationSettingsPage from './pages/NotificationSettingsPage';
import Cashboxes from './pages/Accounting/Cashboxes';
import ReceiptVouchers from './pages/Accounting/ReceiptVouchers';
import PaymentVouchers from './pages/Accounting/PaymentVouchers';
import { API_BASE_URL } from './config';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }
  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }
  componentDidCatch(error, errorInfo) {
    console.error("ErrorBoundary caught an error", error, errorInfo);
  }
  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-slate-950 flex items-center justify-center p-8 text-center">
          <div className="bg-slate-900 border border-slate-800 p-8 rounded-3xl max-w-lg">
            <h1 className="text-2xl font-bold text-rose-500 mb-4">Something went wrong</h1>
            <pre className="text-xs text-slate-400 bg-slate-950 p-4 rounded-xl overflow-auto text-left mb-6">
              {this.state.error?.message}
            </pre>
            <button 
              onClick={() => window.location.reload()}
              className="px-6 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold transition-all"
            >
              Reload Application
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

function Dashboard() {
  const [stats, setStats] = useState({ total_tenants: 0, active_devices: 0, queued_messages: 0 });
  const { user } = useAuth();
  const { t } = useTranslation();

  const fetchStats = useCallback(async () => {
    try {
      const res = await apiClient.get('/api/v1/admin/stats');
      setStats(res.data);
    } catch (err) {
      console.error("Failed to fetch stats", err);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return (
    <div className="p-10 space-y-10 animate-in fade-in duration-700">
      {/* Welcome Hero */}
      <div className="bg-gradient-to-r from-rose-600 to-rose-500 rounded-[2.5rem] p-12 text-white shadow-2xl shadow-rose-900/20 relative overflow-hidden group">
        <div className="absolute top-0 right-0 p-12 opacity-10 group-hover:scale-110 transition-transform duration-700">
            <Building2 className="w-64 h-64" />
        </div>
        <div className="relative z-10">
            <h1 className="text-5xl font-black tracking-tighter mb-2">System Pulse</h1>
            <p className="text-rose-100 font-bold uppercase tracking-widest text-sm opacity-80">
                {user?.user_type === 'SUPER_ADMIN' ? 'Strategic Command Center' : 'Operations Dashboard'}
            </p>
            <div className="mt-8 flex gap-4">
                <div className="px-6 py-2 bg-white/10 backdrop-blur-md rounded-xl border border-white/10 text-xs font-black uppercase tracking-widest">
                    Live Nodes: 42
                </div>
                <div className="px-6 py-2 bg-white/10 backdrop-blur-md rounded-xl border border-white/10 text-xs font-black uppercase tracking-widest text-emerald-300">
                    Network: Stable
                </div>
            </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <DashboardStat icon={Globe} label="Total Assets" value={stats.total_tenants || 0} color="indigo" />
        <DashboardStat icon={Activity} label="Active Links" value={stats.active_devices || 0} color="emerald" />
        <DashboardStat icon={Zap} label="Message Queue" value={stats.queued_messages || 0} color="amber" />
      </div>

      {/* Quick Actions / Activity Feed Placeholder */}
      <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] p-10 shadow-2xl">
        <h3 className="text-xl font-black text-white mb-6 flex items-center gap-4">
            <div className="p-2 bg-rose-500/20 rounded-lg text-rose-500">
                <Activity className="w-5 h-5" />
            </div>
            Global Infrastructure Monitor
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {['Gateway A-1', 'Server Primary', 'API Node East', 'Queue Cluster'].map(item => (
                <div key={item} className="p-6 bg-slate-950 border border-slate-800 rounded-2xl flex items-center justify-between group hover:border-rose-500/30 transition-all">
                    <span className="text-sm font-bold text-slate-400 group-hover:text-white">{item}</span>
                    <div className="w-2 h-2 bg-emerald-500 rounded-full shadow-[0_0_8px_rgba(16,185,129,0.5)]" />
                </div>
            ))}
        </div>
      </div>
    </div>
  );
}

function DashboardStat({ icon: Icon, label, value, color }) {
    const colors = {
        indigo: 'text-indigo-400 bg-indigo-500/10 border-indigo-500/20',
        emerald: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20',
        amber: 'text-amber-400 bg-amber-500/10 border-amber-500/20',
    };
    return (
        <div className={`p-10 rounded-[2rem] border transition-all hover:scale-[1.02] bg-slate-900 shadow-xl ${colors[color]}`}>
            <div className="flex justify-between items-start mb-6">
                <div className="p-4 bg-slate-950 rounded-2xl border border-white/5">
                    <Icon className="w-8 h-8" />
                </div>
            </div>
            <div className="text-5xl font-black text-white tracking-tighter mb-2">{value}</div>
            <p className="text-[10px] font-black uppercase tracking-[0.3em] text-slate-500">{label}</p>
        </div>
    );
}


const NAV_SECTIONS = [
  {
    id: 'general',
    labelKey: 'nav.general',
    items: [
      { id: 'dashboard', labelKey: 'nav.dashboard', path: '/' },
      { id: 'overview', labelKey: 'nav.overview', path: '/overview' },
    ],
  },
  {
    id: 'customers',
    labelKey: 'nav.customers',
    items: [
      { id: 'companies', labelKey: 'nav.tenants', path: '/companies' },
      { id: 'resellers', labelKey: 'nav.resellers', path: '/resellers' },
    ],
  },
  {
    id: 'financial',
    labelKey: 'nav.financial',
    items: [
      { id: 'finance', labelKey: 'nav.finance', path: '/finance' },
      { id: 'cashboxes', labelKey: 'nav.cashboxes', path: '/accounting/cashboxes' },
      { id: 'receipts', labelKey: 'nav.receipts', path: '/accounting/receipts' },
      { id: 'payments', labelKey: 'nav.payments', path: '/accounting/payments' },
    ],
  },
  {
    id: 'infrastructure',
    labelKey: 'nav.infrastructure',
    items: [
      { id: 'servers', labelKey: 'nav.servers', path: '/servers' },
      { id: 'gateways', labelKey: 'nav.gateways', path: '/gateways' },
      { id: 'alerts', labelKey: 'nav.alerts', path: '/alerts' },
    ],
  },
  {
    id: 'security',
    labelKey: 'nav.security',
    items: [
      { id: 'staff', labelKey: 'nav.staff', path: '/staff' },
      { id: 'roles', labelKey: 'nav.roles', path: '/roles' },
      { id: 'audit', labelKey: 'nav.audit', path: '/audit' },
    ],
  },
  {
    id: 'system',
    labelKey: 'nav.system',
    items: [
      { id: 'tariffs', labelKey: 'nav.tariffs', path: '/tariffs' },
      { id: 'support', labelKey: 'nav.support', path: '/support' },
      { id: 'notifications', labelKey: 'nav.notifications', path: '/notifications' },
    ],
  },
  {
    id: 'reports',
    labelKey: 'nav.reports_main',
    items: [
      { id: 'reports', labelKey: 'nav.reports', path: '/reports' },
    ],
  },
];

const routeToSubTab = (pathname) => {
  const first = pathname.split('/')[1];
  const alias = { distribution: 'companies', signalpush: 'companies', tenants: 'companies' };
  if (first === 'accounting') {
    const second = pathname.split('/')[2];
    if (second === 'cashboxes') return 'cashboxes';
    if (second === 'receipts') return 'receipts';
    if (second === 'payments') return 'payments';
  }
  return alias[first] || first || 'dashboard';
};

function MainLayout({ children }) {
  const location = useLocation();
  const { logout, hasPermission } = useAuth();
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  const canViewFinance = hasPermission('view_finances') || hasPermission('approve_payments');
  const visibleNavSections = NAV_SECTIONS.filter(s => s.id !== 'financial' || canViewFinance);

  useEffect(() => {
    document.documentElement.dir = i18n.language === 'ar' ? 'rtl' : 'ltr';
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const activeSub = routeToSubTab(location.pathname);
  const activeSection = visibleNavSections.find((s) => s.items.some((i) => i.id === activeSub)) || visibleNavSections[0];

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col font-sans w-full notranslate" translate="no">
      <div className="sticky top-0 z-40 bg-slate-900 border-b border-slate-800 shadow-lg shadow-black/20">
        <header className="flex items-center justify-between px-6 h-16">
          <h1 className="text-xl font-bold tracking-tight text-white">HUDHUD ADMIN</h1>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <button onClick={handleLogout} className="text-left text-rose-400 hover:text-rose-300 font-bold py-2">
              {t('nav.logout')}
            </button>
          </div>
        </header>

        <nav className="flex gap-2 overflow-x-auto px-6 py-3 notranslate" translate="no" aria-label="Main sections">
          {visibleNavSections.map((section) => {
            const isActive = section.id === activeSection.id;
            return (
              <button
                key={section.id}
                onClick={() => navigate(section.items[0].path)}
                className={`shrink-0 px-4 py-2 rounded-lg font-bold text-sm transition-all ${
                  isActive
                    ? 'bg-rose-600 text-white shadow-lg shadow-rose-900/20'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                }`}
              >
                {t(section.labelKey)}
              </button>
            );
          })}
        </nav>

        <div className="flex gap-2 overflow-x-auto px-6 py-2 bg-slate-950 border-t border-slate-800/60 notranslate" translate="no" aria-label="Sub sections">
          {activeSection.items.map((item) => {
            const isActive = item.id === activeSub;
            return (
              <button
                key={item.id}
                onClick={() => navigate(item.path)}
                className={`shrink-0 px-3 py-1.5 rounded-md text-xs font-bold transition-all ${
                  isActive
                    ? 'bg-rose-500/20 text-rose-400 border border-rose-500/40'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white border border-transparent'
                }`}
              >
                {t(item.labelKey)}
              </button>
            );
          })}
        </div>
      </div>

      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}

// Guards the financial operations section (cashboxes + vouchers + finance).
// Read access requires view_finances; write access requires approve_payments.
function FinanceRouteGuard({ children }) {
  const { hasPermission } = useAuth();
  const { t } = useTranslation();
  const allowed = hasPermission('view_finances') || hasPermission('approve_payments');

  if (!allowed) {
    return (
      <div className="p-12 flex items-center justify-center min-h-[60vh]">
        <div className="text-center max-w-md">
          <div className="p-6 bg-rose-500/10 text-rose-400 border border-rose-500/20 rounded-3xl">
            <p className="text-2xl font-black mb-2">{t('permissions.no_access_title')}</p>
            <p className="text-sm font-bold text-slate-300">{t('permissions.no_access_message')}</p>
          </div>
        </div>
      </div>
    );
  }

  return children;
}

function App() {
  const { user, loading } = useAuth();

  // Initialize CSRF protection on app mount
  useEffect(() => {
    initializeAppCSRF().catch(error => {
      console.error('Failed to initialize CSRF protection:', error);
    });
  }, []);

  if (loading) return <div className="min-h-screen bg-slate-950 flex items-center justify-center text-white">Loading...</div>;

  return (
    <ErrorBoundary>
      <Routes>
        <Route path="/login" element={user ? <Navigate to="/" replace /> : <Login />} />
        <Route
          path="/*"
          element={
            <ProtectedRoute allowedTypes={['SUPER_ADMIN', 'STAFF']}>
              <MainLayout>
                <Routes>
                  <Route path="/" element={<Dashboard />} />
                  <Route path="/overview" element={<Overview />} />
                  <Route path="/companies" element={<CompaniesList />} />
                  <Route path="/companies/:tenantId" element={<CompanyDetails />} />
                  <Route path="/resellers" element={<ResellersList />} />
                  <Route path="/resellers/:id" element={<ResellerDetails />} />
                  <Route path="/servers" element={<ServersList />} />
                  <Route path="/alerts" element={<AlertsList />} />
                  <Route path="/alerts/config" element={<AlertConfig />} />
                  <Route path="/distribution" element={<Navigate to="/companies" replace />} />
                  <Route path="/distribution/:tenantId" element={<DistributionConfig />} />
                  <Route path="/signalpush" element={<Navigate to="/companies" replace />} />
                  <Route path="/signalpush/:tenantId" element={<SignalPushMonitor />} />
                  <Route path="/tenants" element={<TenantManager />} />
                  <Route path="/tariffs" element={<TariffManager />} />
                  <Route path="/finance" element={<FinanceRouteGuard><FinanceManager /></FinanceRouteGuard>} />
                  <Route path="/accounting/cashboxes" element={<FinanceRouteGuard><Cashboxes /></FinanceRouteGuard>} />
                  <Route path="/accounting/receipts" element={<FinanceRouteGuard><ReceiptVouchers /></FinanceRouteGuard>} />
                  <Route path="/accounting/payments" element={<FinanceRouteGuard><PaymentVouchers /></FinanceRouteGuard>} />
                  <Route path="/reports" element={<Reports />} />
                  <Route path="/support" element={<SupportCenter />} />
                  <Route path="/gateways" element={<Gateways />} />
                  <Route path="/staff" element={<StaffManager />} />
                  <Route path="/roles" element={<RoleManager />} />
                  <Route path="/audit" element={<AuditLogManager />} />
                  <Route path="/notifications" element={<NotificationSettingsPage />} />
                  <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
              </MainLayout>
            </ProtectedRoute>
          }
        />
      </Routes>
    </ErrorBoundary>
  );
}

export default App;
