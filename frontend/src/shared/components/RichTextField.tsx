import { useRef } from 'react';
import { Icon, type IconName } from './Icon';
import { useI18n } from '../i18n';
import type { TranslationKey } from '../i18nTranslations';

interface RichTextFieldProps {
  id: string;
  label: string;
  onChange: (value: string) => void;
  value: string;
}

type ToolbarAction = 'bold' | 'italic' | 'underline' | 'ordered' | 'bullet' | 'link' | 'clear';

const actions: Array<{ action: ToolbarAction; icon: IconName; labelKey: TranslationKey; separatorAfter?: boolean }> = [
  { action: 'bold', icon: 'bold', labelKey: 'editor.bold' },
  { action: 'italic', icon: 'italic', labelKey: 'editor.italic' },
  { action: 'underline', icon: 'underline', labelKey: 'editor.underline', separatorAfter: true },
  { action: 'ordered', icon: 'list-ordered', labelKey: 'editor.orderedList' },
  { action: 'bullet', icon: 'list-bullets', labelKey: 'editor.bulletList', separatorAfter: true },
  { action: 'link', icon: 'link', labelKey: 'editor.link' },
  { action: 'clear', icon: 'remove-format', labelKey: 'editor.clearFormatting' }
];

export function RichTextField({ id, label, onChange, value }: RichTextFieldProps) {
  const { t } = useI18n();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  function applyAction(action: ToolbarAction) {
    const textarea = textareaRef.current;
    if (!textarea) return;
    const selectionStart = textarea.selectionStart;
    const selectionEnd = textarea.selectionEnd;
    const selected = value.slice(selectionStart, selectionEnd);
    const before = value.slice(0, selectionStart);
    const after = value.slice(selectionEnd);

    if (action === 'ordered' || action === 'bullet') {
      replaceSelection(before, after, formatLines(selected, action), selectionStart);
      return;
    }
    if (action === 'clear') {
      if (selected) {
        replaceSelection(before, after, clearFormatting(selected), selectionStart);
      } else {
        const cleared = clearFormatting(value);
        onChange(cleared);
        requestAnimationFrame(() => {
          const textarea = textareaRef.current;
          if (!textarea) return;
          textarea.focus();
          textarea.setSelectionRange(0, cleared.length);
        });
      }
      return;
    }
    if (action === 'link') {
      const text = selected || 'Link';
      replaceSelection(before, after, `[${text}](https://)`, selectionStart, 1, text.length + 1);
      return;
    }

    const wrappers = {
      bold: ['**', '**', t('editor.boldPlaceholder')],
      italic: ['_', '_', t('editor.italicPlaceholder')],
      underline: ['<u>', '</u>', t('editor.underlinePlaceholder')]
    } satisfies Record<'bold' | 'italic' | 'underline', [string, string, string]>;
    const [prefix, suffix, placeholder] = wrappers[action];
    const text = selected || placeholder;
    replaceSelection(before, after, `${prefix}${text}${suffix}`, selectionStart, prefix.length, prefix.length + text.length);
  }

  function replaceSelection(before: string, after: string, replacement: string, baseIndex: number, selectionStartOffset = replacement.length, selectionEndOffset = replacement.length) {
    onChange(`${before}${replacement}${after}`);
    requestAnimationFrame(() => {
      const textarea = textareaRef.current;
      if (!textarea) return;
      textarea.focus();
      textarea.setSelectionRange(baseIndex + selectionStartOffset, baseIndex + selectionEndOffset);
    });
  }

  return (
    <label className="field field--wide rich-text-field" htmlFor={id}>
      <span>{label}</span>
      <div className="rich-text-editor">
        <div className="rich-text-toolbar" aria-label={t('editor.toolbar')}>
          {actions.map((item) => (
            <span className="rich-text-toolbar__group" key={item.action}>
              <button
                aria-label={t(item.labelKey)}
                className="rich-text-toolbar__button"
                onClick={() => applyAction(item.action)}
                title={t(item.labelKey)}
                type="button"
              >
                <Icon name={item.icon} />
              </button>
              {item.separatorAfter && <span className="rich-text-toolbar__separator" />}
            </span>
          ))}
        </div>
        <textarea id={id} ref={textareaRef} value={value} onChange={(event) => onChange(event.currentTarget.value)} />
      </div>
    </label>
  );
}

function formatLines(value: string, action: 'ordered' | 'bullet'): string {
  const lines = (value || '').split('\n');
  const source = lines.some((line) => line.trim()) ? lines : [''];
  return source
    .map((line, index) => {
      const clean = line.replace(/^\s*(?:[-*]\s+|\d+[.)]\s+)/, '');
      return action === 'ordered' ? `${index + 1}. ${clean || 'Listeneintrag'}` : `- ${clean || 'Listeneintrag'}`;
    })
    .join('\n');
}

function clearFormatting(value: string): string {
  return value
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/_([^_]+)_/g, '$1')
    .replace(/<u>(.*?)<\/u>/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/^\s*(?:[-*]\s+|\d+[.)]\s+)/gm, '');
}
