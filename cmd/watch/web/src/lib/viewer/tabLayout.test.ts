import { describe, it, expect } from 'vitest';
import { computeTabLayout, type LayoutTab } from './tabLayout';

function tab(id: string, width: number, overrides: Partial<LayoutTab> = {}): LayoutTab {
  return { id, width, isMain: false, selected: false, active: true, status: 'none', ...overrides };
}

const OPTIONS = { minTabWidth: 140, overflowWidth: 80 };

describe('computeTabLayout', () => {
  it('shows every tab at its natural width when they fit', () => {
    const layout = computeTabLayout([tab('main', 100, { isMain: true, selected: true }), tab('a', 200)], 400, OPTIONS);
    expect(layout.visible).toEqual([{ id: 'main' }, { id: 'a' }]);
    expect(layout.overflow).toEqual([]);
  });

  it('shrinks the widest tabs to a shared cap before hiding anything', () => {
    const tabs = [tab('main', 100, { isMain: true, selected: true }), tab('a', 300), tab('b', 300)];
    const layout = computeTabLayout(tabs, 500, OPTIONS);
    expect(layout.overflow).toEqual([]);
    expect(layout.visible).toEqual([{ id: 'main' }, { id: 'a', maxWidth: 200 }, { id: 'b', maxWidth: 200 }]);
  });

  it('never shrinks a tab below the minimum width', () => {
    const tabs = [tab('main', 200, { isMain: true, selected: true }), tab('a', 200), tab('b', 200)];
    const layout = computeTabLayout(tabs, 420, OPTIONS);
    expect(layout.visible.every((t) => (t.maxWidth ?? 200) >= 140)).toBe(true);
  });

  it('collapses tabs into the menu once shrinking is not enough', () => {
    const tabs = [
      tab('main', 200, { isMain: true, selected: true }),
      tab('a', 200),
      tab('b', 200),
      tab('c', 200),
      tab('d', 200),
    ];
    const layout = computeTabLayout(tabs, 500, OPTIONS);
    expect(layout.visible.map((t) => t.id)).toEqual(['main', 'a', 'b']);
    expect(layout.overflow).toEqual(['c', 'd']);
  });

  it('collapses removed tabs first, then clean, then seen, and keeps unseen longest', () => {
    const tabs = [
      tab('main', 200, { isMain: true, selected: true }),
      tab('unseen', 200, { status: 'unseen' }),
      tab('seen', 200, { status: 'seen' }),
      tab('clean', 200, { status: 'none' }),
      tab('removed', 200, { active: false }),
    ];
    // Room for main plus one more tab next to the menu button.
    const layout = computeTabLayout(tabs, 360, OPTIONS);
    expect(layout.visible.map((t) => t.id)).toEqual(['main', 'unseen']);
    expect(layout.overflow).toEqual(['seen', 'clean', 'removed']);
  });

  it('hides the rightmost of equally ranked tabs first', () => {
    const tabs = [
      tab('main', 200, { isMain: true, selected: true }),
      tab('a', 200),
      tab('b', 200),
      tab('c', 200),
    ];
    const layout = computeTabLayout(tabs, 360, OPTIONS);
    expect(layout.visible.map((t) => t.id)).toEqual(['main', 'a']);
    expect(layout.overflow).toEqual(['b', 'c']);
  });

  it('never hides the selected tab, even when it is the lowest priority', () => {
    const tabs = [
      tab('main', 200, { isMain: true }),
      tab('a', 200, { status: 'unseen' }),
      tab('picked', 200, { selected: true, active: false }),
    ];
    const layout = computeTabLayout(tabs, 360, OPTIONS);
    expect(layout.visible.map((t) => t.id)).toContain('picked');
    expect(layout.overflow).toEqual(['a']);
  });

  it('keeps the pinned tabs at the minimum width when nothing else can be hidden', () => {
    const tabs = [tab('main', 200, { isMain: true }), tab('picked', 200, { selected: true })];
    const layout = computeTabLayout(tabs, 200, OPTIONS);
    expect(layout.overflow).toEqual([]);
    expect(layout.visible).toEqual([{ id: 'main', maxWidth: 140 }, { id: 'picked', maxWidth: 140 }]);
  });

  it('keeps the original order of the tabs that stay visible and of those that overflow', () => {
    const tabs = [
      tab('main', 200, { isMain: true }),
      tab('a', 200, { selected: true }),
      tab('b', 200),
      tab('c', 200),
      tab('d', 200),
    ];
    const layout = computeTabLayout(tabs, 360, OPTIONS);
    expect(layout.visible.map((t) => t.id)).toEqual(['main', 'a']);
    expect(layout.overflow).toEqual(['b', 'c', 'd']);
  });

  it('counts the gap between tabs when deciding what fits', () => {
    const tabs = [tab('main', 100, { isMain: true, selected: true }), tab('a', 100), tab('b', 100)];
    const options = { minTabWidth: 50, overflowWidth: 60 };
    // 300 of tab plus two 4px gaps does not fit in 304.
    const tight = computeTabLayout(tabs, 304, { ...options, gap: 4 });
    expect(tight.visible.every((t) => t.maxWidth !== undefined)).toBe(true);
    const roomy = computeTabLayout(tabs, 308, { ...options, gap: 4 });
    expect(roomy.visible).toEqual([{ id: 'main' }, { id: 'a' }, { id: 'b' }]);
  });

  it('keeps one gap between the last tab and the "+N" button', () => {
    const tabs = [
      tab('main', 100, { isMain: true, selected: true }),
      tab('a', 100),
      tab('b', 100),
    ];
    // Without a gap main + a + the 60px button fit in 260; with 4px gaps they need 264.
    const noGap = computeTabLayout(tabs, 260, { minTabWidth: 100, overflowWidth: 60 });
    expect(noGap.visible.map((t) => t.id)).toEqual(['main', 'a']);
    const gapped = computeTabLayout(tabs, 260, { minTabWidth: 100, overflowWidth: 60, gap: 4 });
    expect(gapped.visible.map((t) => t.id)).toEqual(['main']);
  });
});
