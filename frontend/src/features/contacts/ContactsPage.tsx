import { type FormEvent, useEffect, useMemo, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { ConfirmDialog } from '../../shared/components/ConfirmDialog';
import { EmptyState } from '../../shared/components/EmptyState';
import { FormField } from '../../shared/components/FormField';
import { Icon } from '../../shared/components/Icon';
import { useI18n } from '../../shared/i18n';
import type { TranslationKey } from '../../shared/i18nTranslations';
import type { ContactItem } from '../../shared/types';

type ContactSortKey = 'name' | 'company' | 'contact' | 'birthday' | 'notes';
type SortDirection = 'asc' | 'desc';

export function ContactsPage() {
  const { locale, t } = useI18n();
  const [contacts, setContacts] = useState<ContactItem[]>([]);
  const [query, setQuery] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingContact, setEditingContact] = useState<ContactItem | null>(null);
  const [deleteContact, setDeleteContact] = useState<ContactItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [sortKey, setSortKey] = useState<ContactSortKey>('name');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  async function load() {
    const response = await api.contacts(new URLSearchParams({ q: query, limit: '500' }));
    setContacts(response.items);
  }

  useEffect(() => {
    void load();
  }, [query]);

  async function saveContact(body: Partial<ContactItem>) {
    if (editingContact) {
      await api.updateContact(editingContact.id, toContactPayload({ ...editingContact, ...body }));
      setEditingContact(null);
    } else {
      await api.createContact(toContactPayload(body));
      setDialogOpen(false);
    }
    await load();
  }

  async function confirmDeleteContact() {
    if (!deleteContact) {
      return;
    }
    setDeleting(true);
    try {
      await api.deleteContact(deleteContact.id);
      setDeleteContact(null);
      await load();
    } finally {
      setDeleting(false);
    }
  }

  function changeSort(nextKey: ContactSortKey) {
    if (sortKey === nextKey) {
      setSortDirection((current) => current === 'asc' ? 'desc' : 'asc');
      return;
    }
    setSortKey(nextKey);
    setSortDirection('asc');
  }

  const visibleContacts = useMemo(() => {
    const collator = new Intl.Collator(locale, { numeric: true, sensitivity: 'base' });
    return [...contacts].sort((left, right) => {
      const leftValue = getContactSortValue(left, sortKey);
      const rightValue = getContactSortValue(right, sortKey);
      const result = sortKey === 'birthday'
        ? String(leftValue).localeCompare(String(rightValue))
        : collator.compare(String(leftValue), String(rightValue));
      return sortDirection === 'asc' ? result : -result;
    });
  }, [contacts, locale, sortDirection, sortKey]);

  return (
    <div className="page">
      <header className="page-header events-page-header">
        <div>
          <h1>{t('contacts.title')}</h1>
        </div>
        <div className="events-page-header__actions">
          <div className={query ? 'events-search events-search--active' : 'events-search'}>
            <Icon name="search" />
            <input aria-label={t('contacts.search')} value={query} onChange={(event) => setQuery(event.currentTarget.value)} placeholder={t('contacts.search')} />
            {query && (
              <button type="button" onClick={() => setQuery('')} aria-label={t('common.clear')} title={t('common.clear')}>
                <Icon name="x" />
              </button>
            )}
          </div>
          <Button onClick={() => setDialogOpen(true)} type="button">+ {t('contacts.add')}</Button>
        </div>
      </header>

      {!contacts.length ? <EmptyState message={t('contacts.empty')} /> : (
        <div className="table-wrap events-table-wrap">
          <table className="events-table contacts-table">
            <thead>
              <tr>
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('common.name')} onSort={changeSort} sortKey="name" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('contacts.company')} onSort={changeSort} sortKey="company" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('contacts.contactData')} onSort={changeSort} sortKey="contact" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('contacts.birthday')} onSort={changeSort} sortKey="birthday" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('contacts.notes')} onSort={changeSort} sortKey="notes" />
                <th className="events-table__actions-heading">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {visibleContacts.map((contact) => (
                <tr key={contact.id}>
                  <td>
                    <div className="contacts-title-cell">
                      <span className="contact-avatar">{contactInitials(contact)}</span>
                      <div>
                        <strong>{contactName(contact)}</strong>
                        {contact.email && <a href={`mailto:${contact.email}`}>{contact.email}</a>}
                      </div>
                    </div>
                  </td>
                  <td>
                    {contact.company ? (
                      <div className="contacts-meta">
                        <strong>{contact.company}</strong>
                        {contact.companyEmail && <a href={`mailto:${contact.companyEmail}`}>{contact.companyEmail}</a>}
                        {contact.companyPhone && <span>{t('contacts.phone')}: {contact.companyPhone}</span>}
                        {contact.companyMobile && <span>{t('contacts.mobile')}: {contact.companyMobile}</span>}
                      </div>
                    ) : '-'}
                  </td>
                  <td>
                    <div className="contacts-meta">
                      {contact.phone && <span>{t('contacts.phone')}: {contact.phone}</span>}
                      {contact.mobile && <span>{t('contacts.mobile')}: {contact.mobile}</span>}
                      {!contact.phone && !contact.mobile && !contact.email && <span>-</span>}
                    </div>
                  </td>
                  <td>{contact.birthday ? formatDate(contact.birthday, locale) : '-'}</td>
                  <td className="contacts-notes">{contact.notes || contact.address || '-'}</td>
                  <td>
                    <div className="events-table__actions">
                      <button className="icon-button" type="button" onClick={() => setEditingContact(contact)} aria-label={t('common.edit')} title={t('common.edit')}>
                        <Icon name="pencil" />
                      </button>
                      <button className="icon-button icon-button--danger" type="button" onClick={() => setDeleteContact(contact)} aria-label={t('common.delete')} title={t('common.delete')}>
                        <Icon name="trash" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {(dialogOpen || editingContact) && (
        <ContactDialog
          contact={editingContact}
          onClose={() => {
            setDialogOpen(false);
            setEditingContact(null);
          }}
          onSave={saveContact}
        />
      )}
      {deleteContact && (
        <ConfirmDialog
          busy={deleting}
          message={deleteConfirmMessage(deleteContact, t)}
          onCancel={() => setDeleteContact(null)}
          onConfirm={() => void confirmDeleteContact()}
          title={t('contacts.deleteTitle')}
        />
      )}
    </div>
  );
}

function SortableHeader({
  activeKey,
  direction,
  label,
  onSort,
  sortKey
}: {
  activeKey: ContactSortKey;
  direction: SortDirection;
  label: string;
  onSort: (key: ContactSortKey) => void;
  sortKey: ContactSortKey;
}) {
  const active = activeKey === sortKey;
  return (
    <th aria-sort={active ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button className={active ? 'events-table__sort active' : 'events-table__sort'} type="button" onClick={() => onSort(sortKey)}>
        <span>{label}</span>
        <Icon className={active ? `sort-icon sort-icon--${direction}` : 'sort-icon'} name="sort" />
      </button>
    </th>
  );
}

function ContactDialog({ contact, onClose, onSave }: { contact?: ContactItem | null; onClose: () => void; onSave: (body: Partial<ContactItem>) => Promise<void> }) {
  const { t } = useI18n();
  const [firstName, setFirstName] = useState(contact?.firstName ?? '');
  const [lastName, setLastName] = useState(contact?.lastName ?? '');
  const [company, setCompany] = useState(contact?.company ?? '');
  const [companyEmail, setCompanyEmail] = useState(contact?.companyEmail ?? '');
  const [companyPhone, setCompanyPhone] = useState(contact?.companyPhone ?? '');
  const [companyMobile, setCompanyMobile] = useState(contact?.companyMobile ?? '');
  const [email, setEmail] = useState(contact?.email ?? '');
  const [phone, setPhone] = useState(contact?.phone ?? '');
  const [mobile, setMobile] = useState(contact?.mobile ?? '');
  const [address, setAddress] = useState(contact?.address ?? '');
  const [birthday, setBirthday] = useState(contact?.birthday ?? '');
  const [notes, setNotes] = useState(contact?.notes ?? '');
  const [companyOpen, setCompanyOpen] = useState(Boolean(contact?.company || contact?.companyEmail || contact?.companyPhone || contact?.companyMobile));
  const [saving, setSaving] = useState(false);
  const canSave = useMemo(() => Boolean(firstName.trim() || lastName.trim() || company.trim()), [company, firstName, lastName]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!canSave) {
      return;
    }
    setSaving(true);
    try {
      await onSave({ firstName, lastName, company, companyEmail, companyPhone, companyMobile, email, phone, mobile, address, birthday, notes });
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget) {
        onClose();
      }
    }}>
      <section className="modal" role="dialog" aria-modal="true" aria-labelledby="contact-dialog-title">
        <header className="modal__header">
          <h2 id="contact-dialog-title">{contact ? t('contacts.edit') : t('contacts.new')}</h2>
          <button className="icon-button" type="button" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
            <Icon name="x" />
          </button>
        </header>
        <form className="grid-form modal__form" onSubmit={(event) => void submit(event)}>
          <FormField label={t('contacts.firstName')} value={firstName} onChange={(event) => setFirstName(event.currentTarget.value)} />
          <FormField label={t('contacts.lastName')} value={lastName} onChange={(event) => setLastName(event.currentTarget.value)} />
          <FormField label={t('common.email')} value={email} onChange={(event) => setEmail(event.currentTarget.value)} type="email" />
          <FormField label={t('contacts.birthday')} value={birthday} onChange={(event) => setBirthday(event.currentTarget.value)} placeholder="YYYY-MM-DD" />
          <FormField label={t('contacts.phone')} value={phone} onChange={(event) => setPhone(event.currentTarget.value)} />
          <FormField label={t('contacts.mobile')} value={mobile} onChange={(event) => setMobile(event.currentTarget.value)} />
          <section className="contact-company-section field--wide">
            <button aria-expanded={companyOpen} className="contact-company-section__toggle" onClick={() => setCompanyOpen((current) => !current)} type="button">
              <span>{t('contacts.companySection')}</span>
              <Icon name={companyOpen ? 'chevron-down' : 'chevron-right'} />
            </button>
            {companyOpen && (
              <div className="contact-company-section__body">
                <FormField label={t('contacts.companyName')} value={company} onChange={(event) => setCompany(event.currentTarget.value)} />
                <FormField label={t('contacts.companyEmail')} value={companyEmail} onChange={(event) => setCompanyEmail(event.currentTarget.value)} type="email" />
                <FormField label={t('contacts.companyPhone')} value={companyPhone} onChange={(event) => setCompanyPhone(event.currentTarget.value)} />
                <FormField label={t('contacts.companyMobile')} value={companyMobile} onChange={(event) => setCompanyMobile(event.currentTarget.value)} />
              </div>
            )}
          </section>
          <label className="field field--wide">
            <span>{t('contacts.address')}</span>
            <textarea value={address} onChange={(event) => setAddress(event.currentTarget.value)} />
          </label>
          <label className="field field--wide">
            <span>{t('contacts.notes')}</span>
            <textarea value={notes} onChange={(event) => setNotes(event.currentTarget.value)} />
          </label>
          <div className="button-row modal__actions">
            <Button disabled={saving || !canSave} type="submit">{contact ? t('common.save') : t('contacts.add')}</Button>
            <Button disabled={saving} onClick={onClose} type="button" variant="ghost">{t('common.cancel')}</Button>
          </div>
        </form>
      </section>
    </div>
  );
}

