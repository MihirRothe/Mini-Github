import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from './context/AuthContext';
import { AppLayout } from './components/layout/AppLayout';
import { DashboardPage } from './pages/DashboardPage';
import { ExplorePage } from './pages/ExplorePage';
import { SystemStatusPage } from './pages/SystemStatusPage';
import { LoginPage } from './pages/auth/LoginPage';
import { RegisterPage } from './pages/auth/RegisterPage';
import { UserProfilePage } from './pages/user/UserProfilePage';
import { ProfileSettingsPage } from './pages/settings/ProfileSettingsPage';
import { TokensPage } from './pages/settings/TokensPage';
import { NewOrgPage } from './pages/org/NewOrgPage';
import { OrgOverviewPage } from './pages/org/OrgOverviewPage';
import { TeamDetailPage } from './pages/org/TeamDetailPage';
import { NewRepoPage } from './pages/repo/NewRepoPage';
import { RepoOverviewPage } from './pages/repo/RepoOverviewPage';
import { BlobViewPage } from './pages/repo/BlobViewPage';
import { IssuesListPage } from './pages/issues/IssuesListPage';
import { NewIssuePage } from './pages/issues/NewIssuePage';
import { IssueDetailPage } from './pages/issues/IssueDetailPage';
import { NotFoundPage } from './pages/NotFoundPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      staleTime: 5000,
    },
  },
});

export const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<DashboardPage />} />
              <Route path="/explore" element={<ExplorePage />} />
              <Route path="/status" element={<SystemStatusPage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route path="/settings/profile" element={<ProfileSettingsPage />} />
              <Route path="/settings/tokens" element={<TokensPage />} />

              {/* Organization & Team Routes (Phase 3) */}
              <Route path="/orgs/new" element={<NewOrgPage />} />
              <Route path="/orgs/:org" element={<OrgOverviewPage />} />
              <Route path="/orgs/:org/teams/:team" element={<TeamDetailPage />} />

              {/* Repository Creation & Browsing Routes (Phase 4) */}
              <Route path="/new" element={<NewRepoPage />} />
              <Route path="/:owner/:repo/blob/:ref/*" element={<BlobViewPage />} />
              <Route path="/:owner/:repo/tree/:ref/*" element={<RepoOverviewPage />} />
              <Route path="/:owner/:repo/tree/:ref" element={<RepoOverviewPage />} />

              {/* Issues & Collaboration Routes (Phase 5) */}
              <Route path="/:owner/:repo/issues" element={<IssuesListPage />} />
              <Route path="/:owner/:repo/issues/new" element={<NewIssuePage />} />
              <Route path="/:owner/:repo/issues/:number" element={<IssueDetailPage />} />

              <Route path="/:owner/:repo" element={<RepoOverviewPage />} />

              <Route path="/:username" element={<UserProfilePage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
};

export default App;
