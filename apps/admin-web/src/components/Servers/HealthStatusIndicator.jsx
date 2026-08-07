import React from 'react';
import { Activity, AlertTriangle, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

/**
 * HealthStatusIndicator Component
 * 
 * Displays server health status with color-coded indicators
 * - Green: Healthy (response_time < 200ms AND error_rate < 1%)
 * - Yellow: Degraded (response_time 200-500ms OR error_rate 1-5%)
 * - Red: Unhealthy (response_time > 500ms OR error_rate > 5%)
 */
const HealthStatusIndicator = ({ status, size = 'md', showLabel = true, showIcon = true }) => {
  const { t } = useTranslation();

  const getStatusConfig = () => {
    switch (status?.toLowerCase()) {
      case 'healthy':
        return {
          color: 'text-emerald-500',
          bgColor: 'bg-emerald-500/10',
          borderColor: 'border-emerald-500/20',
          icon: Activity,
          label: t('servers.healthy'),
          dotColor: 'bg-emerald-500'
        };
      case 'degraded':
        return {
          color: 'text-amber-500',
          bgColor: 'bg-amber-500/10',
          borderColor: 'border-amber-500/20',
          icon: AlertTriangle,
          label: t('servers.degraded'),
          dotColor: 'bg-amber-500'
        };
      case 'unhealthy':
        return {
          color: 'text-rose-500',
          bgColor: 'bg-rose-500/10',
          borderColor: 'border-rose-500/20',
          icon: XCircle,
          label: t('servers.unhealthy'),
          dotColor: 'bg-rose-500'
        };
      default:
        return {
          color: 'text-slate-500',
          bgColor: 'bg-slate-500/10',
          borderColor: 'border-slate-500/20',
          icon: Activity,
          label: t('companies.unknown'),
          dotColor: 'bg-slate-500'
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;

  const sizeClasses = {
    sm: {
      container: 'px-2 py-1 text-xs',
      icon: 'w-3 h-3',
      dot: 'w-2 h-2'
    },
    md: {
      container: 'px-3 py-1.5 text-sm',
      icon: 'w-4 h-4',
      dot: 'w-2.5 h-2.5'
    },
    lg: {
      container: 'px-4 py-2 text-base',
      icon: 'w-5 h-5',
      dot: 'w-3 h-3'
    }
  };

  const currentSize = sizeClasses[size] || sizeClasses.md;

  return (
    <div
      className={`inline-flex items-center gap-2 rounded-lg border ${config.bgColor} ${config.borderColor} ${currentSize.container}`}
    >
      {showIcon && (
        <Icon className={`${config.color} ${currentSize.icon}`} />
      )}
      {showLabel && (
        <span className={`font-medium ${config.color}`}>
          {config.label}
        </span>
      )}
      {!showIcon && !showLabel && (
        <div className={`${currentSize.dot} ${config.dotColor} rounded-full animate-pulse`} />
      )}
    </div>
  );
};

export default HealthStatusIndicator;
