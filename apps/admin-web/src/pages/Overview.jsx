import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  Building2, MessageSquare, CheckCircle2, Server, 
  TrendingUp, TrendingDown, AlertCircle, BarChart3,
  Globe, ShieldCheck, Zap
} from 'lucide-react';
import apiClient from '../api/axios';

export default function Overview() {
  const { t } = useTranslation();
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get('/api/v1/admin/stats');
        setStats(response.data);
        setError(null);
      } catch (err) {
        setError(err.response?.data?.error || 'Failed to load dashboard statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="relative">
          <div className="w-12 h-12 border-4 border-rose-500/20 border-t-rose-500 rounded-full animate-spin"></div>
          <div className="mt-4 text-slate-400 font-medium animate-pulse">{t('loading')}...</div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[60vh] p-6">
        <div className="bg-rose-500/10 border border-rose-500/20 rounded-2xl p-8 max-w-md text-center">
          <AlertCircle className="w-12 h-12 text-rose-500 mx-auto mb-4" />
          <h2 className="text-xl font-bold text-white mb-2">Error Loading Dashboard</h2>
          <p className="text-slate-400 mb-6">{error}</p>
          <button onClick={() => window.location.reload()} className="px-6 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl transition-all font-bold">
            Retry Now
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8 space-y-8 animate-in fade-in duration-700">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-4xl font-black text-white tracking-tight">
            {t('dashboard.overview')}
          </h1>
          <p className="mt-2 text-slate-400 font-medium">
            {t('dashboard.overview_description')}
          </p>
        </div>
        <div className="flex items-center gap-3 bg-slate-900 border border-slate-800 p-2 rounded-2xl">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/10 rounded-xl">
            <div className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse" />
            <span className="text-xs font-bold text-emerald-400 uppercase tracking-wider">Live System</span>
          </div>
          <div className="text-xs text-slate-500 font-mono px-2">
            Last Update: {new Date().toLocaleTimeString()}
          </div>
        </div>
      </div>

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title={t('dashboard.total_companies')}
          value={stats?.total_companies || 0}
          icon={Building2}
          color="rose"
          trend={stats?.companies_trend}
        />
        <StatCard
          title={t('dashboard.messages_today')}
          value={stats?.messages_today || 0}
          icon={MessageSquare}
          color="blue"
          trend={stats?.messages_trend}
        />
        <StatCard
          title={t('dashboard.success_rate')}
          value={`${(stats?.success_rate || 0).toFixed(1)}%`}
          icon={CheckCircle2}
          color="emerald"
          trend={stats?.success_rate_trend}
        />
        <StatCard
          title={t('dashboard.active_servers')}
          value={stats?.active_servers || 0}
          icon={Server}
          color="violet"
          trend={stats?.servers_trend}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Rating Distribution Chart */}
        <div className="lg:col-span-1 bg-slate-900 border border-slate-800 rounded-3xl p-8 shadow-2xl relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
            <BarChart3 className="w-24 h-24 text-white" />
          </div>
          <h2 className="text-xl font-bold text-white mb-8 flex items-center gap-3">
            <Zap className="w-5 h-5 text-amber-400" />
            {t('dashboard.rating_distribution')}
          </h2>
          <RatingDistributionChart data={stats?.rating_distribution} />
        </div>

        {/* Server Health Summary */}
        <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-3xl p-8 shadow-2xl relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
            <ShieldCheck className="w-24 h-24 text-white" />
          </div>
          <div className="flex items-center justify-between mb-8">
            <h2 className="text-xl font-bold text-white flex items-center gap-3">
              <Globe className="w-5 h-5 text-sky-400" />
              {t('dashboard.server_health')}
            </h2>
            <div className="px-4 py-1.5 bg-slate-950 rounded-xl border border-slate-800 text-xs text-slate-400 font-bold">
              {stats?.servers?.length || 0} Total Nodes
            </div>
          </div>
          <ServerHealthSummary servers={stats?.servers} />
        </div>
      </div>

      {/* Active Alerts Panel */}
      <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 shadow-2xl">
        <h2 className="text-xl font-bold text-white mb-6 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 text-rose-500" />
          {t('dashboard.active_alerts')}
        </h2>
        <ActiveAlertsPanel alerts={stats?.active_alerts} />
      </div>
    </div>
  );
}

