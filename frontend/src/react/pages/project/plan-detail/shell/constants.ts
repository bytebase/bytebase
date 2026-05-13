export const POLLER_INTERVAL = {
  min: 1000,
  max: 30000,
  growth: 2,
  jitter: 250,
} as const;

export const PROJECT_NAME_PREFIX = "projects/";

// Width below which the sidebar collapses into a mobile drawer.
export const MOBILE_BREAKPOINT_PX = 780;
// Width at or above which the sidebar widens.
export const WIDE_SIDEBAR_BREAKPOINT_PX = 1280;
export const SIDEBAR_WIDTH_NARROW_PX = 240;
export const SIDEBAR_WIDTH_WIDE_PX = 336;
