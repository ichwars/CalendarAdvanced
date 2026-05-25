import { useEffect, useState } from 'react';
import { api } from '../../shared/api';
import { Card } from '../../shared/components/Card';
import { EmptyState } from '../../shared/components/EmptyState';
import { useI18n } from '../../shared/i18n';
import type { AuditEntry } from '../../shared/types';

export function AuditPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<AuditEntry[]>([]);

  useEffect(() => {
    void api.audit().then((response) => setItems(response.items));
  }, []);

  return (
    <div className="page">
      <header className="page-header"><h1>{t('audit.title')}</h1></header>
      <Card>
        {!items.length ? <EmptyState message={t('common.empty')} /> : (
          <div className="table-wrap">
            <table>
              <thead><tr><th>{t('audit.createdAt')}</th><th>{t('audit.action')}</th><th>{t('audit.actor')}</th></tr></thead>
              <tbody>{items.map((entry) => <tr key={entry.id}><td>{new Date(entry.createdAt).toLocaleString()}</td><td>{entry.action}</td><td>{entry.actorId}</td></tr>)}</tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
