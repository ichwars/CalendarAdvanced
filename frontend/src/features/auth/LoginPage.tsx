import { FormEvent, useState } from 'react';
import { api, isBackendUnavailable } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { useI18n } from '../../shared/i18n';
import type { User } from '../../shared/types';

export function LoginPage({ onLogin }: { onLogin: (user: User) => void }) {
  const { t } = useI18n();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [backupCode, setBackupCode] = useState('');
  const [twoFactorRequired, setTwoFactorRequired] = useState(false);
  const [showReset, setShowReset] = useState(false);
  const [resetToken, setResetToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  async function login(event: FormEvent) {
    event.preventDefault();
    setError('');
    try {
      const result = await api.login({ email, password, totpCode, backupCode });
      if (result.twoFactorRequired || !result.user) {
        setTwoFactorRequired(true);
        return;
      }
      onLogin(result.user);
    } catch (err) {
      const apiError = err as { code?: string; message?: string };
      if (apiError.code === 'two_factor_required') {
        setTwoFactorRequired(true);
      }
      if (isBackendUnavailable(err)) {
        setError(t('app.backendUnavailableText'));
        return;
      }
      setError(apiError.message ?? t('common.error'));
    }
  }

  async function requestReset() {
    setError('');
    const result = await api.requestReset(email);
    if (result.localToken) {
      setResetToken(result.localToken);
    }
    setMessage(t('common.ok'));
  }

  async function resetPassword() {
    setError('');
    await api.resetPassword(resetToken, newPassword);
    setMessage(t('common.ok'));
    setShowReset(false);
  }

  return (
    <main className="center-screen">
      <Card>
        <form className="stack" onSubmit={(event) => void login(event)}>
          <h1>{t('login.title')}</h1>
          {message && <p className="success" role="status">{message}</p>}
          {error && <p className="error" role="alert">{error}</p>}
          <FormField label={t('login.identifier')} name="email" value={email} onChange={(event) => setEmail(event.currentTarget.value)} required />
          <FormField label={t('common.password')} name="password" type="password" value={password} onChange={(event) => setPassword(event.currentTarget.value)} required />
          {twoFactorRequired && (
            <div className="stack stack--tight">
              <p>{t('login.twoFactorRequired')}</p>
              <FormField label={t('login.totp')} name="totpCode" inputMode="numeric" value={totpCode} onChange={(event) => setTotpCode(event.currentTarget.value)} />
              <FormField label={t('login.backupCode')} name="backupCode" value={backupCode} onChange={(event) => setBackupCode(event.currentTarget.value)} />
            </div>
          )}
          <Button type="submit">{t('login.submit')}</Button>
          <Button type="button" variant="ghost" onClick={() => setShowReset((value) => !value)}>{t('login.forgotPassword')}</Button>
        </form>
        {showReset && (
          <div className="stack separator">
            <h2>{t('reset.requestTitle')}</h2>
            <Button type="button" onClick={() => void requestReset()}>{t('reset.requestSubmit')}</Button>
            <FormField label={t('reset.token')} name="resetToken" value={resetToken} onChange={(event) => setResetToken(event.currentTarget.value)} />
            <FormField label={t('reset.newPassword')} name="newPassword" type="password" value={newPassword} onChange={(event) => setNewPassword(event.currentTarget.value)} />
            <Button type="button" onClick={() => void resetPassword()}>{t('common.save')}</Button>
          </div>
        )}
      </Card>
    </main>
  );
}
