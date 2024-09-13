import { useLocalStorage } from "@vueuse/core";
import axios from "axios";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { restartAppRoot } from "@/AppRootContext";
import { authServiceClient } from "@/grpcweb";
import { unknownUser } from "@/types";
import type { LoginRequest, User } from "@/types/proto/v1/auth_service";
import { LoginResponse } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { getIntCookie } from "@/utils";
import { useUserStore } from ".";

export const useAuthStore = defineStore("auth_v1", () => {
  const userStore = useUserStore();
  const currentUserId = ref<number | undefined>();

  const currentUser = computed(() => {
    if (currentUserId.value) {
      return userStore.getUserById(`${currentUserId.value}`) ?? unknownUser();
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

  const login = async (
    request: Partial<LoginRequest>
  ): Promise<LoginResponse> => {
    const { data } = await axios.post<LoginResponse>("/v1/auth/login", request);
    if (data.mfaTempToken) {
      return data;
    }

    await restoreUser();
    setRequireResetPassword(data.requireResetPassword);

    return data;
  };

  const signup = async (request: Partial<User>) => {
    await authServiceClient.createUser({
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
      currentUserId.value = undefined;
      restartAppRoot();
    }
  };

  const restoreUser = async () => {
    currentUserId.value = getUserIdFromCookie();
    if (currentUserId.value) {
      await useUserStore().getOrFetchUserById(
        String(currentUserId.value),
        true // silent
      );
    }
  };

  const refreshUserIfNeeded = async (name: string) => {
    if (name === currentUser.value.name) {
      await useUserStore().fetchUser(
        name,
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
    refreshUserIfNeeded,
    requireResetPassword,
    setRequireResetPassword,
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
