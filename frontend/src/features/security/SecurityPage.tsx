import { FormEvent, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { useI18n } from '../../shared/i18n';
import type { User } from '../../shared/types';

export function SecurityPage({ user, onUserChange }: { user: User; onUserChange: (user: User | null) => void }) {
  const { t } = useI18n();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [disablePassword, setDisablePassword] = useState('');
  const [secret, setSecret] = useState('');
  const [otpauthUri, setOtpauthUri] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [message, setMessage] = useState('');

  async function changePassword(event: FormEvent) {
    event.preventDefault();
    await api.changePassword({ currentPassword, newPassword });
    onUserChange(null);
  }

  async function start2FA() {
    const result = await api.twoFactorSetup();
    setSecret(result.secret);
    setOtpauthUri(result.otpauthUri);
  }

  async function enable2FA() {
    const result = await api.twoFactorEnable(totpCode);
    setBackupCodes(result.backupCodes);
    setMessage(t('common.ok'));
    onUserChange(null);
  }

  async function disable2FA() {
    await api.twoFactorDisable(disablePassword);
    setMessage(t('common.ok'));
    onUserChange(null);
  }

  return (
    <div className="page">
      <header className="page-header"><h1>{t('security.title')}</h1></header>
      <div className="two-column">
        <Card>
          <form className="stack" onSubmit={(event) => void changePassword(event)}>
            <h2>{t('security.passwordChange')}</h2>
            <FormField label={t('security.currentPassword')} name="currentPassword" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.currentTarget.value)} />
            <FormField label={t('reset.newPassword')} name="newPassword" type="password" value={newPassword} onChange={(event) => setNewPassword(event.currentTarget.value)} />
            <Button type="submit">{t('common.save')}</Button>
          </form>
        </Card>
        <Card unfinished>
          <div className="stack">
            <h2>{t('security.twoFactor')}</h2>
            {message && <p className="success">{message}</p>}
            <p>{user.twoFactorEnabled ? t('security.disable2fa') : t('security.enable2fa')}</p>
            {!user.twoFactorEnabled && <Button onClick={() => void start2FA()}>{t('security.enable2fa')}</Button>}
            {secret && <code className="secret">{secret}</code>}
            {otpauthUri && <code className="secret">{otpauthUri}</code>}
            {secret && <FormField label={t('login.totp')} name="totpCode" value={totpCode} onChange={(event) => setTotpCode(event.currentTarget.value)} />}
            {secret && <Button onClick={() => void enable2FA()}>{t('common.save')}</Button>}
            {user.twoFactorEnabled && <FormField label={t('common.password')} name="disablePassword" type="password" value={disablePassword} onChange={(event) => setDisablePassword(event.currentTarget.value)} />}
            {user.twoFactorEnabled && <Button variant="danger" onClick={() => void disable2FA()}>{t('security.disable2fa')}</Button>}
            {backupCodes.length > 0 && <div><h3>{t('security.backupCodes')}</h3><ul>{backupCodes.map((code) => <li key={code}><code>{code}</code></li>)}</ul></div>}
          </div>
        </Card>
      </div>
    </div>
  );
}
