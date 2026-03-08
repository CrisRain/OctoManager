import { Suspense, lazy, type ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { AppShell } from "@/components/app-shell";
import { useAuthCheck } from "@/hooks/use-auth-check";

const DashboardPage = lazy(() =>
  import("@/pages/dashboard-page").then((module) => ({ default: module.DashboardPage }))
);
const AccountTypesPage = lazy(() =>
  import("@/pages/account-types-page").then((module) => ({ default: module.AccountTypesPage }))
);
const AccountsPage = lazy(() =>
  import("@/pages/accounts-page").then((module) => ({ default: module.AccountsPage }))
);
const EmailAccountsOutlookPage = lazy(() =>
  import("@/pages/email-accounts-outlook-page").then((module) => ({
    default: module.EmailAccountsOutlookPage,
  }))
);
const JobsPage = lazy(() =>
  import("@/pages/jobs-page").then((module) => ({ default: module.JobsPage }))
);
const OctoModulesPage = lazy(() =>
  import("@/pages/octo-modules-page").then((module) => ({ default: module.OctoModulesPage }))
);
const ApiKeysPage = lazy(() =>
  import("@/pages/api-keys-page").then((module) => ({ default: module.ApiKeysPage }))
);
const TriggersPage = lazy(() =>
  import("@/pages/triggers-page").then((module) => ({ default: module.TriggersPage }))
);
const SettingsPage = lazy(() =>
  import("@/pages/settings-page").then((module) => ({ default: module.SettingsPage }))
);
const SSLCertificatePage = lazy(() =>
  import("@/pages/ssl-certificate-page").then((module) => ({ default: module.SSLCertificatePage }))
);
const OAuthCallbackPage = lazy(() =>
  import("@/pages/oauth-callback-page").then((module) => ({ default: module.OAuthCallbackPage }))
);
const SetupPage = lazy(() =>
  import("@/pages/setup-page").then((module) => ({ default: module.SetupPage }))
);
const AuthPage = lazy(() =>
  import("@/pages/auth-page").then((module) => ({ default: module.AuthPage }))
);

function RouteFallback() {
  return (
    <div className="flex min-h-[240px] items-center justify-center rounded-xl border border-dashed bg-card/60">
      <div className="text-center">
        <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/20 border-t-foreground" />
        <p className="mt-3 text-sm text-muted-foreground">页面加载中...</p>
      </div>
    </div>
  );
}

function withRouteSuspense(element: ReactNode) {
  return <Suspense fallback={<RouteFallback />}>{element}</Suspense>;
}

function RequireAuth({ children }: { children: ReactNode }) {
  const location = useLocation();
  const authState = useAuthCheck();

  if (authState === "checking") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/20 border-t-foreground" />
      </div>
    );
  }
  if (authState === "unauthenticated") {
    return <Navigate to="/auth" state={{ from: location.pathname }} replace />;
  }
  if (authState === "needs-setup") {
    return <Navigate to="/setup" replace />;
  }
  return <>{children}</>;
}

// 仅在系统尚未初始化时允许访问，否则跳走
function RequireSetup({ children }: { children: ReactNode }) {
  const authState = useAuthCheck();

  if (authState === "checking") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/20 border-t-foreground" />
      </div>
    );
  }
  if (authState === "needs-setup") {
    return <>{children}</>;
  }
  // 已初始化：已登录跳 dashboard，未登录跳 auth
  return <Navigate to={authState === "ok" ? "/dashboard" : "/auth"} replace />;
}

export function App() {
  return (
    <Routes>
      <Route path="/oauth/callback" element={withRouteSuspense(<OAuthCallbackPage />)} />
      <Route path="/setup" element={<RequireSetup>{withRouteSuspense(<SetupPage />)}</RequireSetup>} />
      <Route path="/auth" element={withRouteSuspense(<AuthPage />)} />
      <Route
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={withRouteSuspense(<DashboardPage />)} />
        <Route path="/account-types" element={withRouteSuspense(<AccountTypesPage />)} />
        <Route path="/accounts" element={withRouteSuspense(<AccountsPage />)} />
        <Route path="/accounts/:typeKey" element={withRouteSuspense(<AccountsPage />)} />
        <Route path="/email-accounts" element={<Navigate to="/email-accounts/outlook" replace />} />
        <Route path="/email-accounts/outlook" element={withRouteSuspense(<EmailAccountsOutlookPage />)} />
        <Route path="/jobs" element={withRouteSuspense(<JobsPage />)} />
        <Route path="/modules" element={withRouteSuspense(<OctoModulesPage />)} />
        <Route path="/api-keys" element={withRouteSuspense(<ApiKeysPage />)} />
        <Route path="/triggers" element={withRouteSuspense(<TriggersPage />)} />
        <Route path="/settings" element={withRouteSuspense(<SettingsPage />)} />
        <Route path="/ssl" element={withRouteSuspense(<SSLCertificatePage />)} />
      </Route>
    </Routes>
  );
}
