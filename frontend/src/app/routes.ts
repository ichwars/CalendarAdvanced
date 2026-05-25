import type { TranslationKey } from '../shared/i18nTranslations';
import type { RoleName } from '../shared/types';

export interface AppRoute {
  path: string;
  label: TranslationKey;
  adminOnly?: boolean;
  minRole?: RoleName;
  nav?: 'main' | 'settings';
}

export const routes: AppRoute[] = [
  { path: 'overview', label: 'app.overview', nav: 'main' },
  { path: 'calendar', label: 'nav.calendar', nav: 'main' },
  { path: 'events', label: 'nav.events', minRole: 'editor', nav: 'main' },
  { path: 'tasks', label: 'nav.tasks', nav: 'main' },
  { path: 'contacts', label: 'nav.contacts', nav: 'main' },
  { path: 'settings', label: 'nav.settings', nav: 'main' },
  { path: 'security', label: 'nav.security', nav: 'settings' },
  { path: 'integrations', label: 'nav.integrations', nav: 'settings' },
  { path: 'exports', label: 'nav.exports', nav: 'settings' },
  { path: 'users', label: 'nav.users', adminOnly: true, nav: 'settings' },
  { path: 'audit', label: 'nav.audit', adminOnly: true, nav: 'settings' }
];
