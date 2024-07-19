import axios from "axios";
import { defineStore } from "pinia";
import { computed } from "vue";
import { authServiceClient } from "@/grpcweb";
import type { SignupInfo, ActivateInfo } from "@/types";
import { unknownUser } from "@/types";
import type {
  LoginRequest,
  LoginResponse,
} from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { getIntCookie } from "@/utils";
import { useUserStore } from ".";

export const useAuthStore = defineStore("auth_v1", () => {
  const userStore = useUserStore();

  const currentUser = computed(() => {
    const userId = getIntCookie("user");
    if (userId) {
      return userStore.getUserById(`${userId}`) ?? unknownUser();
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

    const userId = getIntCookie("user");
    if (userId) {
      await useUserStore().getOrFetchUserById(String(userId));
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
  };

  const logout = async () => {
    try {
      await axios.post("/v1/auth/logout");
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

    // Refresh the corresponding user.
    const user = await useUserStore().getOrFetchUserById(
      String(activatedUser.id)
    );
    return user;
  };

  const restoreUser = async () => {
    const userId = getIntCookie("user");
    if (userId) {
      await useUserStore().getOrFetchUserById(
        String(userId),
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
