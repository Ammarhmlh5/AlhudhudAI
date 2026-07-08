import React from 'react';

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError(error) {
        return { hasError: true };
    }

    componentDidCatch(error, errorInfo) {
        console.error('UI Rendering Error Catch:', error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-6 text-center">
                    <div className="w-20 h-20 bg-rose-500/10 rounded-full flex items-center justify-center mb-6">
                        <svg className="w-10 h-10 text-rose-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                    </div>
                    <h1 className="text-3xl font-bold text-white mb-4">حدث خطأ ما</h1>
                    <p className="text-slate-400 max-w-md mb-8">
                        واجهنا خطأً غير متوقع أثناء عرض هذه الصفحة. عادةً ما يكون هذا بسبب نقص البيانات من الخادم.
                    </p>
                    <button
                        onClick={() => window.location.reload()}
                        className="px-8 py-3 bg-rose-600 hover:bg-rose-500 text-white font-bold rounded-xl transition-all shadow-lg shadow-rose-900/40"
                    >
                        إعادة تحميل الصفحة
                    </button>
                    <button
                        onClick={() => this.setState({ hasError: false })}
                        className="mt-4 text-slate-500 hover:text-slate-300 text-sm underline"
                    >
                        حاول مرة أخرى
                    </button>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;
