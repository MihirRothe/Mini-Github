import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { GitBranch, Lock, User, Eye, EyeOff, AlertCircle, ArrowRight, ShieldCheck } from 'lucide-react';

export const LoginPage: React.FC = () => {
  const [loginInput, setLoginInput] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!loginInput.trim() || !password) {
      setError('Please enter your username/email and password.');
      return;
    }

    try {
      setIsSubmitting(true);
      setError(null);
      await login(loginInput, password);
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Invalid username or password.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleFillDemo = () => {
    setLoginInput('admin');
    setPassword('ForgeHubAdmin123!');
    setError(null);
  };

  return (
    <div className="min-h-[80vh] flex flex-col items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Brand Header */}
        <div className="text-center space-y-2">
          <div className="w-12 h-12 rounded-xl bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center mx-auto shadow-xl shadow-blue-500/20">
            <GitBranch className="w-6 h-6 text-white" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Sign in to ForgeHub</h1>
          <p className="text-xs text-forge-muted">
            Enter your developer credentials to manage repositories and workflows.
          </p>
        </div>

        {/* Demo Credentials Helper Card */}
        <div className="p-3.5 bg-blue-500/10 border border-blue-500/20 rounded-xl flex items-center justify-between text-xs">
          <div className="flex items-center space-x-2 text-blue-400">
            <ShieldCheck className="w-4 h-4 shrink-0" />
            <span>Local Dev Admin Available</span>
          </div>
          <button
            type="button"
            onClick={handleFillDemo}
            className="px-2.5 py-1 bg-blue-600 hover:bg-blue-500 text-white rounded-md text-[11px] font-semibold transition-colors shadow-sm"
          >
            Fill Admin
          </button>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl flex items-start space-x-2.5 text-xs text-rose-400 animate-fade-in">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="glass-panel p-6 rounded-2xl space-y-4 shadow-xl">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-forge-text flex items-center gap-1.5">
              <User className="w-3.5 h-3.5 text-forge-muted" />
              <span>Username or Email</span>
            </label>
            <input
              type="text"
              autoFocus
              required
              value={loginInput}
              onChange={(e) => setLoginInput(e.target.value)}
              placeholder="e.g. alice or alice@example.com"
              className="w-full px-3.5 py-2 bg-forge-card border border-forge-border rounded-lg text-sm text-forge-text placeholder:text-forge-muted focus:outline-none focus:border-blue-500 transition-colors"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-forge-text flex items-center gap-1.5">
              <Lock className="w-3.5 h-3.5 text-forge-muted" />
              <span>Password</span>
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter password"
                className="w-full pl-3.5 pr-10 py-2 bg-forge-card border border-forge-border rounded-lg text-sm text-forge-text placeholder:text-forge-muted focus:outline-none focus:border-blue-500 transition-colors"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-2.5 text-forge-muted hover:text-forge-text"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full py-2.5 px-4 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-semibold shadow-lg shadow-blue-600/25 flex items-center justify-center space-x-2 transition-all"
          >
            <span>{isSubmitting ? 'Signing in...' : 'Sign In'}</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        </form>

        {/* Footer Link */}
        <div className="text-center text-xs text-forge-muted">
          New to ForgeHub?{' '}
          <Link to="/register" className="text-blue-400 hover:underline font-medium">
            Create an account
          </Link>
        </div>
      </div>
    </div>
  );
};
