import { useEffect, useState } from 'react';
import { Shell } from './Shell';
import { LoginPage } from '../features/auth/LoginPage';
import { SetupPage } from '../features/setup/SetupPage';
import { api, isBackendUnavailable } from '../shared/api';
import { Button } from '../shared/components/Button';
import { Card } from '../shared/components/Card';
import { ScrollbarController } from '../shared/components/ScrollbarController';
import { setActiveLocale, useI18n } from '../shared/i18n';
import { getGeneralPreferences, normalizeGeneralPreferences, setGeneralPreferences } from '../shared/preferences';
import { setTheme } from '../shared/theme';
import type { User } from '../shared/types';

export function App() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [setupRequired, setSetupRequired] = useState(false);
  const [backendUnavailable, setBackendUnavailable] = useState(false);
  const [user, setUser] = useState<User | null>(null);

  const chrome = <ScrollbarController />;

  async function bootstrap() {
    setLoading(true);
    setBackendUnavailable(false);
    try {
      const setup = await api.setupStatus();
      setSetupRequired(setup.required);
      if (!setup.required) {
        try {
          const me = await api.me();
          setUser(me.user);
          await syncPreferences().catch(() => undefined);
        } catch {
          setUser(null);
        }
      }
    } catch (err) {
      if (isBackendUnavailable(err)) {
        setBackendUnavailable(true);
      }
      setSetupRequired(false);
      setUser(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void bootstrap();
  }, []);

  if (loading) {
    return <>{chrome}<main className="center-screen" aria-live="polite">{t('app.loading')}</main></>;
  }

  if (backendUnavailable) {
    return <>{chrome}<BackendUnavailable onRetry={() => void bootstrap()} /></>;
  }

  if (setupRequired) {
    return <>{chrome}<SetupPage onComplete={() => void bootstrap()} /></>;
  }

  if (!user) {
    return <>{chrome}<LoginPage onLogin={(nextUser) => {
      setUser(nextUser);
      void syncPreferences().catch(() => undefined);
    }} /></>;
  }

  return <>{chrome}<Shell user={user} onUserChange={setUser} /></>;
}

async function syncPreferences() {
  const result = await api.preferences();
  if (result.persisted) {
    applySyncedPreferences(result.preferences);
    return;
  }
  const saved = await api.savePreferences(getGeneralPreferences());
  applySyncedPreferences(saved.preferences);
}

function applySyncedPreferences(preferences: ReturnType<typeof getGeneralPreferences>) {
  const normalized = normalizeGeneralPreferences(preferences);
  setGeneralPreferences(normalized);
  setTheme(normalized.theme);
  setActiveLocale(normalized.locale);
}

function BackendUnavailable({ onRetry }: { onRetry: () => void }) {
  const { t } = useI18n();
  return (
    <main className="center-screen">
      <Card>
        <div className="stack">
          <div>
            <h1>{t('app.backendUnavailableTitle')}</h1>
            <p>{t('app.backendUnavailableText')}</p>
          </div>
          <code className="secret">docker compose up -d --build</code>
          <code className="secret">cd backend &amp;&amp; go run ./cmd/calendaradvanced</code>
          <Button type="button" onClick={onRetry}>{t('app.retryBackend')}</Button>
        </div>
      </Card>
    </main>
  );
}
