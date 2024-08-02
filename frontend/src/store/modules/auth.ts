import axios from "axios";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { authServiceClient } from "@/grpcweb";
import type { SignupInfo, ActivateInfo } from "@/types";
import { unknownUser } from "@/types";
import type {
  LoginRequest,
  LoginResponse,
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

  const signup = async (signupInfo: SignupInfo) => {
    await authServiceClient.createUser({
      user: {
        email: signupInfo.email,
        title: signupInfo.name,
        password: signupInfo.password,
        userType: UserType.USER,
      },
    });
    await login({
      email: signupInfo.email,
      password: signupInfo.password,
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

  const activate = async (activateInfo: ActivateInfo) => {
    const activatedUser = (
      await axios.post("/api/auth/activate", {
        data: { type: "activateInfo", attributes: activateInfo },
      })
    ).data.data;

    currentUserId.value = activatedUser.id;

    // Refresh the corresponding user.
    const user = await useUserStore().getOrFetchUserById(
      String(activatedUser.id)
    );
    return user;
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
    activate,
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
