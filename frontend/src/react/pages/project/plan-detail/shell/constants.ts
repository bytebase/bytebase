export const POLLER_INTERVAL = {
  min: 1000,
  max: 30000,
  growth: 2,
  jitter: 250,
} as const;

export const PROJECT_NAME_PREFIX = "projects/";

// Width below which the layout switches to mobile mode.
export const MOBILE_BREAKPOINT_PX = 780;
// Width at or above which the task detail panel renders inline as a
// side-by-side column instead of as a drawer.
export const INLINE_TASK_PANEL_BREAKPOINT_PX = 1024;
