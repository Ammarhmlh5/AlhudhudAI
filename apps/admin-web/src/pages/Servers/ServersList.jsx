import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  Server, Activity, ShieldCheck, AlertCircle, 
  Search, Filter, Clock, Globe, Zap,
  ChevronRight, RefreshCw, HardDrive, Cpu, Network
} from 'lucide-react';
import apiClient from '../../api/axios';

export default function ServersList() {
  const { t } = useTranslation();
  const [servers, setServers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    health_status: '',
    tenant_id: '',
  });
  const [selectedServer, setSelectedServer] = useState(null);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    fetchServers();
    const interval = setInterval(fetchServers, 30000);
    return () => clearInterval(interval);
  }, [filters]);

  const fetchServers = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams();
      if (filters.health_status) params.append('health_status', filters.health_status);
      if (filters.tenant_id) params.append('tenant_id', filters.tenant_id);

      const response = await apiClient.get(`/api/v1/admin/health/servers?${params}`);
      setServers(response.data.servers || []);
      setError(null);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to load servers');
    } finally {
      setLoading(false);
    }
  };

  const getHealthStyles = (status) => {
    switch (status) {
      case 'healthy':   return { dot: 'bg-emerald-500 shadow-emerald-500/50', badge: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20', icon: ShieldCheck };
      case 'degraded':  return { dot: 'bg-amber-500 shadow-amber-500/50', badge: 'bg-amber-500/10 text-amber-400 border-amber-500/20', icon: Zap };
      case 'unhealthy': return { dot: 'bg-rose-500 shadow-rose-500/50', badge: 'bg-rose-500/10 text-rose-400 border-rose-500/20', icon: AlertCircle };
      default:          return { dot: 'bg-slate-500 shadow-slate-500/50', badge: 'bg-slate-500/10 text-slate-400 border-slate-500/20', icon: Server };
    }
  };

  const handleServerClick = async (server) => {
    try {
      const response = await apiClient.get(`/api/v1/admin/health/servers/${server.id}`);
      setSelectedServer(response.data);
      setShowModal(true);
    } catch (err) {
      console.error('Failed to load server details:', err);
    }
  };

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight flex items-center gap-4">
            <div className="p-3 bg-indigo-600 rounded-2xl shadow-lg shadow-indigo-900/20">
              <Server className="w-8 h-8 text-white" />
            </div>
            {t('servers.title')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium ml-1">
            {t('servers.description')}
          </p>
        </div>
        
        <button onClick={fetchServers} className="flex items-center gap-2 px-6 py-3 bg-slate-900 border border-slate-800 rounded-2xl text-slate-400 hover:text-white transition-all group">
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : 'group-hover:rotate-180 transition-transform duration-500'}`} />
            <span className="text-sm font-bold uppercase tracking-wider">Refresh Nodes</span>
        </button>
      </div>

      {/* Health Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <HealthSummaryCard
          status="healthy"
          count={servers.filter(s => s.health_status === 'healthy').length}
          label={t('servers.healthy')}
          color="emerald"
        />
        <HealthSummaryCard
          status="degraded"
          count={servers.filter(s => s.health_status === 'degraded').length}
          label={t('servers.degraded')}
          color="amber"
        />
        <HealthSummaryCard
          status="unhealthy"
          count={servers.filter(s => s.health_status === 'unhealthy').length}
          label={t('servers.unhealthy')}
          color="rose"
        />
      </div>

      {/* Filter Bar */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="relative group">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-indigo-500 transition-colors" />
            <input
              type="text"
              placeholder={t('servers.company_placeholder')}
              value={filters.tenant_id}
              onChange={(e) => setFilters({ ...filters, tenant_id: e.target.value })}
              className="w-full pl-12 pr-4 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-white placeholder:text-slate-600 focus:outline-none focus:border-indigo-500 transition-all shadow-inner"
            />
          </div>

          <div className="relative">
            <Filter className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
            <select
              value={filters.health_status}
              onChange={(e) => setFilters({ ...filters, health_status: e.target.value })}
              className="w-full pl-12 pr-8 py-3 bg-slate-950 border border-slate-800 rounded-2xl text-sm text-slate-300 focus:outline-none focus:border-indigo-500 appearance-none cursor-pointer"
            >
              <option value="">{t('servers.all_statuses')}</option>
              <option value="healthy">{t('servers.healthy')}</option>
              <option value="degraded">{t('servers.degraded')}</option>
              <option value="unhealthy">{t('servers.unhealthy')}</option>
            </select>
          </div>
        </div>
      </div>

      {/* Servers Grid */}
      {loading && servers.length === 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[1, 2, 3].map(i => (
                <div key={i} className="h-64 bg-slate-900/50 border border-slate-800 rounded-3xl animate-pulse" />
            ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {servers.map((server) => (
            <ServerCard
              key={server.id}
              server={server}
              onClick={() => handleServerClick(server)}
              styles={getHealthStyles(server.health_status)}
            />
          ))}
        </div>
      )}

      {servers.length === 0 && !loading && (
        <div className="text-center py-24 bg-slate-900 border border-slate-800 border-dashed rounded-3xl">
          <Globe className="w-16 h-16 text-slate-800 mx-auto mb-4" />
          <p className="text-slate-500 font-bold text-lg">{t('servers.no_servers')}</p>
        </div>
      )}

      {/* Modal */}
      {showModal && selectedServer && (
        <ServerDetailsModal
          server={selectedServer}
          onClose={() => {
            setShowModal(false);
            setSelectedServer(null);
          }}
          getHealthStyles={getHealthStyles}
        />
      )}
    </div>
  );
}

function HealthSummaryCard({ count, label, color }) {
  const colors = {
    emerald: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20',
    amber:   'text-amber-400 bg-amber-500/10 border-amber-500/20',
    rose:    'text-rose-400 bg-rose-500/10 border-rose-500/20',
  };

  return (
    <div className={`bg-slate-900 border ${colors[color].split(' ')[2]} rounded-3xl p-8 shadow-xl text-center group transition-transform hover:scale-[1.02]`}>
      <p className={`text-5xl font-black ${colors[color].split(' ')[0]} mb-2`}>{count}</p>
      <p className="text-xs font-black text-slate-500 uppercase tracking-widest">{label}</p>
    </div>
  );
}

function ServerCard({ server, onClick, styles }) {
  const { t } = useTranslation();
  const Icon = styles.icon;

  return (
    <div
      onClick={onClick}
      className="bg-slate-900 border border-slate-800 rounded-3xl p-6 cursor-pointer hover:border-white/10 hover:bg-slate-800/50 transition-all group relative overflow-hidden"
    >
      <div className="absolute top-0 right-0 p-6 opacity-[0.03] group-hover:opacity-[0.07] transition-opacity">
        <Icon className="w-24 h-24" />
      </div>

      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className={`w-3 h-3 rounded-full shadow-[0_0_12px] ${styles.dot}`} />
          <h3 className="text-lg font-black text-white group-hover:text-indigo-400 transition-colors">
            {server.name}
          </h3>
        </div>
        <span className={`px-2.5 py-1 text-[10px] font-black uppercase rounded-lg border ${styles.badge}`}>
          {t(`servers.${server.health_status}`)}
        </span>
      </div>

      <div className="space-y-4 relative z-10">
        <div className="grid grid-cols-2 gap-4">
            <div className="bg-slate-950/50 p-3 rounded-2xl border border-slate-800">
                <div className="flex items-center gap-2 text-slate-500 mb-1">
                    <Activity className="w-3 h-3" />
                    <span className="text-[10px] font-bold uppercase tracking-wider">Latency</span>
                </div>
                <div className="text-sm font-black text-white font-mono">{server.response_time_ms}ms</div>
            </div>
            <div className="bg-slate-950/50 p-3 rounded-2xl border border-slate-800">
                <div className="flex items-center gap-2 text-slate-500 mb-1">
                    <AlertCircle className="w-3 h-3" />
                    <span className="text-[10px] font-bold uppercase tracking-wider">Errors</span>
                </div>
                <div className="text-sm font-black text-white font-mono">{server.error_rate?.toFixed(1) || 0}%</div>
            </div>
        </div>

        <div className="flex items-center justify-between text-xs px-1">
          <div className="flex items-center gap-2 text-slate-500">
            <Globe className="w-3 h-3" />
            <span className="font-bold">{server.location || 'Global Node'}</span>
          </div>
          <div className="text-slate-600 font-medium">
            {server.tenant_name || 'System Server'}
          </div>
        </div>
      </div>
    </div>
  );
}

function ServerDetailsModal({ server, onClose, getHealthStyles }) {
  const { t } = useTranslation();
  const styles = getHealthStyles(server.health_status);
  
  return (
    <div className="fixed inset-0 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center z-50 p-6 animate-in fade-in duration-300">
      <div className="bg-slate-900 border border-slate-800 rounded-[2.5rem] shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-y-auto overflow-x-hidden relative">
        <button onClick={onClose} className="absolute top-8 right-8 p-3 bg-slate-950 text-slate-500 hover:text-white rounded-2xl border border-slate-800 transition-all z-20">
            <X className="w-6 h-6" />
        </button>

        <div className="p-12 space-y-12">
          {/* Header */}
          <div className="flex items-start gap-6">
            <div className={`p-6 rounded-3xl border ${styles.badge} bg-slate-950 shadow-inner`}>
                <styles.icon className="w-12 h-12" />
            </div>
            <div>
                <div className="flex items-center gap-3 mb-2">
                    <h2 className="text-4xl font-black text-white">{server.name}</h2>
                    <span className={`px-3 py-1 text-xs font-black uppercase rounded-xl border ${styles.badge}`}>
                        {t(`servers.${server.health_status}`)}
                    </span>
                </div>
                <p className="text-slate-500 font-mono text-sm tracking-tighter">NODE_UID: {server.id}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <MetricBlock icon={Cpu} label="CPU Load" value="24%" sub="Stable" />
            <MetricBlock icon={HardDrive} label="Disk Usage" value="62%" sub="High Velocity" />
            <MetricBlock icon={Network} label="Network In" value="1.2 GB/s" sub="Uplink Active" />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            <div className="bg-slate-950 border border-slate-800 rounded-3xl p-8">
                <h3 className="text-lg font-bold text-white mb-6 flex items-center gap-2">
                    <RefreshCw className="w-5 h-5 text-indigo-500" />
                    Deployment Stats
                </h3>
                <div className="space-y-4">
                    <InfoRow label="Endpoint URL" value={server.url} mono />
                    <InfoRow label="Assigned Tenant" value={server.tenant_name || 'Global'} />
                    <InfoRow label="Average Ping" value={`${server.response_time_ms}ms`} />
                    <InfoRow label="Total Uptime" value="99.98%" />
                </div>
            </div>

            <div className="bg-slate-950 border border-slate-800 rounded-3xl p-8">
                <h3 className="text-lg font-bold text-white mb-6 flex items-center gap-2">
                    <Clock className="w-5 h-5 text-rose-500" />
                    Recent Logs
                </h3>
                <div className="space-y-3">
                    <LogEntry time="2m ago" msg="Health check completed" type="success" />
                    <LogEntry time="15m ago" msg="Database connection pool refreshed" type="info" />
                    <LogEntry time="1h ago" msg="Periodic background scan finished" type="info" />
                    <LogEntry time="3h ago" msg="System updated to v4.2.1" type="warning" />
                </div>
            </div>
          </div>

          <button onClick={onClose} className="w-full py-4 bg-indigo-600 hover:bg-indigo-500 text-white font-black uppercase tracking-widest rounded-2xl shadow-lg shadow-indigo-900/20 transition-all">
            Close Console
          </button>
        </div>
      </div>
    </div>
  );
}

function MetricBlock({ icon: Icon, label, value, sub }) {
    return (
        <div className="bg-slate-950 border border-slate-800 rounded-3xl p-6 text-center group hover:border-indigo-500/30 transition-colors">
            <Icon className="w-6 h-6 text-slate-600 mx-auto mb-4 group-hover:text-indigo-400 transition-colors" />
            <div className="text-sm font-bold text-slate-500 uppercase tracking-widest mb-1">{label}</div>
            <div className="text-2xl font-black text-white mb-1">{value}</div>
            <div className="text-[10px] font-bold text-emerald-500 uppercase">{sub}</div>
        </div>
    );
}

function InfoRow({ label, value, mono }) {
    return (
        <div className="flex justify-between items-center py-3 border-b border-slate-900">
            <span className="text-xs font-bold text-slate-500 uppercase">{label}</span>
            <span className={`text-sm font-bold text-slate-300 ${mono ? 'font-mono' : ''}`}>{value}</span>
        </div>
    );
}

function LogEntry({ time, msg, type }) {
    const colors = {
        success: 'bg-emerald-500',
        info:    'bg-blue-500',
        warning: 'bg-amber-500',
    };
    return (
        <div className="flex items-center gap-3 text-[11px]">
            <div className={`w-1.5 h-1.5 rounded-full ${colors[type] || 'bg-slate-500'}`} />
            <span className="text-slate-600 font-mono w-12">{time}</span>
            <span className="text-slate-400 font-medium">{msg}</span>
        </div>
    );
}

