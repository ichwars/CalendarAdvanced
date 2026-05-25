import { useEffect, useId, useRef, useState } from 'react';
import { Icon } from './Icon';

export interface SelectOption<T extends string> {
  label: string;
  value: T;
}

interface SelectFieldProps<T extends string> {
  label: string;
  onChange: (value: T) => void;
  options: SelectOption<T>[];
  value: T;
}

export function SelectField<T extends string>({ label, onChange, options, value }: SelectFieldProps<T>) {
  const id = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const selected = options.find((option) => option.value === value) ?? options[0];

  useEffect(() => {
    function closeOnOutsideInteraction(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    }

    document.addEventListener('pointerdown', closeOnOutsideInteraction);
    document.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOnOutsideInteraction);
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, []);

  function select(next: T) {
    onChange(next);
    setOpen(false);
  }

  return (
    <div className="field select-field" ref={rootRef}>
      <span id={`${id}-label`}>{label}</span>
      <button
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-labelledby={`${id}-label ${id}-button`}
        className="select-field__trigger"
        id={`${id}-button`}
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span>{selected?.label}</span>
        <Icon name="chevron-down" />
      </button>
      {open && (
        <div className="select-field__panel" role="listbox" aria-labelledby={`${id}-label`}>
          {options.map((option) => (
            <button
              aria-selected={option.value === value}
              className={option.value === value ? 'select-field__option active' : 'select-field__option'}
              key={option.value}
              onClick={() => select(option.value)}
              role="option"
              type="button"
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
