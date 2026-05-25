import type { InputHTMLAttributes, PropsWithChildren } from 'react';

interface FormFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
}

export function FormField({ label, children, id, ...props }: PropsWithChildren<FormFieldProps>) {
  const inputId = id ?? props.name;
  return (
    <label className="field" htmlFor={inputId}>
      <span>{label}</span>
      {children ?? <input id={inputId} {...props} />}
    </label>
  );
}
