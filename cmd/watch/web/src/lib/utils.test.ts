import { describe, it, expect } from 'vitest';
import { cn } from './utils';

describe('cn', () => {
  it('keeps a custom font size next to a text color', () => {
    expect(cn('text-caption', 'text-muted-foreground')).toBe('text-caption text-muted-foreground');
    expect(cn('text-body font-medium', 'text-foreground')).toBe('text-body font-medium text-foreground');
  });

  it('still lets a later class of the same kind win', () => {
    expect(cn('text-caption', 'text-body')).toBe('text-body');
    expect(cn('text-muted-foreground', 'text-foreground')).toBe('text-foreground');
    expect(cn('px-2', 'px-3')).toBe('px-3');
  });
});
