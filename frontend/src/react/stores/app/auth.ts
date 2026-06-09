import { create } from "@bufbuild/protobuf";
import { Code, createContextValues } from "@connectrpc/connect";
import { uniqueId } from "lodash-es";
import { authServiceClientConnect, userServiceClientConnect } from "@/connect";
import { ignoredCodesContextKey } from "@/connect/context-key";
import {
  AUTH_MFA_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_PROFILE_SETUP_MODULE,
  AUTH_SIGNIN_MODULE,
  SETUP_MODULE,
  SQL_EDITOR_HOME_MODULE,
} from "@/react/router/handles";
import {
  navigateByName,
  navigateToPath,
  resolvePath,
} from "@/react/router/navigation";
import {
  LoginRequestSchema,
  SendEmailLoginCodeRequestSchema,
  SignupRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UNKNOWN_USER_NAME } from "@/types/v1/user";
import { storageKeyResetPassword } from "@/utils/storage-keys";
import type { AppSliceCreator, AuthSlice } from "./types";

// `users/{email}` → `{email}`.
function emailOf(currentUserName: string | undefined): string {
  if (!currentUserName) return "";
  return currentUserName.startsWith("users/")
    ? currentUserName.slice("users/".length)
    : currentUserName;
}

function readResetPassword(email: string): boolean {
  try {
    return localStorage.getItem(storageKeyResetPassword(email)) === "true";
  } catch {
    return false;
  }
}

/**
 * Returns true if the user should be prompted to set up their profile.
 * First-time login with an auto-generated name (email local-part or full
 * email). Ported verbatim from the legacy Pinia auth store.
 */
function needsProfileSetup(user: User): boolean {
  if (user.profile?.lastLoginTime) return false;
  const name = user.title;
  const email = user.email;
  if (!name || !email) return false;
  if (name === email) return true;
  const atIndex = email.indexOf("@");
  if (atIndex > 0 && name === email.substring(0, atIndex)) return true;
  return false;
}

