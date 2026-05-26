import { FormEvent, useEffect, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { EmptyState } from '../../shared/components/EmptyState';
import { FormField } from '../../shared/components/FormField';
import { useI18n } from '../../shared/i18n';
import type { RoleName, User } from '../../shared/types';

export function UsersPage() {
  const { t } = useI18n();
  const [users, setUsers] = useState<User[]>([]);
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [roles, setRoles] = useState<RoleName[]>(['viewer']);
  const [error, setError] = useState('');

  async function load() {
    const response = await api.users();
    setUsers(response.items);
  }

  useEffect(() => {
    void load();
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError('');
    if (password !== passwordConfirm) {
      setError(t('setup.passwordMismatch'));
      return;
    }
    try {
      await api.createUser({ email, username, displayName, password, roles });
      setEmail('');
      setUsername('');
      setDisplayName('');
      setPassword('');
      setPasswordConfirm('');
      await load();
    } catch (err) {
      setError((err as { message?: string }).message ?? t('common.error'));
    }
  }

  function toggleRole(role: RoleName) {
    setRoles((current) => current.includes(role) ? current.filter((item) => item !== role) : [...current, role]);
  }

  return (
    <div className="page">
      <header className="page-header"><h1>{t('users.title')}</h1></header>
      <Card unfinished>
        <form className="grid-form" onSubmit={(event) => void submit(event)}>
          {error && <p className="error" role="alert">{error}</p>}
          <FormField label={t('common.email')} name="email" type="email" value={email} onChange={(event) => setEmail(event.currentTarget.value)} required />
          <FormField label={t('common.username')} name="username" value={username} onChange={(event) => setUsername(event.currentTarget.value)} required />
          <FormField label={t('common.displayName')} name="displayName" value={displayName} onChange={(event) => setDisplayName(event.currentTarget.value)} required />
          <FormField label={t('common.password')} name="password" type="password" value={password} onChange={(event) => setPassword(event.currentTarget.value)} required />
          <FormField label={t('common.passwordConfirm')} name="passwordConfirm" type="password" value={passwordConfirm} onChange={(event) => setPasswordConfirm(event.currentTarget.value)} required />
          <fieldset className="role-group"><legend>{t('common.roles')}</legend>{(['admin', 'editor', 'viewer'] as RoleName[]).map((role) => <label key={role} className="check"><input type="checkbox" checked={roles.includes(role)} onChange={() => toggleRole(role)} />{t(`role.${role}` as const)}</label>)}</fieldset>
          <Button type="submit">{t('users.new')}</Button>
        </form>
      </Card>
      {!users.length ? <EmptyState message={t('common.empty')} /> : (
        <div className="cards-list">
          {users.map((user) => (
            <Card unfinished key={user.id}>
              <article className="event-row">
                <div>
                  <h2>{user.displayName}</h2>
                  <p>{user.username} - {user.email}</p>
                </div>
                <span>{user.roles.join(', ')}</span>
              </article>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
