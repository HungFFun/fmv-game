// Root composer — MUI ThemeProvider/CssBaseline ngoài cùng, rồi
// NotificationProvider → AuthenticationProvider → GameProvider → <GameShell/>.
// Route admin (#admin) tách riêng, lazy-load (code-split) để KHÔNG phình bundle người chơi.
import { lazy, Suspense, useSyncExternalStore } from 'react';

// MUI
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';

// App
import theme from '@app/theme';
import GameShell from '@containers/game';
import GameProvider from '@containers/game/provider';

// Providers
import ProviderComposer from '@providers/composer';
import NotificationProvider from '@providers/notification';
import AuthenticationProvider from '@providers/authentication';

const AdminApp = lazy(() => import('@containers/admin'));

// hash hiện tại (không cần router lib) — dùng useSyncExternalStore cho gọn.
const subscribe = (cb) => {
  window.addEventListener('hashchange', cb);
  return () => window.removeEventListener('hashchange', cb);
};
const useHash = () => useSyncExternalStore(subscribe, () => window.location.hash);

const App = () => {
  const hash = useHash();
  if (hash.startsWith('#admin')) {
    return (
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Suspense fallback={<div style={{ color: '#8aa0c0', padding: 24, fontFamily: 'system-ui' }}>Đang tải editor…</div>}>
          <AdminApp />
        </Suspense>
      </ThemeProvider>
    );
  }
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ProviderComposer
        providers={[
          <NotificationProvider key="notification" />,
          <AuthenticationProvider key="authentication" />,
          <GameProvider key="game" />,
        ]}
      >
        <GameShell />
      </ProviderComposer>
    </ThemeProvider>
  );
};

export default App;