function contactName(contact: ContactItem): string {
  const name = [contact.firstName, contact.lastName].filter(Boolean).join(' ').trim();
  return name || contact.company || '-';
}

function deleteConfirmMessage(contact: ContactItem, t: (key: TranslationKey) => string): string {
  const name = contactName(contact);
  if (name === '-' || name === '?') {
    return t('contacts.deleteConfirm');
  }
  return t('contacts.deleteNamedConfirm').replace('{name}', name);
}

function getContactSortValue(contact: ContactItem, sortKey: ContactSortKey): string {
  if (sortKey === 'company') {
    return contact.company ?? '';
  }
  if (sortKey === 'contact') {
    return [contact.email, contact.phone, contact.mobile, contact.companyEmail, contact.companyPhone, contact.companyMobile].filter(Boolean).join(' ');
  }
  if (sortKey === 'birthday') {
    return contact.birthday || '9999-12-31';
  }
  if (sortKey === 'notes') {
    return contact.notes || contact.address || '';
  }
  return [contact.lastName, contact.firstName].filter(Boolean).join(' ').trim() || contact.company || '';
}

function contactInitials(contact: ContactItem): string {
  const source = [contact.firstName, contact.lastName].filter(Boolean);
  if (source.length > 0) {
    return source.map((item) => item[0]).join('').slice(0, 2).toUpperCase();
  }
  return (contact.company || '?').slice(0, 2).toUpperCase();
}

function formatDate(value: string, locale: string): string {
  const [year, month, day] = value.split('-').map(Number);
  if (!year || !month || !day) {
    return value;
  }
  return new Date(year, month - 1, day, 12, 0, 0, 0).toLocaleDateString(locale);
}

function toContactPayload(contact: Partial<ContactItem>): Partial<ContactItem> {
  return {
    firstName: contact.firstName ?? '',
    lastName: contact.lastName ?? '',
    company: contact.company ?? '',
    companyEmail: contact.companyEmail ?? '',
    companyPhone: contact.companyPhone ?? '',
    companyMobile: contact.companyMobile ?? '',
    email: contact.email ?? '',
    phone: contact.phone ?? '',
    mobile: contact.mobile ?? '',
    address: contact.address ?? '',
    birthday: contact.birthday ?? '',
    notes: contact.notes ?? ''
  };
}
