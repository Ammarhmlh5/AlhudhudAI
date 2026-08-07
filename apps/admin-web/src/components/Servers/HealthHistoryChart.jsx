import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * HealthHistoryChart Component
 * 
 * Displays server health history as a simple line chart
 * Shows response time trends over the last 30 days
 */
const HealthHistoryChart = ({ history = [] }) => {
  const { t } = useTranslation();

  const chartData = useMemo(() => {
    if (!history || history.length === 0) return null;

    // Calculate min and max for scaling
    const responseTimes = history.map(h => h.response_time_ms || 0);
    const maxResponseTime = Math.max(...responseTimes, 500);
    const minResponseTime = Math.min(...responseTimes, 0);

    return {
      points: history.map((item, index) => {
        const x = (index / (history.length - 1)) * 100;
        const normalizedValue = ((item.response_time_ms - minResponseTime) / (maxResponseTime - minResponseTime)) * 100;
        const y = 100 - normalizedValue; // Invert Y axis
        return { x, y, value: item.response_time_ms, status: item.health_status };
      }),
      maxResponseTime,
      minResponseTime
    };
  }, [history]);

  if (!history || history.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 bg-slate-900 rounded-xl border border-slate-800">
        <p className="text-slate-500">{t('servers.no_history')}</p>
      </div>
    );
  }

  if (!chartData) return null;

  // Create SVG path
  const pathData = chartData.points
    .map((point, index) => {
      const command = index === 0 ? 'M' : 'L';
      return `${command} ${point.x} ${point.y}`;
    })
    .join(' ');

  // Create area path (for gradient fill)
  const areaPathData = `${pathData} L 100 100 L 0 100 Z`;

  // Get status color for the line
  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'healthy':
        return '#10b981'; // emerald-500
      case 'degraded':
        return '#f59e0b'; // amber-500
      case 'unhealthy':
        return '#f43f5e'; // rose-500
      default:
        return '#64748b'; // slate-500
    }
  };

  // Use the most recent status for line color
  const lineColor = getStatusColor(chartData.points[chartData.points.length - 1]?.status);

  return (
    <div className="space-y-4">
      {/* Chart */}
      <div className="relative h-48 bg-slate-900 rounded-xl border border-slate-800 p-4">
        <svg
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          className="w-full h-full"
        >
          {/* Grid lines */}
          <line x1="0" y1="25" x2="100" y2="25" stroke="#334155" strokeWidth="0.2" strokeDasharray="2,2" />
          <line x1="0" y1="50" x2="100" y2="50" stroke="#334155" strokeWidth="0.2" strokeDasharray="2,2" />
          <line x1="0" y1="75" x2="100" y2="75" stroke="#334155" strokeWidth="0.2" strokeDasharray="2,2" />

          {/* Area gradient */}
          <defs>
            <linearGradient id="areaGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stopColor={lineColor} stopOpacity="0.3" />
              <stop offset="100%" stopColor={lineColor} stopOpacity="0.05" />
            </linearGradient>
          </defs>

          {/* Area fill */}
          <path
            d={areaPathData}
            fill="url(#areaGradient)"
          />

          {/* Line */}
          <path
            d={pathData}
            fill="none"
            stroke={lineColor}
            strokeWidth="1"
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Data points */}
          {chartData.points.map((point, index) => (
            <circle
              key={index}
              cx={point.x}
              cy={point.y}
              r="1"
              fill={getStatusColor(point.status)}
              className="opacity-75"
            />
          ))}
        </svg>

        {/* Y-axis labels */}
        <div className="absolute left-0 top-0 bottom-0 flex flex-col justify-between text-xs text-slate-500 pr-2">
          <span>{chartData.maxResponseTime}ms</span>
          <span>{Math.round((chartData.maxResponseTime + chartData.minResponseTime) / 2)}ms</span>
          <span>{chartData.minResponseTime}ms</span>
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center justify-center gap-6 text-xs">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-emerald-500" />
          <span className="text-slate-400">{t('servers.healthy')}</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-amber-500" />
          <span className="text-slate-400">{t('servers.degraded')}</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-rose-500" />
          <span className="text-slate-400">{t('servers.unhealthy')}</span>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-3">
          <p className="text-xs text-slate-500">{t('servers.response_time')}</p>
          <p className="text-lg font-bold text-white">
            {chartData.points[chartData.points.length - 1]?.value || 0}ms
          </p>
        </div>
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-3">
          <p className="text-xs text-slate-500">{t('serversList.avg')}</p>
          <p className="text-lg font-bold text-white">
            {Math.round(chartData.points.reduce((sum, p) => sum + p.value, 0) / chartData.points.length)}ms
          </p>
        </div>
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-3">
          <p className="text-xs text-slate-500">{t('serversList.max')}</p>
          <p className="text-lg font-bold text-white">
            {chartData.maxResponseTime}ms
          </p>
        </div>
      </div>
    </div>
  );
};

export default HealthHistoryChart;
