import React from 'react';
import { Link } from 'react-router-dom';
import { HelpCircle, ArrowLeft } from 'lucide-react';

export const NotFoundPage: React.FC = () => {
  return (
    <div className="py-20 text-center space-y-4">
      <div className="w-16 h-16 rounded-2xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
        <HelpCircle className="w-8 h-8" />
      </div>
      <h1 className="text-2xl font-bold text-white">404 — Page Not Found</h1>
      <p className="text-sm text-forge-muted max-w-sm mx-auto">
        The repository, branch, or page you are looking for does not exist or may have been transferred.
      </p>
      <div className="pt-2">
        <Link
          to="/dashboard"
          className="inline-flex items-center space-x-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/20 transition-all"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Dashboard</span>
        </Link>
      </div>
    </div>
  );
};
