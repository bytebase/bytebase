import axios from "axios";
import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { authServiceClient } from "@/grpcweb";
import { SignupInfo, ActivateInfo, unknownUser } from "@/types";
import {
  LoginRequest,
  LoginResponse,
  User,
  UserType,
} from "@/types/proto/v1/auth_service";
import { getIntCookie } from "@/utils";
import { convertUserToPrincipal, useUserStore } from ".";

interface AuthState {
  currentUser: User;
}

export const useAuthStore = defineStore("auth_v1", {
  state: (): AuthState => ({
    currentUser: unknownUser(),
  }),
  actions: {
    isLoggedIn: () => {
      return getIntCookie("user") != undefined;
    },
    async login(request: Partial<LoginRequest>) {
      const {
        data: { mfaTempToken },
      } = await axios.post<LoginResponse>("/v1/auth/login", request);
      if (mfaTempToken) {
        return mfaTempToken;
      }

      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await useUserStore().getOrFetchUserById(
          String(userId)
        );
        this.currentUser = loggedInUser;
      }
    },
    async signup(signupInfo: SignupInfo) {
      await authServiceClient.createUser({
        user: {
          email: signupInfo.email,
          title: signupInfo.name,
          password: signupInfo.password,
          userType: UserType.USER,
        },
      });
      await this.login({
        email: signupInfo.email,
        password: signupInfo.password,
        web: true,
      });
    },
    async logout() {
      const unknown = unknownUser();
      try {
        await axios.post("/v1/auth/logout");
      } finally {
        this.currentUser = unknown;
      }
      return unknown;
    },
    async activate(activateInfo: ActivateInfo) {
      const activatedUser = (
        await axios.post("/api/auth/activate", {
          data: { type: "activateInfo", attributes: activateInfo },
        })
      ).data.data;

      // Refresh the corresponding user.
      const user = await useUserStore().getOrFetchUserById(
        String(activatedUser.id)
      );
      this.currentUser = user;
      return user;
    },
    async restoreUser() {
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await useUserStore().getOrFetchUserById(
          String(userId),
          true // silent
        );
        this.currentUser = loggedInUser;
        return loggedInUser;
      }
      return unknownUser();
    },
    async refreshUserIfNeeded(name: string) {
      if (name === this.currentUser.name) {
        const refreshedUser = await useUserStore().fetchUser(
          name,
          true // silent
        );
        if (!isEqual(refreshedUser, this.currentUser)) {
          this.currentUser = refreshedUser;
        }
      }
    },
  },
});

export const useCurrentUser = () => {
  const authStore = useAuthStore();

  return computed(() => {
    return convertUserToPrincipal(authStore.currentUser);
  });
};

export const useCurrentUserV1 = () => {
  const authStore = useAuthStore();
  return computed(() => authStore.currentUser);
};

export const useIsLoggedIn = () => {
  const store = useAuthStore();
  return computed(() => store.isLoggedIn() && store.currentUser.name !== "");
};
