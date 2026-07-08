import { useState, useEffect } from 'react';
import { FileText, Plus, Edit, Trash2, Globe, Check, X, Calendar } from 'lucide-react';
import { API_BASE_URL } from '../config';

export default function TariffSettings() {
    const [tariffs, setTariffs] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingTariff, setEditingTariff] = useState(null);
    const [formData, setFormData] = useState({
        country_code: 'YE',
        network_operator: '',
        message_type: 'standard',
        cost_per_message: '',
        currency: 'YER',
        effective_from: new Date().toISOString().split('T')[0],
        is_active: true
    });

    useEffect(() => {
        fetchTariffs();
    }, []);

    const fetchTariffs = async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/tariffs`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
            });
            
            if (res.ok) {
                const data = await res.json();
                setTariffs(data);
            }
        } catch (err) {
            // Handled

        } finally {
            setLoading(false);
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        
        try {
            const url = editingTariff 
                ? `${API_BASE_URL}/api/v1/admin/tariffs/${editingTariff.id}`
                : `${API_BASE_URL}/api/v1/admin/tariffs`;

            const res = await fetch(url, {
                method: editingTariff ? 'PUT' : 'POST',
                headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(formData),
            });
            
            if (res.ok) {
                await fetchTariffs();
                handleCloseModal();
            }
        } catch (err) {
            console.error("Failed to save tariff", err);
        }
    };

    const handleEdit = (tariff) => {
        setEditingTariff(tariff);
        setFormData({
            country_code: tariff.country_code,
            network_operator: tariff.network_operator || '',
            message_type: tariff.message_type || 'standard',
            cost_per_message: tariff.cost_per_message,
            currency: tariff.currency || 'YER',
            effective_from: tariff.effective_from || new Date().toISOString().split('T')[0],
            is_active: tariff.is_active
        });
        setShowModal(true);
    };

    const handleDelete = async (id) => {
        if (!confirm('Are you sure you want to delete this tariff?')) return;
        
        try {
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/tariffs/${id}`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
                method: 'DELETE',
            });
            
            if (res.ok) {
                await fetchTariffs();
            }
        } catch (err) {
            console.error("Failed to delete tariff", err);
        }
    };

    const handleCloseModal = () => {
        setShowModal(false);
        setEditingTariff(null);
        setFormData({
            country_code: 'YE',
            network_operator: '',
            message_type: 'standard',
            cost_per_message: '',
            currency: 'YER',
            effective_from: new Date().toISOString().split('T')[0],
            is_active: true
        });
    };

    if (loading) {
        return (
            <div className="text-center py-12 text-slate-500">Loading tariffs...</div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h3 className="text-xl font-black text-white flex items-center gap-2">
                        <FileText className="w-5 h-5 text-rose-500" />
                        Tariff Management
                    </h3>
                    <p className="text-sm text-slate-500 mt-1">Configure pricing rates for different networks and countries</p>
                </div>
                <button
                    onClick={() => setShowModal(true)}
                    className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold text-sm transition-colors shadow-lg shadow-rose-900/20 flex items-center gap-2"
                >
                    <Plus className="w-4 h-4" /> Add Tariff
                </button>
            </div>

            <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden">
                <div className="overflow-x-auto w-full">
<table className="w-full text-left">
                    <thead className="bg-slate-800/50 text-slate-400 text-[10px] uppercase tracking-widest font-bold">
                        <tr>
                            <th className="px-6 py-4">Country/Network</th>
                            <th className="px-6 py-4">Type</th>
                            <th className="px-6 py-4">Rate</th>
                            <th className="px-6 py-4">Effective From</th>
                            <th className="px-6 py-4">Status</th>
                            <th className="px-6 py-4 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800 text-slate-300">
                        {tariffs.length === 0 ? (
                            <tr>
                                <td colSpan="6" className="px-6 py-12 text-center text-slate-500">
                                    <FileText className="w-12 h-12 mx-auto mb-3 opacity-30" />
                                    <p>No tariffs configured yet</p>
                                </td>
                            </tr>
                        ) : (
                            tariffs.map((tariff) => (
                                <tr key={tariff.id} className="hover:bg-slate-800/30 transition-colors">
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-3">
                                            <div className="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center">
                                                <Globe className="w-5 h-5 text-blue-500" />
                                            </div>
                                            <div>
                                                <div className="font-bold text-white">{tariff.country_code}</div>
                                                <div className="text-xs text-slate-500">
                                                    {tariff.network_operator || 'All Networks'}
                                                </div>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className="px-2 py-1 rounded-lg text-xs font-bold bg-slate-800 text-slate-300">
                                            {tariff.message_type || 'Standard'}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4">
                                        <div className="font-mono text-emerald-400 font-bold">
                                            {parseFloat(tariff.cost_per_message || 0).toFixed(4)} {tariff.currency}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-2 text-xs text-slate-400">
                                            <Calendar className="w-4 h-4" />
                                            {tariff.effective_from || 'N/A'}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <span className={`flex items-center gap-1.5 text-xs font-bold ${
                                            tariff.is_active ? 'text-emerald-400' : 'text-slate-500'
                                        }`}>
                                            <div className={`w-1.5 h-1.5 rounded-full ${
                                                tariff.is_active ? 'bg-emerald-500 animate-pulse' : 'bg-slate-500'
                                            }`} />
                                            {tariff.is_active ? 'Active' : 'Inactive'}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 text-right">
                                        <div className="flex items-center justify-end gap-2">
                                            <button
                                                onClick={() => handleEdit(tariff)}
                                                className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"
                                            >
                                                <Edit className="w-4 h-4" />
                                            </button>
                                            <button
                                                onClick={() => handleDelete(tariff.id)}
                                                className="p-2 hover:bg-rose-500/10 rounded-lg text-slate-400 hover:text-rose-400 transition-colors"
                                            >
                                                <Trash2 className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
</div>
            </div>

            {/* Add/Edit Modal */}
            {showModal && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-2xl w-full animate-in fade-in zoom-in-95">
                        <div className="flex items-center justify-between mb-6">
                            <div className="flex items-center gap-3">
                                <div className="w-12 h-12 rounded-2xl bg-rose-500/10 flex items-center justify-center">
                                    <FileText className="w-6 h-6 text-rose-500" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-white">
                                        {editingTariff ? 'Edit Tariff' : 'Add New Tariff'}
                                    </h3>
                                    <p className="text-xs text-slate-500">Configure pricing rate</p>
                                </div>
                            </div>
                            <button
                                onClick={handleCloseModal}
                                className="p-2 hover:bg-slate-800 rounded-lg text-slate-400 hover:text-white transition-colors"
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Country Code
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.country_code}
                                        onChange={(e) => setFormData({...formData, country_code: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="YE"
                                        required
                                    />
                                </div>

                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Network Operator
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.network_operator}
                                        onChange={(e) => setFormData({...formData, network_operator: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="Yemen Mobile (optional)"
                                    />
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Message Type
                                    </label>
                                    <select
                                        value={formData.message_type}
                                        onChange={(e) => setFormData({...formData, message_type: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                    >
                                        <option value="standard">Standard</option>
                                        <option value="unicode">Unicode</option>
                                    </select>
                                </div>

                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Currency
                                    </label>
                                    <select
                                        value={formData.currency}
                                        onChange={(e) => setFormData({...formData, currency: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                    >
                                        <option value="YER">YER</option>
                                        <option value="USD">USD</option>
                                        <option value="SAR">SAR</option>
                                    </select>
                                </div>
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Cost Per Message
                                    </label>
                                    <input
                                        type="number"
                                        step="0.0001"
                                        value={formData.cost_per_message}
                                        onChange={(e) => setFormData({...formData, cost_per_message: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="0.0000"
                                        required
                                    />
                                </div>

                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Effective From
                                    </label>
                                    <input
                                        type="date"
                                        value={formData.effective_from}
                                        onChange={(e) => setFormData({...formData, effective_from: e.target.value})}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        required
                                    />
                                </div>
                            </div>

                            <div className="flex items-center gap-3 p-4 bg-slate-950 rounded-xl border border-slate-800">
                                <input
                                    type="checkbox"
                                    id="is_active"
                                    checked={formData.is_active}
                                    onChange={(e) => setFormData({...formData, is_active: e.target.checked})}
                                    className="w-5 h-5 rounded bg-slate-800 border-slate-700 text-rose-500 focus:ring-rose-500"
                                />
                                <label htmlFor="is_active" className="text-sm text-white font-medium cursor-pointer">
                                    Active (Apply this tariff immediately)
                                </label>
                            </div>

                            <div className="flex gap-3 mt-8">
                                <button
                                    type="button"
                                    onClick={handleCloseModal}
                                    className="flex-1 py-3 text-slate-400 font-bold hover:text-white transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className="flex-1 py-3 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold transition-colors shadow-lg shadow-rose-900/20 flex items-center justify-center gap-2"
                                >
                                    <Check className="w-4 h-4" />
                                    {editingTariff ? 'Update Tariff' : 'Create Tariff'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}
