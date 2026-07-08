import React, { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { Menu, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useAuth } from './contexts/AuthContext';
import { useSocket } from './contexts/SocketContext';
import { initializeCSRF } from './api/axios';
import LanguageSwitcher from './components/LanguageSwitcher';
import Login from './components/Login';
import Register from './components/Register';
import ProtectedRoute from './components/ProtectedRoute';
import GatewayManager from './components/GatewayManager';
import WaitingMessages from './components/WaitingMessages';
import DeviceManager from './components/DeviceManager';
import DashboardView from './components/DashboardView';
import ExcelImport from './components/ExcelImport';
import MessageHistory from './components/MessageHistory';
import SettingsManager from './components/SettingsManager';
import SendMessage from './components/SendMessage';
import HistoryReports from './components/HistoryReports';
import Customers from './components/Customers';
import FinanceDashboard from './components/FinanceDashboard';
const SidebarItem = ({ label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center px-6 py-4 transition-all ${active
      ? 'text-rose-500 bg-rose-500/5 border-r-4 border-rose-500'
      : 'text-slate-400 hover:text-slate-100 hover:bg-slate-900'
      } `}
  >
    <span className="font-medium text-sm">{label}</span>
  </button>
);

function MainApp() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const { user, tenant, logout } = useAuth();
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  useEffect(() => {
    document.documentElement.dir = i18n.language === 'ar' ? 'rtl' : 'ltr';
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const handleTabClick = (tabId) => {
    setActiveTab(tabId);
    setIsMobileMenuOpen(false);
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard': return <DashboardView />;
      case 'send': return <SendMessage />;
      case 'customers': return <Customers />;
      case 'excel': return <ExcelImport />;
      case 'devices': return <DeviceManager />;
      case 'packages': return <FinanceDashboard />;
      case 'gateways': return <GatewayManager />;
      case 'history': return <HistoryReports />;
      case 'settings': return <SettingsManager />;
      default: return <div className="p-12 text-center text-slate-500 font-bold">{t('app.coming_soon')}</div>;
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex font-sans w-full">
      {/* Mobile overlay */}
      {isMobileMenuOpen && (
        <div 
          className="fixed inset-0 bg-black/60 z-40 md:hidden" 
          onClick={() => setIsMobileMenuOpen(false)} 
        />
      )}

      <aside className={`w-64 bg-slate-950 border-r border-slate-900 flex flex-col fixed inset-y-0 start-0 z-50 transition-transform duration-300 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full rtl:translate-x-full md:translate-x-0 rtl:md:translate-x-0'}`}>
        <div className="h-20 flex items-center justify-between px-8 border-b border-slate-900">
          <span className="text-xl font-bold text-white tracking-tight truncate uppercase">{tenant?.business_name || 'HUDHUD'}</span>
          <button className="md:hidden text-slate-400 hover:text-white" onClick={() => setIsMobileMenuOpen(false)}>
            <X className="w-6 h-6" />
          </button>
        </div>

        <nav className="flex-1 py-4 overflow-y-auto">
          <SidebarItem label={t('nav.dashboard') || 'Dashboard'} active={activeTab === 'dashboard'} onClick={() => handleTabClick('dashboard')} />
          <SidebarItem label={t('nav.history')} active={activeTab === 'history'} onClick={() => handleTabClick('history')} />
          <SidebarItem label={t('nav.customers')} active={activeTab === 'customers'} onClick={() => handleTabClick('customers')} />
          <SidebarItem label={t('nav.excel')} active={activeTab === 'excel'} onClick={() => handleTabClick('excel')} />
          <SidebarItem label={t('nav.devices')} active={activeTab === 'devices'} onClick={() => handleTabClick('devices')} />
          <SidebarItem label={t('nav.packages')} active={activeTab === 'packages'} onClick={() => handleTabClick('packages')} />
          <SidebarItem label={t('nav.gateways')} active={activeTab === 'gateways'} onClick={() => handleTabClick('gateways')} />
          <SidebarItem label={t('nav.settings')} active={activeTab === 'settings'} onClick={() => handleTabClick('settings')} />
        </nav>

        <div className="p-4 border-t border-slate-900 flex flex-col gap-3">
          <LanguageSwitcher />
          <button onClick={handleLogout} className="w-full text-left px-4 py-2 text-slate-400 hover:text-rose-500 transition-colors">
            {t('nav.logout')}
          </button>
        </div>
      </aside>

      <main className="flex-1 md:ms-64 p-4 lg:p-8 w-full max-w-full overflow-x-hidden min-h-screen">
        <div className="md:hidden flex justify-between items-center mb-6 bg-slate-900 p-4 rounded-xl border border-slate-800">
            <span className="text-xl font-bold tracking-tight text-white uppercase">{tenant?.business_name || 'HUDHUD CLIENT'}</span>
            <button onClick={() => setIsMobileMenuOpen(true)} className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors">
                <Menu className="w-5 h-5 text-white" />
            </button>
        </div>
        {renderContent()}
      </main>
    </div>
  );
}

function App() {
  const { user, loading } = useAuth();
  const [csrfInitialized, setCsrfInitialized] = useState(false);

  // Initialize CSRF protection on app mount (only once)
  useEffect(() => {
    if (!csrfInitialized) {
      initializeCSRF()
        .catch(error => {
          console.error('Failed to initialize CSRF protection:', error);
          // Continue anyway - CSRF will be fetched on first request
        })
        .finally(() => {
          setCsrfInitialized(true);
        });
    }
  }, [csrfInitialized]);

  if (loading) return <div className="min-h-screen bg-slate-950 flex items-center justify-center text-white">Loading...</div>;

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/*"
        element={
          <ProtectedRoute allowedTypes={['TENANT', 'STAFF']}>
            <MainApp />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}

export default App;
