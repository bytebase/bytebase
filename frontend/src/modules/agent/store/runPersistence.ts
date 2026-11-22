import { withAgentPersistenceLock } from "../logic/runOwnership";

export const createAgentRunPersistence = (
  scopeKey: string,
  writeUnderLock: () => void | Promise<void>
) => {
  let dirty = false;
  let pending: Promise<boolean> | undefined;
  let error: unknown;

  const start = (): Promise<boolean> => {
    pending = Promise.resolve()
      .then(async () => {
        while (dirty) {
          dirty = false;
          try {
            await withAgentPersistenceLock(scopeKey, writeUnderLock);
            error = undefined;
          } catch (failure) {
            dirty = true;
            error = failure ?? new Error("Agent persistence failed");
            return false;
          }
        }
        return true;
      })
      .then((success) => {
        pending = undefined;
        // A mutation can arrive after draining but before this completion.
        if (success && dirty) void start();
        return success;
      });
    return pending;
  };

  return {
    schedule(): void {
      dirty = true;
      if (!pending) void start();
    },
    async flush(): Promise<boolean> {
      while (dirty || pending) {
        if (!(await (pending ?? start()))) return false;
      }
      return true;
    },
    getError(): unknown | undefined {
      return error;
    },
  };
};
