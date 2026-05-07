export const MIN_SIDEBAR_WIDTH = 180;
export const MIN_MAIN_PANEL_WIDTH = 240;
export const WINDOW_CHROME_BUDGET = 40;
export const MIN_WIDTH =
  MIN_SIDEBAR_WIDTH + MIN_MAIN_PANEL_WIDTH + WINDOW_CHROME_BUDGET;
export const MIN_HEIGHT = 400;
export const WINDOW_MARGIN = 16;
export const DEFAULT_SIDEBAR_WIDTH = 200;

const DEFAULT_WIDTH_RATIO = 0.56;
const DEFAULT_HEIGHT_RATIO = 0.72;
const MAX_DEFAULT_WIDTH = 960;
const MAX_DEFAULT_HEIGHT = 820;

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

const clampDisplayDimension = (
  size: number,
  viewportSize: number,
  preferredMin: number
) => {
  const max = Math.max(1, viewportSize - WINDOW_MARGIN * 2);
  const min = Math.min(preferredMin, max);
  return Math.min(max, Math.max(min, Math.round(size)));
};

export const getCenteredAgentWindowPosition = (
  viewportWidth: number,
  viewportHeight: number,
  windowWidth: number,
  windowHeight: number
) => {
  const width = clampDisplayDimension(windowWidth, viewportWidth, MIN_WIDTH);
  const height = clampDisplayDimension(
    windowHeight,
    viewportHeight,
    MIN_HEIGHT
  );

  return {
    x: Math.max(WINDOW_MARGIN, Math.round((viewportWidth - width) / 2)),
    y: Math.max(WINDOW_MARGIN, Math.round((viewportHeight - height) / 2)),
  };
};

export const clampAgentWindowPosition = (
  viewportWidth: number,
  viewportHeight: number,
  windowWidth: number,
  windowHeight: number,
  position: { x: number; y: number }
) => {
  const width = clampDisplayDimension(windowWidth, viewportWidth, MIN_WIDTH);
  const height = clampDisplayDimension(
    windowHeight,
    viewportHeight,
    MIN_HEIGHT
  );
  const maxX = Math.max(WINDOW_MARGIN, viewportWidth - width - WINDOW_MARGIN);
  const maxY = Math.max(WINDOW_MARGIN, viewportHeight - height - WINDOW_MARGIN);

  return {
    x: Math.min(maxX, Math.max(WINDOW_MARGIN, Math.round(position.x))),
    y: Math.min(maxY, Math.max(WINDOW_MARGIN, Math.round(position.y))),
  };
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
    position: getCenteredAgentWindowPosition(
      viewportWidth,
      viewportHeight,
      width,
      height
    ),
    size: {
      width,
      height,
    },
    sidebarWidth: DEFAULT_SIDEBAR_WIDTH,
  };
};
