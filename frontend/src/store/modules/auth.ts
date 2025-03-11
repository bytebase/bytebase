import { useLocalStorage } from "@vueuse/core";
import axios from "axios";
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
import { useAppFeature, useUserStore, useSettingV1Store } from "@/store";
import { unknownUser } from "@/types";
import type { LoginRequest } from "@/types/proto/v1/auth_service";
import { LoginResponse } from "@/types/proto/v1/auth_service";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { User, UserType } from "@/types/proto/v1/user_service";
import { getIntCookie } from "@/utils";

export const useAuthStore = defineStore("auth_v1", () => {
  const userStore = useUserStore();
  const currentUserId = ref<number | undefined>();
  const showLoginModal = ref<boolean>(false);

  const currentUser = computed(() => {
    if (currentUserId.value) {
      return (
        userStore.getUserByIdentifier(`${currentUserId.value}`) ?? unknownUser()
      );
    }
    return unknownUser();
  });

  const isLoggedIn = () => {
    return getUserIdFromCookie() != undefined;
  };

  const getUserIdFromCookie = () => {
    return getIntCookie("user");
  };

  const requireResetPassword = computed(() => {
    if (!currentUserId.value) {
      return false;
    }
    return useLocalStorage<boolean>(
      `${currentUserId.value}.require_reset_password`,
      false
    ).value;
  });

  const setRequireResetPassword = (requireResetPassword: boolean) => {
    if (currentUserId.value) {
      const needResetPasswordCache = useLocalStorage<boolean>(
        `${currentUserId.value}.require_reset_password`,
        false
      );
      needResetPasswordCache.value = requireResetPassword;
    }
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
      return router.push({
        name: AUTH_MFA_MODULE,
        query: {
          mfaTempToken: data.mfaTempToken,
          redirect: redirectUrl,
        },
      });
    }

    await restoreUser();
    setRequireResetPassword(data.requireResetPassword);

    await useSettingV1Store().getOrFetchSettingByName(
      "bb.workspace.profile",
      /* silent */ true
    );

    if (!showLoginModal.value) {
      let nextPage = redirectUrl || "/";
      const mode = useAppFeature("bb.feature.database-change-mode");

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
    showLoginModal.value = false;
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
      showLoginModal.value = false;
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

  const restoreUser = async () => {
    currentUserId.value = getUserIdFromCookie();
    if (currentUserId.value) {
      await useUserStore().getOrFetchUserByIdentifier(
        String(currentUserId.value),
        true // silent
      );
    }
  };

  return {
    currentUser,
    currentUserId,
    isLoggedIn,
    getUserIdFromCookie,
    login,
    signup,
    logout,
    restoreUser,
    requireResetPassword,
    setRequireResetPassword,
    showLoginModal,
  };
});

export const useCurrentUserV1 = () => {
  const authStore = useAuthStore();
  return computed(() => authStore.currentUser);
};

export const useIsLoggedIn = () => {
  const store = useAuthStore();
  return computed(() => store.isLoggedIn() && store.currentUser.name !== "");
};
