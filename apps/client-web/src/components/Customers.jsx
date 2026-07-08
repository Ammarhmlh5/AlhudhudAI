import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { MessageSquare, User, Search, ArrowRight, Clock, CheckCircle, XCircle, ChevronLeft } from 'lucide-react';

const Customers = () => {
    const [allMessages, setAllMessages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedCustomer, setSelectedCustomer] = useState(null); // The phone number being viewed

    useEffect(() => {
        fetchData();
    }, []);

    const fetchData = async () => {
        setLoading(true);
        try {
            // Fetch a larger set to get unique customers
            const response = await apiClient.get('/api/v1/messaging/list?limit=500');
            setAllMessages(response.data.messages || []);
        } catch (error) {
            console.error("Error fetching data for customers", error);
        } finally {
            setLoading(false);
        }
    };

    // Group messages by phone number
    const customersMap = allMessages.reduce((acc, msg) => {
        if (!acc[msg.phone_number]) {
            acc[msg.phone_number] = {
                phone: msg.phone_number,
                count: 0,
                lastMessage: msg,
                messages: []
            };
        }
        acc[msg.phone_number].count += 1;
        acc[msg.phone_number].messages.push(msg);

        // Ensure we keep the most recent message as lastMessage
        if (new Date(msg.created_at) > new Date(acc[msg.phone_number].lastMessage.created_at)) {
            acc[msg.phone_number].lastMessage = msg;
        }
        return acc;
    }, {});

    const customersList = Object.values(customersMap)
        .filter(c => c.phone.includes(searchQuery))
        .sort((a, b) => new Date(b.lastMessage.created_at) - new Date(a.lastMessage.created_at));

    if (selectedCustomer) {
        const customerData = customersMap[selectedCustomer];
        const sortedHistory = [...customerData.messages].sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

        return (
            <div className="animate-in fade-in slide-in-from-right-8 duration-500">
                <button
                    onClick={() => setSelectedCustomer(null)}
                    className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
                >
                    <ChevronLeft className="w-4 h-4" />
                    العودة لقائمة العملاء
                </button>

                <div className="flex items-center gap-4 mb-8">
                    <div className="w-16 h-16 bg-rose-500/10 rounded-2xl flex items-center justify-center text-rose-500">
                        <User className="w-8 h-8" />
                    </div>
                    <div>
                        <h2 className="text-3xl font-bold font-mono">{selectedCustomer}</h2>
                        <p className="text-slate-400 text-sm mt-1">إجمالي الرسائل المرسلة: {customerData.count}</p>
                    </div>
                </div>

                <div className="glass-card overflow-hidden">
                    <div className="px-6 py-4 border-b border-slate-800 bg-slate-800/30">
                        <h3 className="text-sm font-bold text-slate-300">سجل المراسلات</h3>
                    </div>
                    <div className="divide-y divide-slate-800">
                        {sortedHistory.map(msg => (
                            <div key={msg.id} className="p-6 hover:bg-slate-800/10 transition-colors flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <p className="text-slate-200 text-sm mb-2 text-right leading-relaxed">{msg.content}</p>
                                    <div className="flex items-center gap-4 text-xs text-slate-500">
                                        <span className="flex items-center gap-1 font-mono">
                                            <Clock className="w-3 h-3" />
                                            {new Date(msg.created_at).toLocaleString('ar-SA')}
                                        </span>
                                    </div>
                                </div>
                                <div className="flex flex-col items-end gap-2 shrink-0">
                                    <div className="flex items-center gap-2">
                                        {(msg.status === 'DELIVERED' || msg.status === 'SENT') && <CheckCircle className="w-4 h-4 text-emerald-500" />}
                                        {msg.status === 'FAILED' && <XCircle className="w-4 h-4 text-rose-500" />}
                                        {['WAITING', 'PENDING', 'PROCESSING'].includes(msg.status) && <Clock className="w-4 h-4 text-amber-500" />}
                                        <span className={`text-[10px] font-black uppercase tracking-widest ${(msg.status === 'DELIVERED' || msg.status === 'SENT') ? 'text-emerald-500' :
                                            msg.status === 'FAILED' ? 'text-rose-500' : 'text-amber-500'
                                            }`}>{msg.status}</span>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="animate-in fade-in slide-in-from-right-8 duration-500">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-8">
                <div>
                    <h2 className="text-3xl font-bold">العملاء (Customers)</h2>
                    <p className="text-slate-400 text-sm mt-1">تتبع العملاء عبر تاريخ المراسلات.</p>
                </div>

                <div className="relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                    <input
                        type="text"
                        placeholder="بحث برقم الهاتف..."
                        className="bg-slate-900 border border-slate-800 rounded-xl pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-rose-500 w-64"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                </div>
            </div>

            {loading ? (
                <div className="flex items-center justify-center p-20">
                    <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-rose-500" />
                </div>
            ) : customersList.length === 0 ? (
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-16 text-center">
                    <div className="w-16 h-16 bg-slate-800 rounded-2xl flex items-center justify-center mx-auto mb-4">
                        <MessageSquare className="w-8 h-8 text-slate-600" />
                    </div>
                    <h3 className="text-xl font-bold mb-2">لا يوجد عملاء</h3>
                    <p className="text-slate-500 text-sm max-w-sm mx-auto">
                        لم يتم العثور على أي مراسلات سابقة مع عملاء.
                    </p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                    {customersList.map(customer => (
                        <button
                            key={customer.phone}
                            onClick={() => setSelectedCustomer(customer.phone)}
                            className="bg-slate-900/50 border border-slate-800 rounded-2xl p-6 text-right hover:border-rose-500/50 hover:bg-slate-800/40 transition-all group"
                        >
                            <div className="flex items-start justify-between mb-4">
                                <div className="w-12 h-12 bg-slate-800 rounded-xl flex items-center justify-center text-slate-400 group-hover:bg-rose-500/10 group-hover:text-rose-500 transition-colors">
                                    <User className="w-6 h-6" />
                                </div>
                                <div className="text-left">
                                    <span className="text-[10px] font-black text-slate-600 uppercase tracking-tighter bg-slate-800 px-2 py-1 rounded">
                                        إجمالي الرسائل: {customer.count}
                                    </span>
                                </div>
                            </div>

                            <h3 className="text-lg font-bold font-mono text-slate-100 mb-1">{customer.phone}</h3>
                            <p className="text-slate-500 text-xs mb-4 line-clamp-2 leading-relaxed">
                                {customer.lastMessage.content}
                            </p>

                            <div className="flex items-center justify-between pt-4 border-t border-slate-800/50">
                                <span className="text-[10px] text-slate-600 font-mono">
                                    {new Date(customer.lastMessage.created_at).toLocaleDateString('ar-SA')}
                                </span>
                                <div className="flex items-center gap-1 text-rose-500 text-xs font-bold">
                                    عرض السجل
                                    <ArrowRight className="w-3 h-3 rotate-180" />
                                </div>
                            </div>
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
};

export default Customers;
