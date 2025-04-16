import { useLocalStorage } from "@vueuse/core";
import axios from "axios";
import { uniqueId } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { userServiceClient } from "@/grpcweb";
import { router } from "@/router";
import {
  AUTH_SIGNIN_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_MFA_MODULE,
} from "@/router/auth";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useAppFeature, useSettingV1Store, useUserStore } from "@/store";
import { UNKNOWN_USER_NAME, unknownUser } from "@/types";
import type { LoginRequest } from "@/types/proto/v1/auth_service";
import { LoginResponse } from "@/types/proto/v1/auth_service";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { User, UserType } from "@/types/proto/v1/user_service";

export const useAuthStore = defineStore("auth_v1", () => {
  const userStore = useUserStore();
  const authSessionKey = ref<string>(uniqueId());
  const unauthenticatedOccurred = ref<boolean>(false);
  // Format: users/{user}. {user} is a system-generated unique ID.
  const currentUserName = ref<string | undefined>(undefined);

  const isLoggedIn = computed(() => {
    return (
      Boolean(currentUserName.value) &&
      currentUserName.value !== UNKNOWN_USER_NAME
    );
  });

  const requireResetPassword = computed(() => {
    if (!isLoggedIn.value) {
      return false;
    }
    return useLocalStorage<boolean>(
      `${currentUserName.value}.require_reset_password`,
      false
    ).value;
  });

  const setRequireResetPassword = (requireResetPassword: boolean) => {
    if (!isLoggedIn.value) {
      return false;
    }
    const needResetPasswordCache = useLocalStorage<boolean>(
      `${currentUserName.value}.require_reset_password`,
      false
    );
    needResetPasswordCache.value = requireResetPassword;
  };

  const getRedirectQuery = () => {
    const query = new URLSearchParams(window.location.search);
    return query.get("redirect");
  };

  const login = async (
    request: Partial<LoginRequest>,
    redirect: string = ""
  ) => {
    const { data } = await axios.post<LoginResponse>("/v1/auth/login", request);
    const redirectUrl = redirect || getRedirectQuery();
    if (data.mfaTempToken) {
      unauthenticatedOccurred.value = false;
      return router.push({
        name: AUTH_MFA_MODULE,
        query: {
          mfaTempToken: data.mfaTempToken,
          redirect: redirectUrl,
        },
      });
    }

    await fetchCurrentUser();
    setRequireResetPassword(data.requireResetPassword);

    await useSettingV1Store().getOrFetchSettingByName(
      "bb.workspace.profile",
      true // silent
    );

    if (!unauthenticatedOccurred.value) {
      const mode = useAppFeature("bb.feature.database-change-mode");
      let nextPage = redirectUrl || "/";
      if (mode.value === DatabaseChangeMode.EDITOR) {
        const route = router.resolve({
          name: SQL_EDITOR_HOME_MODULE,
        });
        nextPage = route.fullPath;
      }
      if (data.requireResetPassword) {
        return router.push({
          name: AUTH_PASSWORD_RESET_MODULE,
          query: {
            redirect: nextPage,
          },
        });
      }
      return router.replace(nextPage);
    }
    unauthenticatedOccurred.value = false;
  };

  const signup = async (request: Partial<User>) => {
    await userServiceClient.createUser({
      user: {
        email: request.email,
        title: request.name,
        password: request.password,
        userType: UserType.USER,
      },
    });
    await login({
      email: request.email,
      password: request.password,
      web: true,
    });
  };

  const logout = async () => {
    try {
      await axios.post("/v1/auth/logout");
    } catch {
      // nothing
    } finally {
      const pathname = location.pathname;
      // Replace and reload the page to clear frontend state directly.
      window.location.href = router.resolve({
        name: AUTH_SIGNIN_MODULE,
        query: {
          redirect:
            getRedirectQuery() ||
            (pathname.startsWith("/auth") ? undefined : pathname),
        },
      }).fullPath;
    }
  };

  const fetchCurrentUser = async () => {
    try {
      const currentUser = await userStore.fetchCurrentUser();
      currentUserName.value = currentUser.name;
      // After user login, we need to reset the auth session key.
      authSessionKey.value = uniqueId();
    } catch {
      // do nothing.
    }
  };

  return {
    currentUserName,
    isLoggedIn,
    unauthenticatedOccurred,
    requireResetPassword,
    authSessionKey,
    login,
    signup,
    logout,
    fetchCurrentUser,
    setRequireResetPassword,
  };
});

export const useCurrentUserV1 = () => {
  const authStore = useAuthStore();
  const userStore = useUserStore();
  return computed(
    () =>
      userStore.getUserByIdentifier(authStore.currentUserName || "") ||
      unknownUser()
  );
};
