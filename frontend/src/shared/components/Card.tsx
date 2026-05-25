import type { HTMLAttributes, PropsWithChildren } from 'react';

interface CardProps extends HTMLAttributes<HTMLElement> {
  unfinished?: boolean;
}

export function Card({ children, className = '', unfinished = false, ...props }: PropsWithChildren<CardProps>) {
  const classes = ['card', unfinished ? 'card--unfinished' : '', className].filter(Boolean).join(' ');
  return <section className={classes} {...props}>{children}</section>;
}
