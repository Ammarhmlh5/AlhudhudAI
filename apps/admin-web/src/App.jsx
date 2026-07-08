import React, { useState, useEffect, useCallback } from 'react';
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { Menu, X, Building2, Globe, Activity, Zap } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useAuth } from './contexts/AuthContext';
import apiClient, { initializeCSRF } from './api/axios';
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
import CompanyDetails from './pages/Companies/CompanyDetails';
import ServersList from './pages/Servers/ServersList';
import AlertsList from './pages/Alerts/AlertsList';
import AlertConfig from './pages/Alerts/AlertConfig';
import DistributionConfig from './pages/Distribution/DistributionConfig';
import SignalPushMonitor from './pages/SignalPush/Monitor';
import NotificationSettingsPage from './pages/NotificationSettingsPage';
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


function MainLayout({ children }) {
  const location = useLocation();
  const [activeTab, setActiveTab] = useState('dashboard');
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { logout } = useAuth();
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  // Sync activeTab with URL
  useEffect(() => {
    const path = location.pathname.split('/')[1] || 'dashboard';
    setActiveTab(path);
  }, [location.pathname]);

  useEffect(() => {
    document.documentElement.dir = i18n.language === 'ar' ? 'rtl' : 'ltr';
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const NavItem = useCallback(({ id, label }) => (
    <button
      onClick={() => {
        setIsMobileMenuOpen(false);
        navigate(`/${id === 'dashboard' ? '' : id}`);
      }}
      className={`w-full text-left px-4 py-3 rounded-lg transition-all ${activeTab === id
        ? 'bg-rose-600 text-white shadow-lg shadow-rose-900/20'
        : 'text-slate-400 hover:bg-slate-800 hover:text-white'
        }`}
    >
      <span className="font-bold text-sm">{label}</span>
    </button>
  ), [activeTab, navigate]);

  return (
    <div className="min-h-screen bg-slate-950 flex font-sans w-full">
      {/* Mobile overlay */}
      {isMobileMenuOpen && (
        <div 
          className="fixed inset-0 bg-black/60 z-40 md:hidden" 
          onClick={() => setIsMobileMenuOpen(false)} 
        />
      )}

      <aside className={`w-64 bg-slate-900 border-r border-slate-800 p-6 flex flex-col fixed inset-y-0 start-0 z-50 transition-transform duration-300 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full rtl:translate-x-full md:translate-x-0 rtl:md:translate-x-0'}`}>
        <div className="mb-8 flex justify-between items-center">
          <h1 className="text-xl font-bold text-white">HUDHUD ADMIN</h1>
          <button className="md:hidden text-slate-400 hover:text-white" onClick={() => setIsMobileMenuOpen(false)}>
            <X className="w-6 h-6" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto pb-4">
          <NavItem id="dashboard" label={t('nav.dashboard')} />
          <NavItem id="overview" label={t('nav.overview')} />
          <NavItem id="companies" label={t('nav.tenants')} />
          <NavItem id="finance" label={t('nav.finance')} />
          <NavItem id="reports" label={t('nav.reports')} />
          <div className="pt-4 pb-2 px-4 text-[10px] font-bold text-slate-500 uppercase tracking-widest">Infrastructure</div>
          <NavItem id="servers" label={t('nav.servers')} />
          <NavItem id="gateways" label={t('nav.gateways')} />
          <NavItem id="alerts" label={t('nav.alerts')} />
          <div className="pt-4 pb-2 px-4 text-[10px] font-bold text-slate-500 uppercase tracking-widest">Security & Staff</div>
          <NavItem id="staff" label={t('nav.staff')} />
          <NavItem id="roles" label={t('nav.roles')} />
          <NavItem id="audit" label={t('nav.audit')} />
          <div className="pt-4 pb-2 px-4 text-[10px] font-bold text-slate-500 uppercase tracking-widest">System</div>
          <NavItem id="tariffs" label={t('nav.tariffs')} />
          <NavItem id="support" label={t('nav.support')} />
          <NavItem id="notifications" label={t('nav.notifications')} />
        </nav>

        <div className="p-4 flex flex-col gap-3 border-t border-slate-800">
            <LanguageSwitcher />
            <button onClick={handleLogout} className="text-left text-rose-400 hover:text-rose-300 font-bold py-2">
              {t('nav.logout')}
            </button>
        </div>
      </aside>

      <div className="flex-1 md:ms-64 w-full max-w-full overflow-x-hidden min-h-screen flex flex-col">
        <div className="md:hidden m-4 flex justify-between items-center bg-slate-900 p-4 rounded-xl border border-slate-800">
            <span className="text-xl font-bold tracking-tight text-white">HUDHUD ADMIN</span>
            <button onClick={() => setIsMobileMenuOpen(true)} className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors">
                <Menu className="w-5 h-5 text-white" />
            </button>
        </div>
        <div className="overflow-auto flex-1">
            {children}
        </div>
      </div>
    </div>
  );
}

function App() {
  const { user, loading } = useAuth();

  // Initialize CSRF protection on app mount
  useEffect(() => {
    initializeCSRF().catch(error => {
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
                  <Route path="/servers" element={<ServersList />} />
                  <Route path="/alerts" element={<AlertsList />} />
                  <Route path="/alerts/config" element={<AlertConfig />} />
                  <Route path="/distribution" element={<Navigate to="/companies" replace />} />
                  <Route path="/distribution/:tenantId" element={<DistributionConfig />} />
                  <Route path="/signalpush" element={<Navigate to="/companies" replace />} />
                  <Route path="/signalpush/:tenantId" element={<SignalPushMonitor />} />
                  <Route path="/tenants" element={<TenantManager />} />
                  <Route path="/tariffs" element={<TariffManager />} />
                  <Route path="/finance" element={<FinanceManager />} />
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
