import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Headphones, MessageSquare, Archive, MoreHorizontal, Send, User } from 'lucide-react';
import { API_BASE_URL } from '../config';

export default function SupportCenter() {
    const { t } = useTranslation();
    const [tickets, setTickets] = useState([]);
    const [activeTicket, setActiveTicket] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchTickets();
    }, []);

    const fetchTickets = async () => {
        try {
            setLoading(true);
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/support/tickets`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` } });
            if (res.ok) {
                let data = await res.json();
                if (!data) data = [];
                const mapped = data.map(t => ({
                    id: t.id,
                    user: t('supportCenter.user_id', { id: t.user_id }),
                    subject: t.subject || t('supportCenter.no_subject'),
                    status: mapStatus(t.status),
                    priority: mapPriority(t.priority),
                    priorityNum: t.priority,
                    last_msg: t.description || t('supportCenter.no_description'),
                    time: new Date(t.created_at).toLocaleDateString()
                }));
                setTickets(mapped);
                if (mapped.length > 0) setActiveTicket(mapped[0]);
            }
        } catch (err) {
            console.error('Failed to fetch tickets:', err);
        } finally {
            setLoading(false);
        }
    };

    const mapStatus = (s) => {
        switch(s) {
            case 1: return t('supportCenter.status_open');
            case 2: return t('supportCenter.status_in_progress');
            case 3: return t('supportCenter.status_pending');
            case 4: return t('supportCenter.status_resolved');
            case 5: return t('supportCenter.status_closed');
            default: return t('supportCenter.status_open');
        }
    };

    const mapPriority = (p) => {
        switch(p) {
            case 1: return t('supportCenter.priority_low');
            case 2: return t('supportCenter.priority_medium');
            case 3: return t('supportCenter.priority_high');
            case 4: return t('supportCenter.priority_urgent');
            default: return t('supportCenter.priority_medium');
        }
    };

    return (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 h-[calc(100vh-140px)] animate-in fade-in zoom-in-95 duration-500">

            {/* Ticket List */}
            <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden flex flex-col">
                <div className="p-4 border-b border-slate-800 flex justify-between items-center bg-slate-950/50">
                    <h3 className="font-bold text-white flex items-center gap-2">
                        <Archive className="w-4 h-4 text-slate-400" /> {t('supportCenter.inbox')}
                    </h3>
                    <span className="text-xs bg-rose-600 text-white px-2 py-0.5 rounded-full">{t('supportCenter.total', { count: tickets.length })}</span>
                </div>
                <div className="flex-1 overflow-y-auto custom-scrollbar">
                    {loading && <div className="p-4 text-slate-500 text-sm">{t('supportCenter.loading_tickets')}</div>}
                    {!loading && tickets.length === 0 && <div className="p-4 text-slate-500 text-sm">{t('supportCenter.no_tickets_found')}</div>}
                    {!loading && tickets.map(ticket => (
                        <div
                            key={ticket.id}
                            onClick={() => setActiveTicket(ticket)}
                            className={`p-4 border-b border-slate-800 cursor-pointer transition-colors ${activeTicket.id === ticket.id ? 'bg-slate-800/50 border-l-4 border-l-rose-500' : 'hover:bg-slate-800/30 border-l-4 border-l-transparent'
                                }`}
                        >
                            <div className="flex justify-between items-start mb-1">
                                <span className="font-bold text-white text-sm">{ticket.user}</span>
                                <span className="text-[10px] text-slate-500">{ticket.time}</span>
                            </div>
                            <p className="text-xs text-rose-400 font-medium mb-1">{ticket.subject}</p>
                            <p className="text-xs text-slate-400 truncate">{ticket.last_msg}</p>
                            <div className="mt-2 flex gap-2">
                                <span className={`text-[9px] px-1.5 py-0.5 rounded uppercase font-bold ${ticket.priorityNum >= 3 ? 'bg-rose-500/10 text-rose-500' : 'bg-slate-700 text-slate-400'
                                    }`}>{ticket.priority}</span>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Chat View */}
            <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden flex flex-col">
                {activeTicket ? (
                <>
                {/* Header */}
                <div className="p-4 border-b border-slate-800 flex justify-between items-center bg-slate-950/50">
                    <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-full bg-slate-800 flex items-center justify-center text-rose-500 font-bold">
                            {(activeTicket.user || 'U').charAt(0)}
                        </div>
                        <div>
                            <h3 className="font-bold text-white">{activeTicket.subject}</h3>
                            <p className="text-xs text-slate-400 flex items-center gap-1">
                                <User className="w-3 h-3" /> {activeTicket.user}
                            </p>
                        </div>
                    </div>
                    <button className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 transition-colors">
                        <MoreHorizontal className="w-5 h-5" />
                    </button>
                </div>

                {/* Messages */}
                <div className="flex-1 p-6 space-y-6 overflow-y-auto custom-scrollbar bg-slate-950/30">
                    <div className="flex gap-4">
                        <div className="w-8 h-8 rounded-full bg-slate-800 flex-shrink-0 flex items-center justify-center text-xs font-bold text-rose-500">
                            {activeTicket.user.charAt(0)}
                        </div>
                        <div className="bg-slate-800 rounded-2xl p-4 rounded-tl-none max-w-lg text-sm text-slate-200">
                            <p>{activeTicket.last_msg}</p>
                            <span className="text-[10px] text-slate-500 mt-2 block">10:42 AM</span>
                        </div>
                    </div>

                    <div className="flex gap-4 flex-row-reverse">
                        <div className="w-8 h-8 rounded-full bg-rose-600 flex-shrink-0 flex items-center justify-center text-xs font-bold text-white">
                            {t('supportCenter.you')}
                        </div>
                        <div className="bg-rose-600/10 border border-rose-600/20 rounded-2xl p-4 rounded-tr-none max-w-lg text-sm text-rose-100">
                            <p>{t('supportCenter.demo_reply')}</p>
                            <span className="text-[10px] text-rose-400/50 mt-2 block">10:45 AM</span>
                        </div>
                    </div>
                </div>

                {/* Input */}
                <div className="p-4 bg-slate-900 border-t border-slate-800">
                    <div className="relative">
                        <input
                            type="text"
                            placeholder={t('supportCenter.reply_placeholder')}
                            className="w-full bg-slate-950 border border-slate-800 rounded-xl pl-4 pr-12 py-3 text-sm text-white focus:ring-1 focus:ring-rose-500 outline-none transition-all"
                        />
                        <button className="absolute right-2 top-2 p-1.5 bg-rose-600 hover:bg-rose-500 text-white rounded-lg transition-colors">
                            <Send className="w-4 h-4" />
                        </button>
                    </div>
                </div>
                </>
                ) : (
                    <div className="flex-1 flex flex-col items-center justify-center text-slate-500 p-8">
                        <MessageSquare className="w-16 h-16 mb-4 opacity-50 text-slate-600" />
                        <h2 className="text-xl font-bold text-white mb-2">{t('supportCenter.no_ticket_selected')}</h2>
                        <p className="text-center max-w-sm">{t('supportCenter.no_ticket_selected_hint')}</p>
                    </div>
                )}
            </div>

        </div>
    );
}
