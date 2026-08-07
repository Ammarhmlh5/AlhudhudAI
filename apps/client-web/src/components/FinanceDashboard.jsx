import { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { usePackages, useSubscription, useUsage, useSubscribe, useUpdateAutoRenew } from '../hooks/usePackages';
import { CreditCard, Wallet, Zap, TrendingUp, Award, CheckCircle2, RefreshCw } from 'lucide-react';

const BILLING_CYCLE_LABELS = {
  monthly: 'شهري',
  weekly: 'أسبوعي',
  yearly: 'سنوي'
};

const STATUS_LABELS = {
  active: 'نشط',
  grace_period: 'فترة سماح',
  suspended: 'موقوف',
  expired: 'منتهي',
  cancelled: 'ملغي'
};

const formatNumber = (value) => {
  const num = Number(value);
  return Number.isFinite(num) ? num.toLocaleString('en-US') : '0';
};

export default function FinanceDashboard() {
  const { tenant } = useAuth();
  const { packages, loading: packagesLoading } = usePackages();
  const { subscription, loading: subLoading, refetch: refetchSubscription } = useSubscription();
  const { usage, refetch: refetchUsage } = useUsage();
  const { subscribe, loading: subscribing, error: subscribeError } = useSubscribe();
  const { updateAutoRenew, loading: updatingRenew } = useUpdateAutoRenew();
  const [activeTab, setActiveTab] = useState('balance');
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
    const confirmed = window.confirm(`الاشتراك في باقة "${pkg.name}" بمبلغ ${formatNumber(pkg.price_usd)} TND؟`);
    if (!confirmed) return;

    const result = await subscribe({ package_id: Number(pkg.id), auto_renew: true });
    if (result) {
      setActionMessage(result.message || 'تم الاشتراك بنجاح');
      refreshAll();
    }
  };

  const handleToggleAutoRenew = async () => {
    if (!subscription) return;
    setActionMessage(null);
    const ok = await updateAutoRenew(String(subscription.id), !subscription.auto_renew);
    if (ok) {
      setActionMessage(subscription.auto_renew ? 'تم إيقاف التجديد التلقائي' : 'تم تفعيل التجديد التلقائي');
      refetchSubscription();
    }
  };

  const balanceCards = [
    {
      icon: Wallet,
      title: 'الرصيد النقدي',
      value: `${formatNumber(tenant?.balance)} TND`,
      sub: 'الرصيد المتاح للاشتراكات والرسائل الزائدة'
    },
    {
      icon: Zap,
      title: 'رسائل الباقة',
      value: isUnlimited ? 'غير محدود' : `${formatNumber(remaining)} رسالة`,
      sub: isUnlimited ? 'باقة غير محدودة' : `من أصل ${formatNumber(includedMessages)} رسالة`
    },
    {
      icon: CreditCard,
      title: 'الحالة',
      value: STATUS_LABELS[subscription?.status] || 'لا يوجد اشتراك',
      sub: usage?.package_name ? `الباقة الحالية: ${usage.package_name}` : 'اشترك في إحدى الباقات للبدء'
    }
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">المالية والاشتراكات</h1>
        <p className="text-slate-400 mt-1">إدارة رصيدك، رسائلك، وباقات الاشتراك</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2">
        <button
          onClick={() => setActiveTab('balance')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'balance' ? 'bg-rose-500 text-white' : 'bg-slate-900 text-slate-400 hover:text-white'
          }`}
        >
          الرصيد
        </button>
        <button
          onClick={() => setActiveTab('packages')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'packages' ? 'bg-rose-500 text-white' : 'bg-slate-900 text-slate-400 hover:text-white'
          }`}
        >
          الباقات
        </button>
      </div>

      {/* Action feedback */}
      {(actionMessage || subscribeError) && (
        <div className="flex items-center justify-between bg-slate-900 border border-slate-800 rounded-xl px-4 py-3">
          <span className={`text-sm ${subscribeError ? 'text-rose-400' : 'text-emerald-400'}`}>
            <CheckCircle2 className="inline w-4 h-4 ml-2" />
            {subscribeError || actionMessage}
          </span>
          <button
            onClick={() => { setActionMessage(null); }}
            className="text-slate-500 hover:text-white"
          >
            ✕
          </button>
        </div>
      )}

      {activeTab === 'balance' && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {balanceCards.map((card) => (
              <div key={card.title} className="bg-slate-900 border border-slate-800 rounded-xl p-6">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-slate-800 rounded-lg">
                    <card.icon className="w-5 h-5 text-rose-400" />
                  </div>
                  <h3 className="font-semibold text-slate-300">{card.title}</h3>
                </div>
                <p className="text-2xl font-bold text-white mt-4">{card.value}</p>
                <p className="text-sm text-slate-500 mt-1">{card.sub}</p>
              </div>
            ))}
          </div>

          {/* Usage progress */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold text-white">استخدام رسائل الباقة</h3>
              {subscription && (
                <button
                  onClick={handleToggleAutoRenew}
                  disabled={updatingRenew}
                  className="text-sm text-slate-400 hover:text-white flex items-center gap-2"
                >
                  <RefreshCw className={`w-4 h-4 ${updatingRenew ? 'animate-spin' : ''}`} />
                  التجديد التلقائي: {subscription.auto_renew ? 'مفعّل' : 'موقوف'}
                </button>
              )}
            </div>
            {isUnlimited ? (
              <div className="flex items-center gap-2 text-emerald-400">
                <Award className="w-5 h-5" />
                <span>باقتك غير محدودة — لا يوجد حد لعدد الرسائل</span>
              </div>
            ) : (
              <>
                <div className="w-full bg-slate-800 rounded-full h-3 overflow-hidden">
                  <div
                    className={`h-3 rounded-full transition-all ${usagePercent > 90 ? 'bg-rose-500' : 'bg-emerald-500'}`}
                    style={{ width: `${usagePercent}%` }}
                  />
                </div>
                <div className="flex justify-between text-sm text-slate-400 mt-3">
                  <span>المستخدم: {formatNumber(messagesUsed)} رسالة</span>
                  <span>المتبقي: {formatNumber(remaining)} رسالة</span>
                  <span>إجمالي الباقة: {formatNumber(includedMessages)} رسالة</span>
                </div>
                {usage?.overage_messages > 0 && (
                  <div className="mt-3 text-sm text-amber-400">
                    رسائل زائدة: {formatNumber(usage.overage_messages)} رسالة بتكلفة {formatNumber(usage.overage_cost)} TND
                  </div>
                )}
              </>
            )}
            {subscription?.end_date && (
              <div className="mt-3 text-sm text-slate-400">
                تاريخ انتهاء الباقة: <span className="text-slate-200">{new Date(subscription.end_date).toLocaleDateString()}</span>
              </div>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
              <h3 className="font-semibold text-white mb-3">معلومات الحساب</h3>
              <div className="space-y-2 text-sm">
                <p className="flex justify-between"><span className="text-slate-500">الاسم التجاري</span><span className="text-slate-200">{tenant?.business_name || '-'}</span></p>
                <p className="flex justify-between"><span className="text-slate-500">البريد</span><span className="text-slate-200">{tenant?.email || '-'}</span></p>
                <p className="flex justify-between"><span className="text-slate-500">رقم الهاتف</span><span className="text-slate-200">{tenant?.phone || '-'}</span></p>
              </div>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
              <h3 className="font-semibold text-white mb-3">الرسائل الزائدة</h3>
              <p className="text-2xl font-bold text-amber-400">
                {formatNumber(usage?.overage_messages ?? 0)} رسالة
              </p>
              <p className="text-sm text-slate-500 mt-1">
                {usage?.overage_cost && Number(usage.overage_cost) > 0
                  ? `بتكلفة ${formatNumber(usage.overage_cost)} TND تُخصم من رصيدك`
                  : 'لا توجد رسائل زائدة حالياً'}
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
              <div className="flex items-center gap-2 mb-3">
                <TrendingUp className="w-5 h-5 text-rose-400" />
                <h3 className="font-semibold text-white">ترقية الباقة</h3>
              </div>
              <p className="text-sm text-slate-400 mb-4">
                قم بترقية باقتك الحالية للحصول على المزيد من الرسائل ومزايا أفضل.
              </p>
              <button
                onClick={() => setActiveTab('packages')}
                className="w-full px-4 py-2 bg-rose-500 hover:bg-rose-600 text-white rounded-lg text-sm font-medium transition-colors"
              >
                تصفح الباقات
              </button>
            </div>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
            <h3 className="font-semibold text-white mb-3">تواصل معنا</h3>
            <p className="text-sm text-slate-400">
              للمساعدة في اختيار الباقة المناسبة أو تجديد اشتراكك، يرجى التواصل معنا عبر البريد الإلكتروني{' '}
              <span className="text-slate-200">support@hudhud.app</span> أو الهاتف <span className="text-slate-200">+216 00 000 000</span>.
            </p>
          </div>
        </>
      )}

      {activeTab === 'packages' && (
        <div>
          {(packagesLoading || subLoading) && (
            <div className="text-slate-400 text-center py-12">جاري تحميل الباقات...</div>
          )}

          {!packagesLoading && packages.length === 0 && (
            <div className="text-center py-12 text-slate-500">لا توجد باقات متاحة حالياً</div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {packages.map((pkg) => {
              const isCurrent = currentPackage && Number(currentPackage.id) === Number(pkg.id);
              const isUpgrade = !!subscription && !isCurrent;
              return (
                <div
                  key={pkg.id}
                  className={`bg-slate-900 border rounded-xl p-6 flex flex-col ${
                    isCurrent ? 'border-emerald-500' : 'border-slate-800'
                  }`}
                >
                  {isCurrent && (
                    <span className="inline-flex self-start items-center px-2 py-1 rounded-full bg-emerald-500/10 text-emerald-400 text-xs font-medium mb-3">
                      الباقة الحالية
                    </span>
                  )}
                  <div className="flex items-center justify-between">
                    <h3 className="text-lg font-bold text-white">{pkg.name}</h3>
                    <span className="text-rose-400 font-bold">
                      {formatNumber(pkg.price_usd)} TND
                    </span>
                  </div>
                  <p className="text-sm text-slate-500 mt-1">
                    {BILLING_CYCLE_LABELS[pkg.billing_cycle] || pkg.billing_cycle || 'شهري'}
                  </p>
                  <ul className="mt-4 space-y-2 flex-1">
                    <li className="text-sm text-slate-300 flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4 text-emerald-500 shrink-0" />
                      {pkg.is_unlimited ? 'رسائل غير محدودة' : `${formatNumber(pkg.included_messages)} رسالة شهرياً`}
                    </li>
                    {pkg.rollover_enabled && (
                      <li className="text-sm text-slate-300 flex items-center gap-2">
                        <CheckCircle2 className="w-4 h-4 text-emerald-500 shrink-0" />
                        ترحيل الرسائل غير المستخدمة
                      </li>
                    )}
                  </ul>
                  <button
                    onClick={() => handleSubscribe(pkg)}
                    disabled={subscribing || isCurrent}
                    className={`mt-5 w-full px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isCurrent
                        ? 'bg-slate-800 text-slate-500 cursor-not-allowed'
                        : 'bg-rose-500 hover:bg-rose-600 text-white'
                    }`}
                  >
                    {subscribing ? 'جاري المعالجة...' : isCurrent ? 'الباقة الحالية' : isUpgrade ? 'ترقية إلى هذه الباقة' : 'اشترك الآن'}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