export const createAuthSlice: AppSliceCreator<AuthSlice> = (set, get) => ({
  unauthenticatedOccurred: false,
  authSessionKey: uniqueId(),
  isSelfEmailUpdate: false,

  isLoggedIn: () => {
    const name = get().currentUserName;
    return Boolean(name) && name !== UNKNOWN_USER_NAME;
  },

  requireResetPassword: () => {
    if (!get().isLoggedIn()) return false;
    return readResetPassword(emailOf(get().currentUserName));
  },

  setRequireResetPassword: (value) => {
    if (!get().isLoggedIn()) return;
    try {
      localStorage.setItem(
        storageKeyResetPassword(emailOf(get().currentUserName)),
        value ? "true" : "false"
      );
    } catch {
      // localStorage unavailable (private mode / quota) — non-fatal.
    }
  },

  setUnauthenticatedOccurred: (value) =>
    set({ unauthenticatedOccurred: value }),

  loadCurrentUser: async () => {
    const existing = get().currentUser;
    if (existing) return existing;
    const pending = get().currentUserRequest;
    if (pending) return pending;
    const request = userServiceClientConnect
      .getCurrentUser({})
      .then((user) => {
        set({
          currentUser: user,
          currentUserName: user.name,
          currentUserRequest: undefined,
        });
        return user;
      })
      .catch(() => {
        // Don't cache failures — the next call (e.g., after login) should
        // retry. Without this, a pre-login failure permanently prevents
        // the React store from loading the authenticated user.
        set({
          currentUser: undefined,
          currentUserName: undefined,
          currentUserRequest: undefined,
        });
        return undefined;
      });
    set({ currentUserRequest: request });
    return request;
  },

  // Force re-fetch (mirrors the Pinia `fetchCurrentUser`, which always hits
  // the server — login/signup need the fresh authenticated user).
  fetchCurrentUser: async () => {
    try {
      const user = await userServiceClientConnect.getCurrentUser({});
      set({ currentUser: user, currentUserName: user.name });
      return user;
    } catch {
      return undefined;
    }
  },

  // sometimes we have to redirect users even if we don't want to redirect them.
  // for example, the user is forced to reset their password,
  // or the user is using the LDAP to signin.
  login: async ({ request, redirect = true, redirectUrl }) => {
    const resp = await authServiceClientConnect.login(
      create(LoginRequestSchema, { ...request, web: true }),
      {
        contextValues: createContextValues().set(ignoredCodesContextKey, [
          Code.NotFound,
        ]),
      }
    );
    const getRedirectQuery = () =>
      new URLSearchParams(window.location.search).get("redirect");
    let nextPage = redirectUrl ?? (getRedirectQuery() || "/");
    if (resp.mfaTempToken) {
      set({ unauthenticatedOccurred: false });
      navigateByName(AUTH_MFA_MODULE, {
        query: { mfaTempToken: resp.mfaTempToken, redirect: nextPage },
      });
      return;
    }

    const user = await get().fetchCurrentUser();
    set({ unauthenticatedOccurred: !user });
    if (get().unauthenticatedOccurred) {
      return;
    }

    get().setRequireResetPassword(resp.requireResetPassword);
    await get().fetchServerInfo(user?.workspace);
    // Re-fetch the current workspace now that we're authenticated.
    await get().loadWorkspace();

    // After user login, reset the auth session key.
    set({ authSessionKey: uniqueId() });

    if (
      get().appFeatures["bb.feature.database-change-mode"] ===
      DatabaseChangeMode.EDITOR
    ) {
      nextPage = resolvePath(SQL_EDITOR_HOME_MODULE);
    }
    if (resp.requireResetPassword) {
      navigateByName(AUTH_PASSWORD_RESET_MODULE, {
        query: { redirect: nextPage },
      });
      return;
    }
    if (get().isSaaSMode() && resp.user && needsProfileSetup(resp.user)) {
      navigateByName(AUTH_PROFILE_SETUP_MODULE, {
        query: { redirect: nextPage },
      });
      return;
    }
    if (redirect) {
      navigateToPath(nextPage, { replace: true });
    }
  },

  signup: async (request) => {
    await authServiceClientConnect.signup(
      create(SignupRequestSchema, {
        email: request.email,
        title: request.name,
        password: request.password,
      })
    );

    // Signup sets HTTP-only cookies automatically. Fetch the current user and
    // proceed with the post-login flow.
    const user = await get().fetchCurrentUser();
    set({ unauthenticatedOccurred: !user });
    if (get().unauthenticatedOccurred) {
      return;
    }

    await get().fetchServerInfo(user?.workspace);
    set({ authSessionKey: uniqueId() });

    if (get().enableOnboarding()) {
      navigateByName(SETUP_MODULE, { replace: true });
      return;
    }

    const getRedirectQuery = () =>
      new URLSearchParams(window.location.search).get("redirect");
    let nextPage = getRedirectQuery() || "/";
    if (
      get().appFeatures["bb.feature.database-change-mode"] ===
      DatabaseChangeMode.EDITOR
    ) {
      nextPage = resolvePath(SQL_EDITOR_HOME_MODULE);
    }
    navigateToPath(nextPage, { replace: true });
  },

  logout: async () => {
    try {
      await authServiceClientConnect.logout({});
    } catch {
      // nothing
    } finally {
      set({ unauthenticatedOccurred: false });
      const pathname = location.pathname;
      const getRedirectQuery = () =>
        new URLSearchParams(window.location.search).get("redirect");
      // Replace and reload the page to clear frontend state directly.
      window.location.href = resolvePath(AUTH_SIGNIN_MODULE, {
        query: {
          redirect:
            getRedirectQuery() ||
            (pathname.startsWith("/auth") ? undefined : pathname),
        },
      });
    }
  },

  sendEmailLoginCode: async (email, workspace) => {
    await authServiceClientConnect.sendEmailLoginCode(
      create(SendEmailLoginCodeRequestSchema, { email, workspace })
    );
  },

  // Update currentUserName after a self email change. Sets the flag to
  // suppress the "logged in as another user" notification.
  updateCurrentUserNameForEmailChange: (newName) => {
    set({ isSelfEmailUpdate: true, currentUserName: newName });
  },

  setIsSelfEmailUpdate: (value) => set({ isSelfEmailUpdate: value }),
});
