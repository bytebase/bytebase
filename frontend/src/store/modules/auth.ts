import { defineStore } from "pinia";
import axios from "axios";
import { isEqual } from "lodash-es";
import { computed } from "vue";
import { SignupInfo, ActivateInfo } from "@/types";
import { getIntCookie } from "@/utils";
import { authServiceClient } from "@/grpcweb";
import { LoginRequest, User, UserType } from "@/types/proto/v1/auth_service";
import { convertUserToPrincipal, useUserStore } from ".";
import { unknown } from "@/utils/common";

interface AuthState {
  currentUser: User;
}

export const useAuthStore = defineStore("auth_v1", {
  state: (): AuthState => ({
    currentUser: User.fromPartial({}),
  }),
  actions: {
    isLoggedIn: () => {
      return getIntCookie("user") != undefined;
    },
    async login(request: Partial<LoginRequest>) {
      await axios.post("/v1/auth/login", request);
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await useUserStore().getOrFetchUserById(userId);
        this.currentUser = loggedInUser;
        return loggedInUser;
      }
      return unknown("USER");
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
      const user = await this.login({
        email: signupInfo.email,
        password: signupInfo.password,
        web: true,
      });
      return user;
    },
    async logout() {
      const unknownUser = unknown("USER");
      try {
        await axios.post("/v1/auth/logout");
      } finally {
        this.currentUser = unknownUser;
      }
      return unknownUser;
    },
    async activate(activateInfo: ActivateInfo) {
      const activatedUser = (
        await axios.post("/api/auth/activate", {
          data: { type: "activateInfo", attributes: activateInfo },
        })
      ).data.data;

      // Refresh the corresponding user.
      const user = await useUserStore().getOrFetchUserById(activatedUser.id);
      this.currentUser = user;
      return user;
    },
    async restoreUser() {
      const userId = getIntCookie("user");
      if (userId) {
        const loggedInUser = await useUserStore().getOrFetchUserById(userId);
        this.currentUser = loggedInUser;
        return loggedInUser;
      }
      return unknown("USER");
    },
    async refreshUserIfNeeded(name: string) {
      if (name === this.currentUser.name) {
        const refreshedUser = await useUserStore().getOrFetchUserByName(name);
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

export const useIsLoggedIn = () => {
  const store = useAuthStore();
  return computed(() => store.isLoggedIn() && store.currentUser.name !== "");
};
