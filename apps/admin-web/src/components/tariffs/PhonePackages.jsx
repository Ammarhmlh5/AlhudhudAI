import { Smartphone, Package2, Zap, Clock, Users, TrendingUp } from 'lucide-react';

const stats = [
    { label: 'إجمالي الباقات النشطة', value: '—', icon: Package2, color: 'text-violet-400', bg: 'bg-violet-500/10 border-violet-500/20' },
    { label: 'المشتركون في الباقات', value: '—', icon: Users, color: 'text-sky-400', bg: 'bg-sky-500/10 border-sky-500/20' },
    { label: 'الرسائل اليومية', value: '—', icon: Zap, color: 'text-amber-400', bg: 'bg-amber-500/10 border-amber-500/20' },
    { label: 'معدل التجديد', value: '—', icon: TrendingUp, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20' },
];

const planTypes = [
    { name: 'باقة ستارتر', messages: '1,000 رسالة / شهر', devices: 'جهاز واحد', price: '—', badge: 'أساسية', badgeColor: 'bg-slate-700 text-slate-300' },
    { name: 'باقة بيزنس', messages: '10,000 رسالة / شهر', devices: '5 أجهزة', price: '—', badge: 'شائعة', badgeColor: 'bg-violet-600/30 text-violet-300' },
    { name: 'باقة برو', messages: '50,000 رسالة / شهر', devices: '20 جهاز', price: '—', badge: 'متقدمة', badgeColor: 'bg-sky-600/30 text-sky-300' },
    { name: 'باقة إنتربرايز', messages: 'غير محدودة', devices: 'أجهزة غير محدودة', price: '—', badge: 'مؤسسية', badgeColor: 'bg-amber-600/30 text-amber-300' },
];

export default function PhonePackages() {
    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-3 duration-500" dir="rtl">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-3 rounded-2xl bg-violet-500/10 border border-violet-500/20">
                        <Smartphone className="w-6 h-6 text-violet-400" />
                    </div>
                    <div>
                        <h3 className="text-xl font-bold text-white">باقات الهواتف</h3>
                        <p className="text-slate-400 text-sm mt-0.5">إدارة باقات الاشتراك للهواتف والأجهزة المحمولة</p>
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
                        <div className="flex items-center justify-between">
                            <stat.icon className={`w-5 h-5 ${stat.color}`} />
                        </div>
                        <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                        <p className="text-slate-400 text-xs leading-relaxed">{stat.label}</p>
                    </div>
                ))}
            </div>

            {/* Plans Table */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl overflow-hidden">
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
                    <h4 className="text-white font-bold">هيكل الباقات المخطط</h4>
                    <span className="text-xs text-slate-500 bg-slate-800 px-3 py-1 rounded-full">معاينة / غير مفعّل</span>
                </div>
                <div className="divide-y divide-slate-800/50">
                    {planTypes.map((plan) => (
                        <div key={plan.name} className="flex items-center justify-between px-6 py-4 hover:bg-slate-800/30 transition-colors group">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-slate-800 group-hover:bg-slate-700 transition-colors">
                                    <Smartphone className="w-4 h-4 text-violet-400" />
                                </div>
                                <div>
                                    <div className="flex items-center gap-2">
                                        <p className="text-white font-bold text-sm">{plan.name}</p>
                                        <span className={`text-xs px-2 py-0.5 rounded-full font-bold ${plan.badgeColor}`}>{plan.badge}</span>
                                    </div>
                                    <p className="text-slate-500 text-xs mt-0.5">{plan.devices}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-6">
                                <div className="text-right hidden md:block">
                                    <p className="text-slate-400 text-xs">الرسائل</p>
                                    <p className="text-white text-sm font-bold">{plan.messages}</p>
                                </div>
                                <div className="text-right">
                                    <p className="text-slate-400 text-xs">السعر</p>
                                    <p className="text-violet-400 text-sm font-bold">{plan.price}</p>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Coming Soon Notice */}
            <div className="flex items-center gap-4 p-5 rounded-2xl bg-violet-500/5 border border-violet-500/15 border-dashed">
                <div className="p-3 rounded-xl bg-violet-500/10 shrink-0">
                    <Package2 className="w-5 h-5 text-violet-400" />
                </div>
                <div>
                    <p className="text-violet-300 font-bold text-sm">هذا القسم قيد الإعداد</p>
                    <p className="text-slate-500 text-xs mt-1">
                        سيتمكن المشرف من إنشاء وتعديل وأرشفة باقات الهواتف، وربطها بالمستأجرين، وتحديد حدود الرسائل والأجهزة لكل باقة.
                    </p>
                </div>
            </div>
        </div>
    );
}
