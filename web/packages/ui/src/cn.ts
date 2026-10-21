import { clsx, type ClassValue } from 'clsx';

/** Joins class names (clsx). Tailwind conflicts are avoided by design, not by tailwind-merge. */
export function cn(...inputs: ClassValue[]): string {
  return clsx(inputs);
}
