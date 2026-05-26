import { FormEvent, useEffect, useState } from 'react';
import QRCode from 'qrcode';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { useI18n } from '../../shared/i18n';
import type { User } from '../../shared/types';

type ActionState = 'idle' | 'loading' | 'saving';

export function SecurityPage({ user, onUserChange }: { user: User; onUserChange: (user: User | null) => void }) {
  const { t } = useI18n();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [disablePassword, setDisablePassword] = useState('');
  const [secret, setSecret] = useState('');
  const [otpauthUri, setOtpauthUri] = useState('');
  const [qrCode, setQrCode] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [message, setMessage] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [twoFactorError, setTwoFactorError] = useState('');
  const [actionState, setActionState] = useState<ActionState>('idle');

  useEffect(() => {
    if (!otpauthUri) {
      setQrCode('');
      return;
    }
    let active = true;
    void QRCode.toDataURL(otpauthUri, {
      errorCorrectionLevel: 'M',
      margin: 1,
      width: 220,
      color: { dark: '#172033', light: '#ffffff' }
    }).then((url) => {
      if (active) {
        setQrCode(url);
      }
    }).catch(() => {
      if (active) {
        setTwoFactorError(t('security.qrFailed'));
      }
    });
    return () => {
      active = false;
    };
  }, [otpauthUri, t]);

  async function changePassword(event: FormEvent) {
    event.preventDefault();
    setPasswordError('');
    setMessage('');
    setActionState('saving');
    try {
      await api.changePassword({ currentPassword, newPassword });
      onUserChange(null);
    } catch (err) {
      setPasswordError(errorMessage(err, t('security.passwordChangeFailed')));
    } finally {
      setActionState('idle');
    }
  }

  async function start2FA() {
    setTwoFactorError('');
    setMessage('');
    setBackupCodes([]);
    setActionState('loading');
    try {
      const result = await api.twoFactorSetup();
      setSecret(result.secret);
      setOtpauthUri(result.otpauthUri);
    } catch (err) {
      setTwoFactorError(errorMessage(err, t('security.twoFactorSetupFailed')));
    } finally {
      setActionState('idle');
    }
  }

  async function enable2FA(event: FormEvent) {
    event.preventDefault();
    setTwoFactorError('');
    setMessage('');
    setActionState('saving');
    try {
      const result = await api.twoFactorEnable(totpCode);
      setBackupCodes(result.backupCodes);
      setMessage(t('security.twoFactorEnabled'));
      setSecret('');
      setOtpauthUri('');
      setTotpCode('');
    } catch (err) {
      setTwoFactorError(errorMessage(err, t('security.twoFactorEnableFailed')));
    } finally {
      setActionState('idle');
    }
  }

  async function disable2FA(event: FormEvent) {
    event.preventDefault();
    setTwoFactorError('');
    setMessage('');
    setActionState('saving');
    try {
      await api.twoFactorDisable(disablePassword);
      onUserChange(null);
    } catch (err) {
      setTwoFactorError(errorMessage(err, t('security.twoFactorDisableFailed')));
    } finally {
      setActionState('idle');
    }
  }

  async function copyBackupCodes() {
    if (!backupCodes.length) {
      return;
    }
    try {
      await navigator.clipboard.writeText(backupCodes.join('\n'));
      setMessage(t('security.backupCodesCopied'));
    } catch {
      setTwoFactorError(t('security.backupCodesCopyFailed'));
    }
  }

  function downloadBackupCodes() {
    if (!backupCodes.length) {
      return;
    }
    const blob = new Blob([backupCodes.join('\n')], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'calendaradvanced-2fa-backup-codes.txt';
    link.click();
    URL.revokeObjectURL(url);
    setMessage(t('security.backupCodesDownloaded'));
  }

  return (
    <div className="page">
      <header className="page-header"><h1>{t('security.title')}</h1></header>
      <div className="two-column">
        <Card>
          <form className="stack" onSubmit={(event) => void changePassword(event)}>
            <h2>{t('security.passwordChange')}</h2>
            {passwordError && <p className="error" role="alert">{passwordError}</p>}
            <FormField label={t('security.currentPassword')} name="currentPassword" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.currentTarget.value)} />
            <FormField label={t('reset.newPassword')} name="newPassword" type="password" value={newPassword} onChange={(event) => setNewPassword(event.currentTarget.value)} />
            <Button disabled={actionState === 'saving'} type="submit">{t('common.save')}</Button>
          </form>
        </Card>
        <Card>
          <div className="stack">
            <h2>{t('security.twoFactor')}</h2>
            {message && <p className="success">{message}</p>}
            {twoFactorError && <p className="error" role="alert">{twoFactorError}</p>}
            <p>{user.twoFactorEnabled ? t('security.twoFactorActive') : t('security.twoFactorInactive')}</p>
            {backupCodes.length > 0 ? (
              <div className="two-factor-recovery" aria-live="polite">
                <h3>{t('security.backupCodes')}</h3>
                <p>{t('security.backupCodesHint')}</p>
                <div className="backup-code-grid">
                  {backupCodes.map((code) => <code key={code}>{code}</code>)}
                </div>
                <div className="button-row">
                  <Button onClick={() => void copyBackupCodes()} type="button" variant="ghost">{t('security.copyBackupCodes')}</Button>
                  <Button onClick={downloadBackupCodes} type="button" variant="ghost">{t('security.downloadBackupCodes')}</Button>
                  <Button onClick={() => onUserChange(null)} type="button">{t('security.continueToLogin')}</Button>
                </div>
              </div>
            ) : user.twoFactorEnabled ? (
              <form className="stack" onSubmit={(event) => void disable2FA(event)}>
                <FormField label={t('common.password')} name="disablePassword" type="password" value={disablePassword} onChange={(event) => setDisablePassword(event.currentTarget.value)} required />
                <Button disabled={actionState === 'saving'} variant="danger" type="submit">{t('security.disable2fa')}</Button>
              </form>
            ) : secret ? (
              <form className="stack" onSubmit={(event) => void enable2FA(event)}>
                <div className="two-factor-setup">
                  {qrCode ? <img alt={t('security.qrAlt')} src={qrCode} /> : <div className="two-factor-setup__qr-placeholder">{t('common.loading')}</div>}
                  <div className="stack stack--tight">
                    <p>{t('security.scanQr')}</p>
                    <code className="secret">{secret}</code>
                  </div>
                </div>
                <details className="two-factor-uri">
                  <summary>{t('security.manualSetup')}</summary>
                  <code className="secret">{otpauthUri}</code>
                </details>
                <FormField label={t('login.totp')} name="totpCode" inputMode="numeric" autoComplete="one-time-code" value={totpCode} onChange={(event) => setTotpCode(event.currentTarget.value)} required />
                <div className="button-row">
                  <Button disabled={actionState === 'saving'} type="submit">{t('security.confirm2fa')}</Button>
                  <Button disabled={actionState === 'saving'} onClick={() => {
                    setSecret('');
                    setOtpauthUri('');
                    setTotpCode('');
                  }} type="button" variant="ghost">{t('common.cancel')}</Button>
                </div>
              </form>
            ) : (
              <Button disabled={actionState === 'loading'} onClick={() => void start2FA()} type="button">{t('security.enable2fa')}</Button>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}

function errorMessage(error: unknown, fallback: string): string {
  return (error as { message?: string } | undefined)?.message ?? fallback;
}
