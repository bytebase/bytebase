import axios from "axios";
import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { authServiceClient } from "@/grpcweb";
import { ActivateInfo, SignupInfo } from "@/types";
import { LoginRequest, User } from "@/types/proto/v1/auth_service";
import { getIntCookie } from "@/utils";
import { unknown } from "@/utils/common";
import { useUserStore } from "../user";
import { getUserId } from "./common";

interface AuthState {
  currentUser: User;
}

export const useAuthV1Store = defineStore("auth_v1", {
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
      await authServiceClient().createUser({
        user: {
          email: signupInfo.email,
          title: signupInfo.name,
          password: signupInfo.password,
        },
      });
      const user = await this.login({
        email: signupInfo.email,
        password: signupInfo.password,
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
  const authStore = useAuthV1Store();

  return computed(() => {
    return {
      ...authStore.currentUser,
      id: getUserId(authStore.currentUser.name),
    };
  });
};

export const useIsLoggedIn = () => {
  const store = useAuthV1Store();
  return computed(() => store.isLoggedIn() && store.currentUser.name !== "");
};
