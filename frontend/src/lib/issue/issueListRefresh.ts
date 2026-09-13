import { useEffect } from "react";

// Global "refresh the issue list" signal.
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
