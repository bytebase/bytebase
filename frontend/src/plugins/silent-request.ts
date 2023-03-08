const state = {
  silent: false,
};

export const isSilent = () => {
  return state.silent;
};

// Requests wrapped in useSilentRequest won't be intercepted by our global
// error handling (pushNotifications)
export const useSilentRequest = async <T>(fn: () => Promise<T>): Promise<T> => {
  state.silent = true;
  const result = await fn();
  state.silent = false;
  return result;
};