function StatCard({ title, value, icon: Icon, color, trend }) {
  const colors = {
    rose:    'from-rose-500/20 to-rose-500/5 text-rose-500 border-rose-500/20',
    blue:    'from-blue-500/20 to-blue-500/5 text-blue-500 border-blue-500/20',
    emerald: 'from-emerald-500/20 to-emerald-500/5 text-emerald-500 border-emerald-500/20',
    violet:  'from-violet-500/20 to-violet-500/5 text-violet-500 border-violet-500/20',
  };

  return (
    <div className={`bg-gradient-to-br ${colors[color]} border rounded-3xl p-8 transition-all hover:scale-[1.02] cursor-default group`}>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-bold text-slate-400 uppercase tracking-widest mb-1">
            {title}
          </p>
          <p className="text-4xl font-black text-white tracking-tight">
            {typeof value === 'number' ? value.toLocaleString() : value}
          </p>
          {trend !== undefined && (
            <div className={`mt-4 flex items-center gap-1.5 text-sm font-bold ${trend >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
              {trend >= 0 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
              {Math.abs(trend).toFixed(1)}%
              <span className="text-slate-500 font-medium ml-1">vs last week</span>
            </div>
          )}
        </div>
        <div className={`p-4 bg-slate-950/50 rounded-2xl border border-white/5 group-hover:scale-110 transition-transform`}>
          <Icon className="w-8 h-8" />
        </div>
      </div>
    </div>
  );
}

function RatingDistributionChart({ data }) {
  if (!data) return <div className="text-slate-500">No data available</div>;

  const categories = [
    { name: 'Excellent', value: data.excellent || 0, color: 'bg-emerald-500', shadow: 'shadow-emerald-500/20' },
    { name: 'Good',      value: data.good || 0,      color: 'bg-blue-500',    shadow: 'shadow-blue-500/20'    },
    { name: 'Average',   value: data.average || 0,   color: 'bg-amber-500',   shadow: 'shadow-amber-500/20'   },
    { name: 'Poor',      value: data.poor || 0,      color: 'bg-rose-500',    shadow: 'shadow-rose-500/20'    },
  ];

  const total = categories.reduce((sum, cat) => sum + cat.value, 0);

  return (
    <div className="space-y-6">
      {categories.map((category) => {
        const percentage = total > 0 ? (category.value / total) * 100 : 0;
        return (
          <div key={category.name} className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="font-bold text-slate-300">{category.name}</span>
              <span className="text-slate-500 font-mono">
                {category.value} <span className="text-[10px] opacity-50">({percentage.toFixed(0)}%)</span>
              </span>
            </div>
            <div className="h-3 bg-slate-950 rounded-full overflow-hidden p-0.5 border border-slate-800">
              <div
                className={`${category.color} ${category.shadow} h-full rounded-full transition-all duration-1000 shadow-lg`}
                style={{ width: `${percentage}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ActiveAlertsPanel({ alerts }) {
  if (!alerts || alerts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-600 bg-slate-950/30 rounded-2xl border border-dashed border-slate-800">
        <ShieldCheck className="w-12 h-12 mb-3 opacity-20" />
        <p className="font-medium">System is running smooth. No active alerts.</p>
      </div>
    );
  }

  const getSeverityStyles = (severity) => {
    switch (severity?.toLowerCase()) {
      case 'critical': return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
      case 'warning':  return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
      default:         return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
    }
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {alerts.slice(0, 4).map((alert) => (
        <div key={alert.id} className={`p-5 rounded-2xl border ${getSeverityStyles(alert.severity)} bg-slate-950/50 flex items-start gap-4 hover:border-white/10 transition-colors`}>
          <div className="p-3 bg-slate-900 rounded-xl">
            <AlertCircle className="w-5 h-5" />
          </div>
          <div className="flex-1">
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs font-black uppercase tracking-tighter">{alert.severity}</span>
              <span className="text-[10px] font-mono opacity-50">{new Date(alert.created_at).toLocaleTimeString()}</span>
            </div>
            <h3 className="text-sm font-bold text-white mb-1">{alert.title}</h3>
            <p className="text-xs text-slate-500 line-clamp-1">{alert.message}</p>
          </div>
        </div>
      ))}
    </div>
  );
}

function ServerHealthSummary({ servers }) {
  if (!servers || servers.length === 0) return <div className="text-slate-500">No servers available</div>;

  const healthCounts = servers.reduce((acc, s) => {
    acc[s.health_status] = (acc[s.health_status] || 0) + 1;
    return acc;
  }, {});

  return (
    <div className="space-y-8">
      <div className="grid grid-cols-3 gap-6">
        <div className="bg-slate-950 p-6 rounded-2xl border border-slate-800 text-center">
          <div className="text-3xl font-black text-emerald-400 mb-1">{healthCounts.healthy || 0}</div>
          <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Healthy</div>
        </div>
        <div className="bg-slate-950 p-6 rounded-2xl border border-slate-800 text-center">
          <div className="text-3xl font-black text-amber-400 mb-1">{healthCounts.degraded || 0}</div>
          <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Degraded</div>
        </div>
        <div className="bg-slate-950 p-6 rounded-2xl border border-slate-800 text-center">
          <div className="text-3xl font-black text-rose-500 mb-1">{healthCounts.unhealthy || 0}</div>
          <div className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Down</div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {servers.slice(0, 6).map((server) => (
          <div key={server.id} className="flex items-center justify-between p-4 bg-slate-950/50 border border-slate-800 rounded-2xl hover:bg-slate-800/50 transition-colors group">
            <div className="flex items-center gap-3">
              <div className={`w-2 h-2 rounded-full shadow-[0_0_8px_rgba(0,0,0,0.3)] ${
                server.health_status === 'healthy' ? 'bg-emerald-500 shadow-emerald-500/50' : 
                server.health_status === 'degraded' ? 'bg-amber-500 shadow-amber-500/50' : 'bg-rose-500 shadow-rose-500/50'
              }`} />
              <span className="text-sm font-bold text-slate-300 group-hover:text-white transition-colors">{server.name}</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-[10px] font-mono text-slate-500 uppercase tracking-tighter">{server.location || 'Global'}</span>
              <span className="text-xs font-bold text-slate-400">{server.response_time_ms}ms</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

