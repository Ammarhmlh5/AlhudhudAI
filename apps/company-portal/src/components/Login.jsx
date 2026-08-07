import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Building2, LogIn } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await login(email, password);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err.response?.data?.error || 'فشل تسجيل الدخول. تحقق من البيانات وحاول مرة أخرى.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center p-8">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="p-6 bg-rose-500/10 rounded-[2rem] inline-block mb-6 border border-rose-500/20">
            <Building2 className="w-14 h-14 text-rose-500" />
          </div>
          <h1 className="text-3xl font-black text-white mb-2">بوابة الشركات</h1>
          <p className="text-slate-400">منصة إدارة خدمات الهدهد للشركات</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-slate-900 border border-slate-800 rounded-2xl p-8 space-y-5">
          <div>
            <label className="block text-sm font-semibold text-slate-300 mb-2">البريد الإلكتروني</label>
            <input
              type="email"
              required
              dir="ltr"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="company@example.com"
              className="w-full px-4 py-3 bg-slate-800 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-rose-500"
            />
          </div>
          <div>
            <label className="block text-sm font-semibold text-slate-300 mb-2">كلمة المرور</label>
            <input
              type="password"
              required
              dir="ltr"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="w-full px-4 py-3 bg-slate-800 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:border-rose-500"
            />
          </div>

          {error && (
            <div className="px-4 py-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-sm">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-rose-600 hover:bg-rose-500 disabled:opacity-50 rounded-xl text-white font-bold transition-colors"
          >
            <LogIn className="w-5 h-5" />
            {submitting ? 'جارٍ الدخول...' : 'تسجيل الدخول'}
          </button>
        </form>
      </div>
    </div>
  );
}
