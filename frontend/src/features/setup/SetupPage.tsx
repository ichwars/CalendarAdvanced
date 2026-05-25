import { FormEvent, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { useI18n } from '../../shared/i18n';

export function SetupPage({ onComplete }: { onComplete: () => void }) {
  const { t } = useI18n();
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [error, setError] = useState('');

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (password !== passwordConfirm) {
      setError(t('setup.passwordMismatch'));
      return;
    }
    try {
      await api.setupAdmin({ email, username, displayName, password });
      onComplete();
    } catch (err) {
      setError((err as { message?: string }).message ?? t('common.error'));
    }
  }

  return (
    <main className="center-screen">
      <Card>
        <form className="stack" onSubmit={(event) => void submit(event)}>
          <div>
            <h1>{t('setup.title')}</h1>
            <p>{t('setup.subtitle')}</p>
          </div>
          {error && <p className="error" role="alert">{error}</p>}
          <FormField label={t('common.email')} name="email" type="email" value={email} onChange={(event) => setEmail(event.currentTarget.value)} required />
          <FormField label={t('common.username')} name="username" value={username} onChange={(event) => setUsername(event.currentTarget.value)} required />
          <FormField label={t('common.displayName')} name="displayName" value={displayName} onChange={(event) => setDisplayName(event.currentTarget.value)} required />
          <FormField label={t('common.password')} name="password" type="password" value={password} onChange={(event) => setPassword(event.currentTarget.value)} required />
          <FormField label={t('common.passwordConfirm')} name="passwordConfirm" type="password" value={passwordConfirm} onChange={(event) => setPasswordConfirm(event.currentTarget.value)} required />
          <Button type="submit">{t('setup.submit')}</Button>
        </form>
      </Card>
    </main>
  );
}
