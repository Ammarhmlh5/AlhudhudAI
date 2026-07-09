import { Globe, LayoutGrid, Zap, Users, TrendingUp, Clock, Shield } from 'lucide-react';

const stats = [
    { label: 'الخطط النشطة', value: '—', icon: LayoutGrid, color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
    { label: 'إجمالي المشتركين', value: '—', icon: Users, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20' },
    { label: 'العمليات اليومية', value: '—', icon: Zap, color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
    { label: 'معدل النمو', value: '—', icon: TrendingUp, color: 'text-rose-400', bg: 'bg-rose-500/10 border-rose-500/20' },
];

const planTypes = [
    { name: 'خطة أساسية', ops: '5,000 عملية / شهر', feature: 'بوابة مشتركة', badge: 'مجانية', badgeColor: 'bg-slate-700 text-slate-300', icon: '🔹' },
    { name: 'خطة احترافية', ops: '25,000 عملية / شهر', feature: 'بوابة مخصصة', badge: 'شائعة', badgeColor: 'bg-sky-600/30 text-sky-300', icon: '🔷' },
    { name: 'خطة متقدمة', ops: '100,000 عملية / شهر', feature: 'بوابات متعددة', badge: 'مميزة', badgeColor: 'bg-amber-600/30 text-amber-300', icon: '💎' },
    { name: 'خطة مؤسسية', ops: 'غير محدودة', feature: 'عزل كامل', badge: 'بريميوم', badgeColor: 'bg-rose-600/30 text-rose-300', icon: '🏆' },
];

export default function MyGatewayPlans() {
    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-3 duration-500" dir="rtl">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-3 rounded-2xl bg-sky-500/10 border border-sky-500/20">
                        <Globe className="w-6 h-6 text-sky-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white">خطط بوابتي</h3>
                        <p className="text-slate-400 text-sm mt-0.5">إدارة خطط الاشتراك لبوابات الرسائل الخاصة بالعملاء</p>
                    </div>
                </div>
                <div className="flex items-center gap-2 px-4 py-2 bg-amber-500/10 border border-amber-500/20 rounded-xl">
                    <Clock className="w-4 h-4 text-amber-400" />
                    <span className="text-amber-400 text-sm font-bold">قيد التطوير</span>
                </div>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {stats.map((stat) => (
                    <div key={stat.label} className={`p-4 rounded-2xl border ${stat.bg} flex flex-col gap-2`}>
                        <stat.icon className={`w-5 h-5 ${stat.color}`} />
                        <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                        <p className="text-slate-400 text-xs leading-relaxed">{stat.label}</p>
                    </div>
                ))}
            </div>

            {/* Plans Table */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
                    <h4 className="text-white font-bold">هيكل الخطط المخطط</h4>
                    <span className="text-xs text-slate-500 bg-slate-800 px-3 py-1 rounded-full">معاينة / غير مفعّل</span>
                </div>
                <div className="divide-y divide-slate-800/50">
                    {planTypes.map((plan) => (
                        <div key={plan.name} className="flex items-center justify-between px-6 py-4 hover:bg-slate-800/30 transition-colors group">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-slate-800 group-hover:bg-slate-700 transition-colors text-lg">
                                    {plan.icon}
                                </div>
                                <div>
                                    <div className="flex items-center gap-2">
                                        <p className="text-white font-bold text-sm">{plan.name}</p>
                                        <span className={`text-xs px-2 py-0.5 rounded-full font-bold ${plan.badgeColor}`}>{plan.badge}</span>
                                    </div>
                                    <p className="text-slate-500 text-xs mt-0.5">{plan.feature}</p>
                                </div>
                            </div>
                            <div className="text-right">
                                <p className="text-slate-400 text-xs">العمليات</p>
                                <p className="text-sky-400 text-sm font-bold">{plan.ops}</p>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Info Banner */}
            <div className="flex items-start gap-4 p-5 rounded-2xl bg-sky-500/5 border border-sky-500/15 border-dashed">
                <div className="p-3 rounded-xl bg-sky-500/10 shrink-0 mt-0.5">
                    <Shield className="w-5 h-5 text-sky-400" />
                </div>
                <div>
                    <p className="text-sky-300 font-bold text-sm">هذا القسم قيد الإعداد</p>
                    <p className="text-slate-500 text-xs mt-1">
                        سيتمكن المشرف من تحديد خطط بوابتي المتعددة وربطها بالشركات المشتركة، مع إمكانية تحديد حدود العمليات الشهرية، نوع البوابة (مشتركة أو مخصصة)، وسياسات الفوترة التلقائية.
                    </p>
                </div>
            </div>
        </div>
    );
}
