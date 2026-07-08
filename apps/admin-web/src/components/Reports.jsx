import { useState, useEffect } from 'react';
import {
  TrendingUp, DollarSign, Users, Package, Download, Calendar,
  BarChart3, PieChart, Activity, FileText, ArrowUpRight, ArrowDownRight
} from 'lucide-react';
import apiClient from '../api/axios';
import { API_BASE_URL } from '../config';

export default function Reports() {
  const [loading, setLoading] = useState(true);
  const [dateRange, setDateRange] = useState('month');
  const [reports, setReports] = useState({
    revenue: {
      total: 0,
      subscriptions: 0,
      overage: 0,
      gateways: 0,
      topups: 0
    },
    costs: {
      total: 0,
      messages: 0,
      infrastructure: 0
    },
    customers: {
      total: 0,
      byPackage: []
    },
    subscriptions: {
      renewals: 0,
      cancellations: 0,
      renewalRate: 0
    }
  });

  useEffect(() => {
    fetchReports();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dateRange]);

  const fetchReports = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get(`/api/v1/admin/reports?range=${dateRange}`);
      setReports(res.data);
    } catch (err) {
      console.error("Failed to fetch reports", err);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = (format) => {
    window.open(
      `${API_BASE_URL}/api/v1/admin/reports/export?format=${format}&range=${dateRange}`,
      '_blank'
    );
  };

  const profitMargin = reports.revenue.total - reports.costs.total;
  const profitPercentage = reports.revenue.total > 0
    ? ((profitMargin / reports.revenue.total) * 100).toFixed(1)
    : 0;

  if (loading) {
    return (
      <div className="space-y-8">
        <div className="text-center py-12 text-slate-500">Loading reports...</div>
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-black text-white mb-2">Financial Reports</h1>
          <p className="text-slate-400">Comprehensive financial analytics and insights</p>
        </div>
        <div className="flex items-center gap-4">
          {/* Date Range Selector */}
          <select
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value)}
            className="px-4 py-2 bg-slate-900 border border-slate-800 rounded-xl text-white font-bold focus:outline-none focus:border-rose-500"
          >
            <option value="week">This Week</option>
            <option value="month">This Month</option>
            <option value="quarter">This Quarter</option>
            <option value="year">This Year</option>
          </select>

          {/* Export Buttons */}
          <div className="flex gap-2">
            <button
              onClick={() => handleExport('pdf')}
              className="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white rounded-xl font-bold flex items-center gap-2 transition-colors"
            >
              <Download className="w-4 h-4" />
              PDF
            </button>
            <button
              onClick={() => handleExport('excel')}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl font-bold flex items-center gap-2 transition-colors"
            >
              <Download className="w-4 h-4" />
              Excel
            </button>
            <button
              onClick={() => handleExport('csv')}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-bold flex items-center gap-2 transition-colors"
            >
              <Download className="w-4 h-4" />
              CSV
            </button>
          </div>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <SummaryCard
          icon={DollarSign}
          label="Total Revenue"
          value={`$${reports.revenue.total.toFixed(2)}`}
          color="from-emerald-500 to-teal-500"
          trend="+12.5%"
          trendUp={true}
        />
        <SummaryCard
          icon={TrendingUp}
          label="Total Costs"
          value={`$${reports.costs.total.toFixed(2)}`}
          color="from-rose-500 to-pink-500"
          trend="+5.2%"
          trendUp={true}
        />
        <SummaryCard
          icon={Activity}
          label="Profit Margin"
          value={`$${profitMargin.toFixed(2)}`}
          color="from-blue-500 to-cyan-500"
          trend={`${profitPercentage}%`}
          trendUp={profitMargin > 0}
        />
        <SummaryCard
          icon={Users}
          label="Total Customers"
          value={reports.customers.total}
          color="from-amber-500 to-orange-500"
          trend="+8.3%"
          trendUp={true}
        />
      </div>

      {/* Revenue Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 flex items-center justify-center">
              <DollarSign className="w-6 h-6 text-emerald-500" />
            </div>
            <div>
              <h2 className="text-xl font-black text-white">Revenue Breakdown</h2>
              <p className="text-xs text-slate-500">Income by source</p>
            </div>
          </div>

          <div className="space-y-4">
            <RevenueItem
              label="Subscriptions"
              amount={reports.revenue.subscriptions}
              percentage={(reports.revenue.subscriptions / reports.revenue.total * 100).toFixed(1)}
              color="bg-emerald-500"
            />
            <RevenueItem
              label="Overage Usage"
              amount={reports.revenue.overage}
              percentage={(reports.revenue.overage / reports.revenue.total * 100).toFixed(1)}
              color="bg-blue-500"
            />
            <RevenueItem
              label="Gateway Rentals"
              amount={reports.revenue.gateways}
              percentage={(reports.revenue.gateways / reports.revenue.total * 100).toFixed(1)}
              color="bg-purple-500"
            />
            <RevenueItem
              label="Direct Top-ups"
              amount={reports.revenue.topups}
              percentage={(reports.revenue.topups / reports.revenue.total * 100).toFixed(1)}
              color="bg-amber-500"
            />
          </div>

          {/* Simple Bar Chart Visualization */}
          <div className="mt-6 pt-6 border-t border-slate-800">
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <div className="text-xs text-slate-500 w-24">Subscriptions</div>
                <div className="flex-1 h-8 bg-slate-950 rounded-lg overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-emerald-500 to-emerald-600 flex items-center justify-end px-2"
                    style={{ width: `${(reports.revenue.subscriptions / reports.revenue.total * 100)}%` }}
                  >
                    <span className="text-xs font-bold text-white">
                      {(reports.revenue.subscriptions / reports.revenue.total * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="text-xs text-slate-500 w-24">Overage</div>
                <div className="flex-1 h-8 bg-slate-950 rounded-lg overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-500 to-blue-600 flex items-center justify-end px-2"
                    style={{ width: `${(reports.revenue.overage / reports.revenue.total * 100)}%` }}
                  >
                    <span className="text-xs font-bold text-white">
                      {(reports.revenue.overage / reports.revenue.total * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="text-xs text-slate-500 w-24">Gateways</div>
                <div className="flex-1 h-8 bg-slate-950 rounded-lg overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-purple-500 to-purple-600 flex items-center justify-end px-2"
                    style={{ width: `${(reports.revenue.gateways / reports.revenue.total * 100)}%` }}
                  >
                    <span className="text-xs font-bold text-white">
                      {(reports.revenue.gateways / reports.revenue.total * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Cost Breakdown */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-12 h-12 rounded-2xl bg-rose-500/10 flex items-center justify-center">
              <TrendingUp className="w-6 h-6 text-rose-500" />
            </div>
            <div>
              <h2 className="text-xl font-black text-white">Cost Breakdown</h2>
              <p className="text-xs text-slate-500">Expenses by category</p>
            </div>
          </div>

          <div className="space-y-4">
            <CostItem
              label="Message Costs"
              amount={reports.costs.messages}
              percentage={(reports.costs.messages / reports.costs.total * 100).toFixed(1)}
              color="bg-rose-500"
            />
            <CostItem
              label="Infrastructure"
              amount={reports.costs.infrastructure}
              percentage={(reports.costs.infrastructure / reports.costs.total * 100).toFixed(1)}
              color="bg-orange-500"
            />
          </div>

          {/* Profit Summary */}
          <div className="mt-6 pt-6 border-t border-slate-800">
            <div className="flex items-center justify-between mb-4">
              <span className="text-slate-400 font-bold">Net Profit</span>
              <span className={`text-2xl font-black ${profitMargin > 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                ${profitMargin.toFixed(2)}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400 font-bold">Profit Margin</span>
              <span className={`text-xl font-black ${profitMargin > 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                {profitPercentage}%
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Customer Distribution & Subscription Stats */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Customer Distribution by Package */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center">
              <Package className="w-6 h-6 text-blue-500" />
            </div>
            <div>
              <h2 className="text-xl font-black text-white">Customer Distribution</h2>
              <p className="text-xs text-slate-500">By package type</p>
            </div>
          </div>

          <div className="space-y-3">
            {reports.customers.byPackage.map((pkg, index) => (
              <div
                key={index}
                className="bg-slate-950 border border-slate-800 rounded-2xl p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
                    <Package className="w-5 h-5 text-blue-500" />
                  </div>
                  <div>
                    <div className="font-bold text-white">{pkg.name}</div>
                    <div className="text-xs text-slate-500">{pkg.count} customers</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-mono text-blue-400 font-bold">
                    {((pkg.count / reports.customers.total) * 100).toFixed(1)}%
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Subscription Statistics */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-8">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-12 h-12 rounded-2xl bg-purple-500/10 flex items-center justify-center">
              <Activity className="w-6 h-6 text-purple-500" />
            </div>
            <div>
              <h2 className="text-xl font-black text-white">Subscription Stats</h2>
              <p className="text-xs text-slate-500">Renewals and cancellations</p>
            </div>
          </div>

          <div className="space-y-6">
            <div className="bg-slate-950 border border-slate-800 rounded-2xl p-6">
              <div className="flex items-center justify-between mb-2">
                <span className="text-slate-400 font-bold">Renewals</span>
                <div className="flex items-center gap-2 text-emerald-400">
                  <ArrowUpRight className="w-4 h-4" />
                  <span className="font-bold">{reports.subscriptions.renewals}</span>
                </div>
              </div>
              <div className="text-2xl font-black text-white">{reports.subscriptions.renewalRate}%</div>
              <div className="text-xs text-slate-500 mt-1">Renewal rate</div>
            </div>

            <div className="bg-slate-950 border border-slate-800 rounded-2xl p-6">
              <div className="flex items-center justify-between mb-2">
                <span className="text-slate-400 font-bold">Cancellations</span>
                <div className="flex items-center gap-2 text-rose-400">
                  <ArrowDownRight className="w-4 h-4" />
                  <span className="font-bold">{reports.subscriptions.cancellations}</span>
                </div>
              </div>
              <div className="text-2xl font-black text-white">
                {(100 - reports.subscriptions.renewalRate).toFixed(1)}%
              </div>
              <div className="text-xs text-slate-500 mt-1">Cancellation rate</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SummaryCard({ icon: Icon, label, value, color, trend, trendUp }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 group hover:border-slate-700 transition-all">
      <div className="flex items-center gap-4 mb-4">
        <div className={`w-14 h-14 rounded-2xl bg-gradient-to-br ${color} flex items-center justify-center shadow-lg`}>
          <Icon className="w-7 h-7 text-white" />
        </div>
        <div className="flex-1">
          <div className="text-sm text-slate-500 font-medium mb-1">{label}</div>
          <div className="text-2xl font-black text-white">{value}</div>
        </div>
      </div>
      {trend && (
        <div className={`flex items-center gap-1 text-sm font-bold ${trendUp ? 'text-emerald-400' : 'text-rose-400'}`}>
          {trendUp ? <ArrowUpRight className="w-4 h-4" /> : <ArrowDownRight className="w-4 h-4" />}
          {trend}
        </div>
      )}
    </div>
  );
}

function RevenueItem({ label, amount, percentage, color }) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <div className={`w-3 h-3 rounded-full ${color}`} />
        <span className="text-slate-400 font-medium">{label}</span>
      </div>
      <div className="text-right">
        <div className="font-mono text-white font-bold">${amount.toFixed(2)}</div>
        <div className="text-xs text-slate-500">{percentage}%</div>
      </div>
    </div>
  );
}

function CostItem({ label, amount, percentage, color }) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <div className={`w-3 h-3 rounded-full ${color}`} />
        <span className="text-slate-400 font-medium">{label}</span>
      </div>
      <div className="text-right">
        <div className="font-mono text-white font-bold">${amount.toFixed(2)}</div>
        <div className="text-xs text-slate-500">{percentage}%</div>
      </div>
    </div>
  );
}
