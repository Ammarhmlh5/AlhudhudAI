import { Router, Network, Zap, Lock, Clock, Shield, Globe } from 'lucide-react';

const stats = [
    { label: 'بوابات الهدهد المُدارة', value: '—', icon: Router, color: 'text-rose-400', bg: 'bg-rose-500/10 border-rose-500/20' },
    { label: 'بوابات نشطة الآن', value: '—', icon: Network, color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
    { label: 'العمليات اليومية', value: '—', icon: Zap, color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
    { label: 'بوابات مؤمّنة', value: '—', icon: Lock, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20' },
];

const gatewayTiers = [
    {
        name: 'بوابة مشتركة - أساسية',
        desc: 'بوابة مشتركة بين عدة عملاء',
        throughput: 'حتى 10,000 عملية / يوم',
        isolation: 'عزل منطقي',
        badge: 'مشتركة',
        badgeColor: 'bg-slate-700 text-slate-300',
        icon: Globe,
        iconColor: 'text-slate-400',
    },
    {
        name: 'بوابة هدهد - مخصصة',
        desc: 'بوابة مخصصة لعميل واحد فقط',
        throughput: 'حتى 100,000 عملية / يوم',
        isolation: 'عزل كامل',
        badge: 'مخصصة',
        badgeColor: 'bg-rose-600/30 text-rose-300',
        icon: Router,
        iconColor: 'text-rose-400',
    },
    {
        name: 'بوابة هدهد - مؤسسية',
        desc: 'بوابات متعددة لشركة واحدة',
        throughput: 'غير محدودة',
        isolation: 'عزل مؤسسي كامل',
        badge: 'بريميوم',
        badgeColor: 'bg-amber-600/30 text-amber-300',
        icon: Network,
        iconColor: 'text-amber-400',
    },
];

export default function HudhudGateways() {
    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-3 duration-500" dir="rtl">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-3 rounded-2xl bg-rose-500/10 border border-rose-500/20">
                        <Router className="w-6 h-6 text-rose-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white">بوابات الهدهد</h3>
                        <p className="text-slate-400 text-sm mt-0.5">إدارة بوابات الرسائل المملوكة لمنصة الهدهد وتعرفاتها</p>
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

            {/* Gateways Tiers Table */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
                    <h4 className="text-white font-bold">فئات بوابات الهدهد</h4>
                    <span className="text-xs text-slate-500 bg-slate-800 px-3 py-1 rounded-full">معاينة / غير مفعّل</span>
                </div>
                <div className="divide-y divide-slate-800/50">
                    {gatewayTiers.map((tier) => (
                        <div key={tier.name} className="flex items-center justify-between px-6 py-5 hover:bg-slate-800/30 transition-colors group">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-slate-800 group-hover:bg-slate-700 transition-colors">
                                    <tier.icon className={`w-4 h-4 ${tier.iconColor}`} />
                                </div>
                                <div>
                                    <div className="flex items-center gap-2">
                                        <p className="text-white font-bold text-sm">{tier.name}</p>
                                        <span className={`text-xs px-2 py-0.5 rounded-full font-bold ${tier.badgeColor}`}>{tier.badge}</span>
                                    </div>
                                    <p className="text-slate-500 text-xs mt-0.5">{tier.desc}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-6">
                                <div className="text-right hidden md:block">
                                    <p className="text-slate-400 text-xs">الإنتاجية</p>
                                    <p className="text-white text-sm font-bold">{tier.throughput}</p>
                                </div>
                                <div className="text-right">
                                    <p className="text-slate-400 text-xs">العزل</p>
                                    <p className="text-rose-400 text-sm font-bold">{tier.isolation}</p>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Info Notice */}
            <div className="flex items-start gap-4 p-5 rounded-2xl bg-rose-500/5 border border-rose-500/15 border-dashed">
                <div className="p-3 rounded-xl bg-rose-500/10 shrink-0 mt-0.5">
                    <Shield className="w-5 h-5 text-rose-400" />
                </div>
                <div>
                    <p className="text-rose-300 font-bold text-sm">هذا القسم قيد الإعداد</p>
                    <p className="text-slate-500 text-xs mt-1">
                        سيتمكن المشرف من تعريف بوابات الهدهد، تحديد نوع كل بوابة (مشتركة / مخصصة / مؤسسية)، تخصيصها للشركات، وإدارة تعرفات استخدام كل فئة بشكل مستقل.
                    </p>
                </div>
            </div>
        </div>
    );
}
