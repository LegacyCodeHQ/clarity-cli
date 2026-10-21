/**
 * Decides which worktree tabs fit in the top bar.
 *
 * Tabs shrink first (long labels truncate down to a minimum width). Only when
 * that is not enough do tabs collapse into a "+N" menu. The main worktree and
 * the selected tab are never collapsed.
 */

import type { WorktreeChangeStatus } from './viewerState';

export const MIN_TAB_WIDTH = 140;

export interface LayoutTab {
  id: string;
  // Width the tab takes at its natural, untruncated size.
  width: number;
  isMain: boolean;
  selected: boolean;
  // False once the worktree was removed.
  active: boolean;
  status: WorktreeChangeStatus;
}

export interface VisibleTab {
  id: string;
  // Set when the tab has to truncate its label to fit.
  maxWidth?: number;
}

export interface TabLayout {
  visible: VisibleTab[];
  // Ids of the tabs that moved into the menu, in their original order.
  overflow: string[];
}

export interface TabLayoutOptions {
  minTabWidth?: number;
  // Space between neighbouring tabs.
  gap?: number;
  // Width the "+N" button takes when it is shown.
  overflowWidth: number;
}

// The tabs that collapse first have the lowest rank: removed tabs, then clean
// ones, then ones with changes already seen. Tabs with unseen changes stay
// visible the longest.
function collapseRank(tab: LayoutTab): number {
  if (!tab.active) return 0;
  switch (tab.status) {
    case 'none':
      return 1;
    case 'seen':
      return 2;
    case 'unseen':
      return 3;
  }
}

/**
 * Finds the widest cap that lets every tab fit in `available`, never going
 * below `minWidth`. Returns Infinity when no tab needs to shrink and null when
 * even the minimum width does not fit.
 */
function widthCap(widths: number[], available: number, minWidth: number, gap: number): number | null {
  const gaps = gap * Math.max(widths.length - 1, 0);
  const sum = (cap: number) => widths.reduce((total, w) => total + Math.min(w, cap), gaps);
  if (sum(Infinity) <= available) return Infinity;
  if (sum(minWidth) > available) return null;

  let low = minWidth;
  let high = Math.max(...widths);
  while (low < high) {
    const mid = Math.ceil((low + high) / 2);
    if (sum(mid) <= available) {
      low = mid;
    } else {
      high = mid - 1;
    }
  }
  return low;
}

function visibleWithCap(tabs: LayoutTab[], cap: number): VisibleTab[] {
  return tabs.map((tab) => (tab.width > cap ? { id: tab.id, maxWidth: cap } : { id: tab.id }));
}

export function computeTabLayout(tabs: LayoutTab[], available: number, options: TabLayoutOptions): TabLayout {
  const minWidth = options.minTabWidth ?? MIN_TAB_WIDTH;
  const gap = options.gap ?? 0;

  const everything = widthCap(tabs.map((t) => t.width), available, minWidth, gap);
  if (everything !== null) {
    return { visible: visibleWithCap(tabs, everything), overflow: [] };
  }

  // Collapse the least important tabs one at a time, rightmost first among
  // equals, until what is left fits next to the "+N" button.
  const candidates = tabs
    .map((tab, index) => ({ tab, index }))
    .filter(({ tab }) => !tab.isMain && !tab.selected)
    .sort((a, b) => collapseRank(a.tab) - collapseRank(b.tab) || b.index - a.index);

  const hidden = new Set<string>();
  const remaining = () => tabs.filter((t) => !hidden.has(t.id));
  // The "+N" button sits one gap after the last tab.
  const room = available - options.overflowWidth - gap;

  for (const { tab } of candidates) {
    hidden.add(tab.id);
    const left = remaining();
    const cap = widthCap(left.map((t) => t.width), room, minWidth, gap);
    if (cap !== null) {
      return {
        visible: visibleWithCap(left, cap),
        overflow: tabs.filter((t) => hidden.has(t.id)).map((t) => t.id),
      };
    }
  }

  // Only the pinned tabs are left and they still do not fit: show them at the
  // minimum width rather than hiding them.
  return {
    visible: visibleWithCap(remaining(), minWidth),
    overflow: tabs.filter((t) => hidden.has(t.id)).map((t) => t.id),
  };
}
