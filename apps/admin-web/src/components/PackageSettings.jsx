import { useState, useEffect } from 'react';
import { Package, Plus, Edit, Trash2, Users, Check, X, DollarSign, MessageSquare, Info, ToggleLeft, ToggleRight } from 'lucide-react';
import { API_BASE_URL } from '../config';

export default function PackageSettings() {
    const [packages, setPackages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingPackage, setEditingPackage] = useState(null);
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        monthly_fee: '',
        messages_included: '',
        billing_cycle: 'monthly',
        rollover_enabled: false,
        rollover_limit: '',
        rollover_expiry_months: '1',
        conditional_rollover_enabled: false,
        is_active: true
    });

    useEffect(() => {
        fetchPackages();
    }, []);

    const fetchPackages = async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/packages`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
            });

            if (res.ok) {
                const data = await res.json();
                setPackages(data);
            }
        } catch (err) {
            console.error("Failed to fetch packages", err);
        } finally {
            setLoading(false);
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();

        try {
            const url = editingPackage
                ? `${API_BASE_URL}/api/v1/admin/packages/${editingPackage.id}`
                : `${API_BASE_URL}/api/v1/admin/packages`;

            const res = await fetch(url, {
                method: editingPackage ? 'PUT' : 'POST',
                headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(formData),
            });

            if (res.ok) {
                await fetchPackages();
                handleCloseModal();
            }
        } catch (err) {
            console.error("Failed to save package", err);
        }
    };

    const handleEdit = (pkg) => {
        setEditingPackage(pkg);
        setFormData({
            name: pkg.name,
            description: pkg.description || '',
            monthly_fee: pkg.monthly_fee,
            messages_included: pkg.messages_included || '',
            billing_cycle: pkg.billing_cycle || 'monthly',
            rollover_enabled: pkg.rollover_enabled || false,
            rollover_limit: pkg.rollover_limit || '',
            rollover_expiry_months: pkg.rollover_expiry_months || '1',
            conditional_rollover_enabled: pkg.conditional_rollover_enabled || false,
            is_active: pkg.is_active
        });
        setShowModal(true);
    };

    const handleDelete = async (id) => {
        if (!confirm('Are you sure you want to delete this package?')) return;

        try {
            const res = await fetch(`${API_BASE_URL}/api/v1/admin/packages/${id}`, { headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
                method: 'DELETE',
            });

            if (res.ok) {
                await fetchPackages();
            }
        } catch (err) {
            console.error("Failed to delete package", err);
        }
    };

    const handleCloseModal = () => {
        setShowModal(false);
        setEditingPackage(null);
        setFormData({
            name: '',
            description: '',
            monthly_fee: '',
            messages_included: '',
            billing_cycle: 'monthly',
            rollover_enabled: false,
            rollover_limit: '',
            rollover_expiry_months: '1',
            conditional_rollover_enabled: false,
            is_active: true
        });
    };

    if (loading) {
        return (
            <div className="text-center py-12 text-slate-500">Loading packages...</div>
        );
    }

    return (
        <div className="space-y-6 flex-1 w-full bg-slate-950 min-h-screen text-white rounded-lg border-slate-800 p-6">
            <div className="flex justify-between items-center">
                <div>
                    <h3 className="text-xl font-black text-rose-500">
                        Package Management
                    </h3>
                    <p className="text-sm text-slate-400 mt-1">Configure subscription packages and pricing plans</p>
                </div>
                <button
                    onClick={() => setShowModal(true)}
                    className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-xl font-bold text-sm transition-colors shadow-lg shadow-rose-900/20 flex items-center gap-2"
                >
                    + Add Package
                </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {packages.length === 0 ? (
                    <div className="col-span-full text-center py-12 text-slate-500 border border-dashed border-slate-800 rounded-3xl">
                        <Package className="w-12 h-12 mx-auto mb-3 opacity-30" />
                        <p>No packages configured yet</p>
                    </div>
                ) : (
                    packages.map((pkg) => (
                        <div
                            key={pkg.id}
                            className="bg-slate-900 border border-slate-800 rounded-3xl p-6 hover:border-slate-700 transition-all group"
                        >
                            <div className="flex justify-between items-start mb-4">
                                <div className="w-12 h-12 rounded-2xl bg-rose-500/10 flex items-center justify-center">
                                    <Package className="w-6 h-6 text-rose-500" />
                                </div>
                                <span className={`px-2 py-1 rounded-lg text-[10px] font-bold uppercase ${pkg.is_active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-slate-800 text-slate-500'
                                    }`}>
                                    {pkg.is_active ? 'Active' : 'Inactive'}
                                </span>
                            </div>

                            <h3 className="text-lg font-bold text-white mb-1">{pkg.name}</h3>
                            <p className="text-xs text-slate-500 mb-4 line-clamp-2">{pkg.description || 'No description'}</p>

                            <div className="space-y-3 mb-6">
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-slate-500">Monthly Fee</span>
                                    <span className="font-mono text-emerald-400 font-bold">
                                        ${parseFloat(pkg.monthly_fee || 0).toFixed(2)}
                                    </span>
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-slate-500">Messages</span>
                                    <span className="text-white font-bold">
                                        {pkg.messages_included ? pkg.messages_included.toLocaleString() : 'Unlimited'}
                                    </span>
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-slate-500">Rollover</span>
                                    <span className={`text-xs font-bold ${pkg.rollover_enabled ? 'text-emerald-400' : 'text-slate-500'}`}>
                                        {pkg.rollover_enabled ? 'Enabled' : 'Disabled'}
                                    </span>
                                </div>
                                {pkg.rollover_enabled && pkg.conditional_rollover_enabled && (
                                    <div className="flex items-center gap-2 p-2 bg-blue-500/10 rounded-lg border border-blue-500/20">
                                        <Info className="w-3 h-3 text-blue-400 shrink-0" />
                                        <span className="text-[10px] text-blue-400 font-bold">Conditional Rollover Active</span>
                                    </div>
                                )}
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-slate-500">Subscribers</span>
                                    <span className="flex items-center gap-1 text-white font-bold">
                                        <Users className="w-3 h-3" />
                                        {pkg.subscriber_count || 0}
                                    </span>
                                </div>
                            </div>

                            <div className="flex gap-2 pt-4 border-t border-slate-800">
                                <button
                                    onClick={() => handleEdit(pkg)}
                                    className="flex-1 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-xl text-xs font-bold transition-colors flex items-center justify-center gap-2"
                                >
                                    <Edit className="w-3 h-3" /> Edit
                                </button>
                                <button
                                    onClick={() => handleDelete(pkg.id)}
                                    className="flex-1 py-2 bg-rose-500/10 hover:bg-rose-500 text-rose-400 hover:text-white rounded-xl text-xs font-bold transition-colors flex items-center justify-center gap-2"
                                >
                                    <Trash2 className="w-3 h-3" /> Delete
                                </button>
                            </div>
                        </div>
                    ))
                )}
            </div>

            {/* Add/Edit Modal */}
            {showModal && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4 overflow-y-auto">
                    <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8 max-w-3xl w-full my-8 animate-in fade-in zoom-in-95">
                        <div className="flex items-center justify-between mb-6">
                            <div className="flex items-center gap-3">
                                <div className="w-12 h-12 rounded-2xl bg-rose-500/10 flex items-center justify-center">
                                    <Package className="w-6 h-6 text-rose-500" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-white">
                                        {editingPackage ? 'Edit Package' : 'Add New Package'}
                                    </h3>
                                    <p className="text-xs text-slate-500">Configure subscription package</p>
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
                                <div className="col-span-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Package Name
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="e.g., Basic Plan"
                                        required
                                    />
                                </div>

                                <div className="col-span-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Description
                                    </label>
                                    <textarea
                                        value={formData.description}
                                        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500 resize-none"
                                        rows="2"
                                        placeholder="Brief description of the package"
                                    />
                                </div>

                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Monthly Fee ($)
                                    </label>
                                    <input
                                        type="number"
                                        step="0.01"
                                        value={formData.monthly_fee}
                                        onChange={(e) => setFormData({ ...formData, monthly_fee: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="0.00"
                                        required
                                    />
                                </div>

                                <div>
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Messages Included
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.messages_included}
                                        onChange={(e) => setFormData({ ...formData, messages_included: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                        placeholder="Leave empty for unlimited"
                                    />
                                </div>

                                <div className="col-span-2">
                                    <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                        Billing Cycle
                                    </label>
                                    <select
                                        value={formData.billing_cycle}
                                        onChange={(e) => setFormData({ ...formData, billing_cycle: e.target.value })}
                                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                    >
                                        <option value="monthly">Monthly</option>
                                        <option value="yearly">Yearly</option>
                                    </select>
                                </div>
                            </div>

                            {/* Rollover Settings */}
                            <div className="p-4 bg-slate-950 rounded-xl border border-slate-800 space-y-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <input
                                            type="checkbox"
                                            id="rollover_enabled"
                                            checked={formData.rollover_enabled}
                                            onChange={(e) => setFormData({ ...formData, rollover_enabled: e.target.checked })}
                                            className="w-5 h-5 rounded bg-slate-800 border-slate-700 text-rose-500 focus:ring-rose-500"
                                        />
                                        <label htmlFor="rollover_enabled" className="text-sm text-white font-medium cursor-pointer">
                                            Enable Message Rollover
                                        </label>
                                    </div>
                                </div>

                                {formData.rollover_enabled && (
                                    <>
                                        <div className="grid grid-cols-2 gap-4 pl-8">
                                            <div>
                                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                                    Rollover Limit
                                                </label>
                                                <input
                                                    type="number"
                                                    value={formData.rollover_limit}
                                                    onChange={(e) => setFormData({ ...formData, rollover_limit: e.target.value })}
                                                    className="w-full bg-slate-900 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                                    placeholder="Leave empty for no limit"
                                                />
                                            </div>

                                            <div>
                                                <label className="text-xs font-bold text-slate-500 uppercase block mb-2">
                                                    Expiry (Months)
                                                </label>
                                                <input
                                                    type="number"
                                                    value={formData.rollover_expiry_months}
                                                    onChange={(e) => setFormData({ ...formData, rollover_expiry_months: e.target.value })}
                                                    className="w-full bg-slate-900 border border-slate-800 rounded-xl px-4 py-3 text-white focus:outline-none focus:border-rose-500"
                                                    placeholder="1"
                                                />
                                            </div>
                                        </div>

                                        {/* Conditional Rollover Policy */}
                                        <div className="pl-8 pt-4 border-t border-slate-800">
                                            <div className="flex items-start gap-3 p-4 bg-blue-500/5 rounded-xl border border-blue-500/20">
                                                <div className="flex-1">
                                                    <div className="flex items-center justify-between mb-2">
                                                        <div className="flex items-center gap-2">
                                                            {formData.conditional_rollover_enabled ? (
                                                                <ToggleRight className="w-6 h-6 text-blue-400" />
                                                            ) : (
                                                                <ToggleLeft className="w-6 h-6 text-slate-500" />
                                                            )}
                                                            <span className="text-sm font-bold text-white">
                                                                Conditional Rollover Policy
                                                            </span>
                                                        </div>
                                                        <button
                                                            type="button"
                                                            onClick={() => setFormData({
                                                                ...formData,
                                                                conditional_rollover_enabled: !formData.conditional_rollover_enabled
                                                            })}
                                                            className={`px-3 py-1 rounded-lg text-xs font-bold transition-colors ${formData.conditional_rollover_enabled
                                                                ? 'bg-blue-500 text-white'
                                                                : 'bg-slate-800 text-slate-400 hover:bg-slate-700'
                                                                }`}
                                                        >
                                                            {formData.conditional_rollover_enabled ? 'Enabled' : 'Disabled'}
                                                        </button>
                                                    </div>
                                                    <p className="text-xs text-slate-400 leading-relaxed">
                                                        When enabled, rollover messages are only activated upon successful automatic payment.
                                                        If payment fails, messages are forfeited to prevent losses.
                                                    </p>
                                                </div>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>

                            <div className="flex items-center gap-3 p-4 bg-slate-950 rounded-xl border border-slate-800">
                                <input
                                    type="checkbox"
                                    id="is_active"
                                    checked={formData.is_active}
                                    onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                                    className="w-5 h-5 rounded bg-slate-800 border-slate-700 text-rose-500 focus:ring-rose-500"
                                />
                                <label htmlFor="is_active" className="text-sm text-white font-medium cursor-pointer">
                                    Active (Make this package available for subscription)
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
                                    {editingPackage ? 'Update Package' : 'Create Package'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}
