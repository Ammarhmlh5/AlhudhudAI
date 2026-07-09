import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';
import { Clock, AlertCircle, RefreshCcw, Smartphone, Settings, Info, Search } from 'lucide-react';
import { useSocket } from '../contexts/SocketContext';

const WaitingMessages = () => {
    const [messages, setMessages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const { messageUpdateNonce } = useSocket();

    useEffect(() => {
        fetchWaitingMessages();
        const interval = setInterval(fetchWaitingMessages, 5000); // Poll every 5 seconds
        return () => clearInterval(interval);
    }, [messageUpdateNonce]);

    const fetchWaitingMessages = async () => {
        try {
            const response = await apiClient.get('/api/v1/tenant/messages/history?status=PENDING');
            setMessages(response.data.messages || []);
        } catch (error) {
            console.error('Failed to fetch waiting messages:', error);
        } finally {
            setLoading(false);
        }
    };

    const filteredMessages = messages.filter(msg =>
        msg.phone_number.includes(searchTerm) || msg.content.toLowerCase().includes(searchTerm.toLowerCase())
    );

    return (
        <div className="animate-in slide-in-from-right-8 duration-500">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h2 className="text-3xl font-bold">بانتظار الربط (Wait Mode)</h2>
                    <p className="text-slate-400 text-sm mt-1">
                        رسائل معلقة في النظام بانتظار اتصال أجهزتك (Hibernation Mode).
                    </p>
                </div>
                <div className="bg-amber-500/10 border border-amber-500/20 p-4 rounded-2xl flex items-center gap-4">
                    <AlertCircle className="w-6 h-6 text-amber-500" />
                    <div>
                        <div className="text-xs text-amber-500 font-bold uppercase tracking-widest">Status: Hibernate</div>
                        <div className="text-sm text-slate-300 font-bold">{filteredMessages.length} Messages Buffered</div>
                    </div>
                </div>
            </div>

            {/* Filter Bar */}
            <div className="glass-card p-4 mb-6 flex items-center gap-4 border-slate-800/50">
                <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                    <input
                        type="text"
                        placeholder="Search waiting messages..."
                        className="w-full bg-slate-900/50 border border-slate-800 rounded-xl py-2 pl-10 pr-4 text-sm focus:outline-none focus:border-amber-500/50 transition-colors"
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                    />
                </div>
                <button className="p-2 bg-slate-800 hover:bg-slate-700 rounded-xl transition-colors">
                    <RefreshCcw className="w-5 h-5 text-slate-400" />
                </button>
            </div>

            <div className="glass-card overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="bg-slate-900/50 border-b border-slate-800">
                                <th className="px-6 py-4 text-[10px] font-bold uppercase tracking-widest text-slate-500">Recipient</th>
                                <th className="px-6 py-4 text-[10px] font-bold uppercase tracking-widest text-slate-500">Content</th>
                                <th className="px-6 py-4 text-[10px] font-bold uppercase tracking-widest text-slate-500">Wait Reason</th>
                                <th className="px-6 py-4 text-[10px] font-bold uppercase tracking-widest text-slate-500">Age</th>
                                <th className="px-6 py-4 text-[10px] font-bold uppercase tracking-widest text-slate-500 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800/50">
                            {filteredMessages.length === 0 ? (
                                <tr>
                                    <td colSpan="5" className="px-6 py-20 text-center">
                                        <div className="flex flex-col items-center gap-4 opacity-40">
                                            <div className="bg-slate-800 p-6 rounded-full">
                                                <Clock className="w-10 h-10 text-slate-400" />
                                            </div>
                                            <p className="text-slate-500 italic">No messages are currently in wait mode.</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : filteredMessages.map((msg) => (
                                <tr key={msg.id} className="hover:bg-amber-500/[0.02] transition-colors group">
                                    <td className="px-6 py-5">
                                        <div className="font-mono text-sm text-slate-200">{msg.phone_number}</div>
                                    </td>
                                    <td className="px-6 py-5">
                                        <div className="text-sm text-slate-400 truncate max-w-xs">{msg.content}</div>
                                    </td>
                                    <td className="px-6 py-5">
                                        <div className="flex items-center gap-2">
                                            <div className="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse" />
                                            <span className="text-xs font-bold text-amber-500 bg-amber-500/10 px-2 py-1 rounded-md border border-amber-500/20">
                                                {msg.wait_reason || 'OFFLINE Device'}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-6 py-5 text-xs text-slate-500 font-mono">
                                        {new Date(msg.created_at).toLocaleTimeString()}
                                    </td>
                                    <td className="px-6 py-5 text-right">
                                        <div className="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                            <button className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-all">
                                                <Settings className="w-4 h-4" />
                                            </button>
                                            <button className="p-2 hover:bg-slate-800 rounded-lg text-rose-500 transition-all">
                                                <AlertCircle className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Instruction Footer */}
            <div className="mt-6 flex items-start gap-4 p-4 bg-slate-900/30 rounded-2xl border border-slate-800/50">
                <Info className="w-5 h-5 text-blue-500 shrink-0 mt-0.5" />
                <p className="text-[11px] text-slate-500 leading-relaxed">
                    <strong>Note:</strong> Messages in <span className="text-amber-500">Wait Mode</span> are queued locally on the server. They will be automatically pushed (Fast-Forward) to your devices as soon as a connection is established. Financial settlement occurs only upon successful delivery.
                </p>
            </div>
        </div>
    );
};

export default WaitingMessages;
