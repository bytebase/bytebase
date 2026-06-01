import { useEffect } from "react";

// Global "refresh the issue list" signal, relocated from the legacy Pinia
// `issue` module (which used a Vue ref + watch). Plain module-level listener
// set so callers don't pull Vue reactivity into React feature code.
const listeners = new Set<() => void>();

export const refreshIssueList = (): void => {
  for (const listener of listeners) {
    listener();
  }
};

export const useRefreshIssueList = (callback: () => void): void => {
  useEffect(() => {
    listeners.add(callback);
    return () => {
      listeners.delete(callback);
    };
  }, [callback]);
};
