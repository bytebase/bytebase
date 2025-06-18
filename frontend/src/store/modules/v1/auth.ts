import { create } from "@bufbuild/protobuf";
import { useLocalStorage } from "@vueuse/core";
import { uniqueId } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { authServiceClientConnect, userServiceClient } from "@/grpcweb";
import { router } from "@/router";
import {
  AUTH_SIGNIN_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_MFA_MODULE,
} from "@/router/auth";
import { SETUP_MODULE } from "@/router/setup";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import {
  useAppFeature,
  useActuatorV1Store,
  useSettingV1Store,
  useUserStore,
} from "@/store";
import { UNKNOWN_USER_NAME, unknownUser } from "@/types";
import {
  LoginRequestSchema,
  type LoginRequest,
} from "@/types/proto-es/v1/auth_service_pb";
import {
  DatabaseChangeMode,
  Setting_SettingName,
} from "@/types/proto/v1/setting_service";
import { User, UserType } from "@/types/proto/v1/user_service";

export const useAuthStore = defineStore("auth_v1", () => {
  const userStore = useUserStore();
  const authSessionKey = ref<string>(uniqueId());
  const actuatorStore = useActuatorV1Store();
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

  const login = async (request: LoginRequest, redirect: string = "") => {
    const resp = await authServiceClientConnect.login(request);
    const redirectUrl = redirect || getRedirectQuery() || "/";
    if (resp.mfaTempToken) {
      unauthenticatedOccurred.value = false;
      return router.push({
        name: AUTH_MFA_MODULE,
        query: {
          mfaTempToken: resp.mfaTempToken,
          redirect: redirectUrl,
        },
      });
    }

    await fetchCurrentUser();
    setRequireResetPassword(resp.requireResetPassword);

    await useSettingV1Store().getOrFetchSettingByName(
      Setting_SettingName.WORKSPACE_PROFILE,
      true // silent
    );

    // After user login, we need to reset the auth session key.
    authSessionKey.value = uniqueId();
    if (!unauthenticatedOccurred.value) {
      if (actuatorStore.needAdminSetup) {
        await actuatorStore.fetchServerInfo();
        actuatorStore.onboardingState.isOnboarding = true;
        return router.replace({
          name: SETUP_MODULE,
        });
      }
      const mode = useAppFeature("bb.feature.database-change-mode");
      let nextPage = redirectUrl;
      if (mode.value === DatabaseChangeMode.EDITOR) {
        const route = router.resolve({
          name: SQL_EDITOR_HOME_MODULE,
        });
        nextPage = route.fullPath;
      }
      if (resp.requireResetPassword) {
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
    await login(
      create(LoginRequestSchema, {
        email: request.email,
        password: request.password,
        web: true,
      })
    );
  };

  const logout = async () => {
    try {
      await authServiceClientConnect.logout({});
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
      const user = await userStore.fetchCurrentUser();
      currentUserName.value = user.name;
      return user;
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
