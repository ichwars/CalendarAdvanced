import type { ButtonHTMLAttributes, PropsWithChildren } from 'react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'ghost' | 'danger' | 'selected';
}

export function Button({ variant = 'primary', children, ...props }: PropsWithChildren<ButtonProps>) {
  return (
    <button className={`button button--${variant}`} {...props}>
      {children}
    </button>
  );
}
