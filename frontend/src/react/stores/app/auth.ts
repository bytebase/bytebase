import { authServiceClientConnect, userServiceClientConnect } from "@/connect";
import type { AppSliceCreator, AuthSlice } from "./types";

export const createAuthSlice: AppSliceCreator<AuthSlice> = (set, get) => ({
  loadCurrentUser: async () => {
    const existing = get().currentUser;
    if (existing) return existing;
    const pending = get().currentUserRequest;
    if (pending) return pending;
    const request = userServiceClientConnect
      .getCurrentUser({})
      .then((user) => {
        set({ currentUser: user, currentUserRequest: undefined });
        return user;
      })
      .catch(() => {
        // Don't cache failures — the next call (e.g., after login) should
        // retry. Without this, a pre-login failure permanently prevents
        // the React store from loading the authenticated user.
        set({ currentUser: undefined, currentUserRequest: undefined });
        return undefined;
      });
    set({ currentUserRequest: request });
    return request;
  },

  logout: async (signinUrl) => {
    try {
      await authServiceClientConnect.logout({});
    } catch {
      // Ignore logout errors and clear the local session by redirecting anyway.
    } finally {
      window.location.href = signinUrl;
    }
  },
});
