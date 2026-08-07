import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { usePackages, useSubscription, useUsage, useSubscribe, useUpdateAutoRenew } from '../hooks/usePackages';
import { Package, CheckCircle2, RefreshCw, CreditCard, Zap, Award } from 'lucide-react';

const BILLING_CYCLE_LABELS = {
  monthly: 'monthly',
  weekly: 'weekly',
  yearly: 'yearly'
};

const Packages = () => {
  const { t } = useTranslation();
  const { packages, loading: packagesLoading } = usePackages();
  const { subscription, loading: subLoading, refetch: refetchSubscription } = useSubscription();
  const { usage, refetch: refetchUsage } = useUsage();
  const { subscribe, loading: subscribing, error: subscribeError } = useSubscribe();
  const { updateAutoRenew, loading: updatingRenew } = useUpdateAutoRenew();
  const [actionMessage, setActionMessage] = useState(null);

  const currentPackage = packages.find(
    (pkg) => subscription && Number(pkg.id) === Number(subscription.package_id)
  );

  const includedMessages = usage?.included_messages;
  const messagesUsed = usage?.messages_used ?? 0;
  const isUnlimited = includedMessages == null;
  const remaining = isUnlimited ? null : (usage?.messages_remaining ?? 0);
  const usagePercent = includedMessages ? Math.min(100, Math.round((messagesUsed / includedMessages) * 100)) : 0;

  const refreshAll = () => {
    refetchSubscription();
    refetchUsage();
  };

  const handleSubscribe = async (pkg) => {
    setActionMessage(null);
    const confirmed = window.confirm(
      `"${pkg.name}" ${formatNumber(pkg.price_usd)} ${t('common.currency')}?`
    );
    if (!confirmed) return;

    const result = await subscribe({ package_id: Number(pkg.id), auto_renew: true });
    if (result) {
      setActionMessage(result.message || t('packages.subscribed'));
      refreshAll();
    }
  };

  const handleToggleAutoRenew = async () => {
    if (!subscription) return;
    setActionMessage(null);
    const ok = await updateAutoRenew(String(subscription.id), !subscription.auto_renew);
    if (ok) {
      setActionMessage(subscription.auto_renew ? t('packages.auto_renew_disabled') : t('packages.auto_renew_enabled'));
      refetchSubscription();
    }
  };

  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex justify-between items-end">
        <div>
          <h1 className="text-3xl font-black text-white tracking-tight">{t('packages.title')}</h1>
          <p className="text-slate-500 font-bold uppercase tracking-widest text-xs mt-2">{t('packages.subtitle')}</p>
        </div>
      </div>

      {(actionMessage || subscribeError) && (
        <div className="glass-card flex items-center justify-between px-6 py-4">
          <span className={`text-sm font-bold ${subscribeError ? 'text-rose-500' : 'text-emerald-500'}`}>
            <CheckCircle2 className="inline w-4 h-4 mr-2" />
            {subscribeError || actionMessage}
          </span>
          <button onClick={() => setActionMessage(null)} className="text-slate-500 hover:text-white transition-all">
            ✕
          </button>
        </div>
      )}

      {subLoading ? (
        <div className="glass-card py-20 text-center text-slate-500 font-bold italic">{t('common.loading')}</div>
      ) : subscription && (
        <div className="glass-card p-8">
          <div className="flex items-center justify-between mb-6">
            <div>
              <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mb-1">{t('packages.current')}</p>
              <h3 className="text-2xl font-black text-white">
                {currentPackage?.name || usage?.package_name || '—'}
              </h3>
            </div>
            <span className={`px-4 py-2 rounded-full text-[10px] font-black uppercase tracking-widest ${
              subscription.status === 'active'
                ? 'bg-emerald-500/10 text-emerald-500'
                : 'bg-amber-500/10 text-amber-500'
            }`}>
              {subscription.status === 'active' ? t('customers.active') : subscription.status}
            </span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-6 rounded-2xl bg-slate-950 border border-slate-800">
              <CreditCard className="w-5 h-5 text-rose-500 mb-3" />
              <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mb-1">{t('packages.package_messages')}</p>
              <p className="text-2xl font-black text-white">
                {isUnlimited ? t('packages.unlimited') : `${formatNumber(includedMessages)} / ${t('packages.messages_month').replace(' / month', 'month')}`}
              </p>
            </div>
            <div className="p-6 rounded-2xl bg-slate-950 border border-slate-800">
              <Zap className="w-5 h-5 text-rose-500 mb-3" />
              <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mb-1">{t('packages.messages_remaining')}</p>
              <p className="text-2xl font-black text-white">
                {isUnlimited ? t('packages.unlimited') : formatNumber(remaining)}
              </p>
            </div>
            <div className="p-6 rounded-2xl bg-slate-950 border border-slate-800">
              <RefreshCw className="w-5 h-5 text-rose-500 mb-3" />
              <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mb-1">{t('packages.auto_renewal')}</p>
              <button
                onClick={handleToggleAutoRenew}
                disabled={updatingRenew}
                className={`mt-1 px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest transition-all ${
                  subscription.auto_renew
                    ? 'bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20'
                    : 'bg-slate-800 text-slate-400 hover:text-white'
                }`}
              >
                {updatingRenew ? t('packages.updating') : subscription.auto_renew ? t('packages.enabled') : t('packages.disabled')}
              </button>
            </div>
          </div>

          {!isUnlimited && (
            <div className="mt-6">
              <div className="w-full bg-slate-800 rounded-full h-3 overflow-hidden">
                <div
                  className={`h-3 rounded-full transition-all ${usagePercent > 90 ? 'bg-rose-500' : 'bg-emerald-500'}`}
                  style={{ width: `${usagePercent}%` }}
                />
              </div>
              <div className="flex justify-between text-sm text-slate-500 mt-3">
                <span>{t('packages.used')}: {formatNumber(messagesUsed)}</span>
                <span>{t('packages.remaining')}: {formatNumber(remaining)}</span>
                <span>{t('packages.included')}: {formatNumber(includedMessages)}</span>
              </div>
              {usage?.overage_messages > 0 && (
                <div className="mt-3 text-sm text-amber-400">
                  {t('packages.overage')}: {formatNumber(usage.overage_messages)} {t('common.currency')} {formatNumber(usage.overage_cost)}
                </div>
              )}
            </div>
          )}

          {subscription.end_date && (
            <p className="mt-6 text-sm text-slate-500">
              {t('packages.ends')}: <span className="text-slate-300 font-bold">{new Date(subscription.end_date).toLocaleDateString()}</span>
            </p>
          )}
        </div>
      )}

      <div>
        <h2 className="text-xl font-black text-white tracking-tight mb-4">{t('packages.available')}</h2>
        {packagesLoading ? (
          <div className="glass-card py-20 text-center text-slate-500 font-bold italic">{t('common.loading')}</div>
        ) : packages.length === 0 ? (
          <div className="glass-card py-20 text-center text-slate-500 font-bold italic">{t('packages.empty')}</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {packages.map((pkg) => {
              const isCurrent = currentPackage && Number(currentPackage.id) === Number(pkg.id);
              const isUpgrade = !!subscription && !isCurrent;
              return (
                <div
                  key={pkg.id}
                  className={`glass-card p-8 flex flex-col group hover:border-slate-700 transition-all ${
                    isCurrent ? 'ring-1 ring-emerald-500/50' : ''
                  }`}
                >
                  {isCurrent && (
                    <span className="inline-flex self-start items-center px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-500 text-[10px] font-black uppercase tracking-widest mb-4">
                      {t('packages.current_plan')}
                    </span>
                  )}
                  <div className="flex items-center justify-between mb-1">
                    <h3 className="text-xl font-black text-white">{pkg.name}</h3>
                    <span className="text-rose-500 font-black">
                      {formatNumber(pkg.price_usd)} <span className="text-xs">{t('common.currency')}</span>
                    </span>
                  </div>
                  <p className="text-[10px] font-black text-slate-600 uppercase tracking-widest mb-4">
                    {t(`packages.${BILLING_CYCLE_LABELS[pkg.billing_cycle] || 'monthly'}`)}
                  </p>
                  <ul className="space-y-3 flex-1">
                    <li className="text-sm text-slate-300 flex items-center gap-3">
                      <Package className="w-4 h-4 text-emerald-500 shrink-0" />
                      {pkg.is_unlimited ? t('packages.unlimited') : `${formatNumber(pkg.included_messages)} ${t('packages.messages_month')}`}
                    </li>
                    {pkg.rollover_enabled && (
                      <li className="text-sm text-slate-300 flex items-center gap-3">
                        <CheckCircle2 className="w-4 h-4 text-emerald-500 shrink-0" />
                        {t('packages.rollover')}
                      </li>
                    )}
                    {pkg.is_unlimited && (
                      <li className="text-sm text-slate-300 flex items-center gap-3">
                        <Award className="w-4 h-4 text-emerald-500 shrink-0" />
                        {t('packages.no_overage')}
                      </li>
                    )}
                  </ul>
                  <button
                    onClick={() => handleSubscribe(pkg)}
                    disabled={subscribing || isCurrent}
                    className={`mt-6 w-full py-3 rounded-2xl font-black uppercase tracking-widest text-xs transition-all active:scale-95 ${
                      isCurrent
                        ? 'bg-slate-800 text-slate-600 cursor-not-allowed'
                        : 'bg-rose-600 hover:bg-rose-500 text-white shadow-xl shadow-rose-900/20'
                    }`}
                  >
                    {subscribing ? t('packages.processing') : isCurrent ? t('packages.current_plan') : isUpgrade ? t('packages.upgrade') : t('packages.subscribe_now')}
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default Packages;
