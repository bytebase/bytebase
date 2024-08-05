import axios from "axios";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { authServiceClient } from "@/grpcweb";
import { unknownUser } from "@/types";
import type {
  LoginRequest,
  LoginResponse,
  User,
} from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { getIntCookie } from "@/utils";
import { useUserStore, useWorkspaceV1Store } from ".";

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
    return getIntCookie("user") != undefined;
  };

  const login = async (request: Partial<LoginRequest>) => {
    const {
      data: { mfaTempToken },
    } = await axios.post<LoginResponse>("/v1/auth/login", request);
    if (mfaTempToken) {
      return mfaTempToken;
    }

    currentUserId.value = getIntCookie("user");
    if (currentUserId.value) {
      await useUserStore().getOrFetchUserById(String(currentUserId.value));
      await useWorkspaceV1Store().fetchIamPolicy();
    }
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
    await useWorkspaceV1Store().fetchIamPolicy();
  };

  const logout = async () => {
    try {
      await axios.post("/v1/auth/logout");
      currentUserId.value = undefined;
    } catch {
      // nothing
    }
  };

  const restoreUser = async () => {
    currentUserId.value = getIntCookie("user");
    if (currentUserId.value) {
      await useUserStore().getOrFetchUserById(
        String(currentUserId.value),
        true // silent
      );
      await useWorkspaceV1Store().fetchIamPolicy();
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
    login,
    signup,
    logout,
    restoreUser,
    refreshUserIfNeeded,
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
