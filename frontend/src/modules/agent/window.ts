export const MIN_SIDEBAR_WIDTH = 180;
export const MIN_MAIN_PANEL_WIDTH = 240;
export const WINDOW_CHROME_BUDGET = 40;
export const MIN_WIDTH =
  MIN_SIDEBAR_WIDTH + MIN_MAIN_PANEL_WIDTH + WINDOW_CHROME_BUDGET;
export const MIN_HEIGHT = 400;
export const WINDOW_MARGIN = 16;
export const DEFAULT_SIDEBAR_WIDTH = 200;

const DEFAULT_WIDTH_RATIO = 0.6;
const DEFAULT_HEIGHT_RATIO = 0.6;
const MAX_DEFAULT_WIDTH = 750;
const MAX_DEFAULT_HEIGHT = 590;

const clampDefaultDimension = (
  preferred: number,
  preferredMin: number,
  viewportMax: number,
  preferredMax: number
) => {
  const max = Math.max(1, viewportMax - WINDOW_MARGIN * 2);
  const min = Math.min(preferredMin, max);
  return Math.min(
    Math.min(preferredMax, max),
    Math.max(min, Math.round(preferred))
  );
};

export const getDefaultAgentWindowState = (
  viewportWidth: number,
  viewportHeight: number
) => {
  const availableWidth = Math.max(1, viewportWidth - WINDOW_MARGIN * 2);
  const availableHeight = Math.max(1, viewportHeight - WINDOW_MARGIN * 2);
  const width = clampDefaultDimension(
    availableWidth * DEFAULT_WIDTH_RATIO,
    MIN_WIDTH,
    viewportWidth,
    MAX_DEFAULT_WIDTH
  );
  const height = clampDefaultDimension(
    availableHeight * DEFAULT_HEIGHT_RATIO,
    MIN_HEIGHT,
    viewportHeight,
    MAX_DEFAULT_HEIGHT
  );

  return {
    size: {
      width,
      height,
    },
    sidebarWidth: DEFAULT_SIDEBAR_WIDTH,
  };
};
