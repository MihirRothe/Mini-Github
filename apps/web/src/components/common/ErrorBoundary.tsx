import { Component, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, Home, ChevronDown, ChevronUp } from 'lucide-react';

interface Props {
  children: ReactNode;
  fallbackTitle?: string;
  fallbackMessage?: string;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  showDetails: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
    showDetails: false,
  };

  public static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error caught by ErrorBoundary:', error, errorInfo);
    this.setState({ errorInfo });
  }

  private handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null, showDetails: false });
    window.location.href = '/dashboard';
  };

  private handleReload = () => {
    window.location.reload();
  };

  public render() {
    if (this.state.hasError) {
      const title = this.props.fallbackTitle || 'Something went wrong';
      const message =
        this.props.fallbackMessage ||
        'An unexpected error occurred while rendering this view. Your session and data are safe.';

      return (
        <div className="min-h-[400px] flex items-center justify-center p-6 select-none animate-fade-in">
          <div className="w-full max-w-xl glass-panel p-8 rounded-2xl border border-rose-500/30 bg-forge-surface/90 shadow-2xl space-y-6 text-center">
            <div className="w-16 h-16 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center mx-auto text-rose-400 shadow-lg shadow-rose-500/10">
              <AlertTriangle className="w-8 h-8" />
            </div>

            <div className="space-y-2">
              <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
              <p className="text-xs text-forge-muted max-w-md mx-auto leading-relaxed">
                {message}
              </p>
            </div>

            {this.state.error && (
              <div className="text-left">
                <button
                  onClick={() => this.setState((prev) => ({ showDetails: !prev.showDetails }))}
                  className="flex items-center space-x-1 text-xs text-forge-muted hover:text-white mx-auto transition-colors"
                >
                  <span>{this.state.showDetails ? 'Hide technical diagnostics' : 'Show technical diagnostics'}</span>
                  {this.state.showDetails ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                </button>

                {this.state.showDetails && (
                  <div className="mt-3 p-3.5 rounded-xl bg-black/60 border border-forge-border text-[11px] font-mono text-rose-300 overflow-x-auto space-y-2 max-h-48">
                    <div className="font-semibold">{this.state.error.name}: {this.state.error.message}</div>
                    {this.state.errorInfo?.componentStack && (
                      <pre className="text-[10px] text-forge-muted whitespace-pre-wrap leading-tight">
                        {this.state.errorInfo.componentStack}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            )}

            <div className="flex flex-wrap items-center justify-center gap-3 pt-2">
              <button
                onClick={this.handleReload}
                className="flex items-center space-x-1.5 px-4 py-2 rounded-xl bg-forge-card hover:bg-forge-subtle border border-forge-border text-xs font-semibold text-white transition-all shadow-sm"
              >
                <RefreshCw className="w-3.5 h-3.5 text-blue-400" />
                <span>Reload Page</span>
              </button>

              <button
                onClick={this.handleReset}
                className="flex items-center space-x-1.5 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-xs font-semibold text-white transition-all shadow-lg shadow-blue-600/20"
              >
                <Home className="w-3.5 h-3.5" />
                <span>Return to Dashboard</span>
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
