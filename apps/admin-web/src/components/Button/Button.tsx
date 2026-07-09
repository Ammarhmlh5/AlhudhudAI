import { ButtonHTMLAttributes, ReactNode } from 'react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
  children?: ReactNode;
}

export function Button({ loading, disabled, children, ...props }: ButtonProps) {
  return (
    <button disabled={disabled || loading} {...props}>
      {loading ? 'loading...' : children}
    </button>
  );
}
