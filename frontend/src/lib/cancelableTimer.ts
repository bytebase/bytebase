// Cancelable countdown timer for non-component callers (e.g. service
// modules, Zustand store factories). React components should use a hook
// that wires this into useEffect for proper cleanup.
export type CancelableTimer = {
  start: () => void;
  stop: () => void;
  elapsedMS: () => number;
  expired: () => boolean;
};

export const createCancelableTimer = (timeoutMS: number): CancelableTimer => {
  let running = false;
  let startTS = 0;
  return {
    start: () => {
      startTS = Date.now();
      running = true;
    },
    stop: () => {
      running = false;
    },
    elapsedMS: () => (running ? Date.now() - startTS : 0),
    expired: () => running && Date.now() - startTS > timeoutMS,
  };
};
