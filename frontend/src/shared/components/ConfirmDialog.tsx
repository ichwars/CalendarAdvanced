import { useId } from 'react';
import { Button } from './Button';
import { Icon, type IconName } from './Icon';
import { useI18n } from '../i18n';

interface ConfirmDialogProps {
  busy?: boolean;
  confirmLabel?: string;
  icon?: IconName;
  message: string;
  onCancel: () => void;
  onConfirm: () => void;
  title: string;
  variant?: 'danger';
}

export function ConfirmDialog({
  busy = false,
  confirmLabel,
  icon = 'trash',
  message,
  onCancel,
  onConfirm,
  title,
  variant = 'danger'
}: ConfirmDialogProps) {
  const { t } = useI18n();
  const id = useId();
  const titleId = `${id}-confirm-title`;
  const messageId = `${id}-confirm-message`;
  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !busy) {
        onCancel();
      }
    }}>
      <section className="modal modal--confirm" role="alertdialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={messageId}>
        <header className="modal__header confirm-dialog__header">
          <div className="confirm-dialog__title">
            <span className={`confirm-dialog__icon confirm-dialog__icon--${variant}`} aria-hidden="true">
              <Icon name={icon} />
            </span>
            <h2 id={titleId}>{title}</h2>
          </div>
          <button className="icon-button" disabled={busy} type="button" onClick={onCancel} aria-label={t('common.close')} title={t('common.close')}>
            <Icon name="x" />
          </button>
        </header>
        <p id={messageId} className="confirm-dialog__message">{message}</p>
        <div className="button-row modal__actions confirm-dialog__actions">
          <Button disabled={busy} onClick={onConfirm} type="button" variant={variant}>{confirmLabel ?? t('common.delete')}</Button>
          <Button disabled={busy} onClick={onCancel} type="button" variant="ghost">{t('common.cancel')}</Button>
        </div>
      </section>
    </div>
  );
}
