import { defineStore } from "pinia";
import { authServiceClient } from "@/grpcweb";
import { User } from "@/types/proto/v1/auth_service";

interface UserState {
  userMapByName: Map<string, User>;
}

export const useUserStore = defineStore("user", {
  state: (): UserState => ({
    userMapByName: new Map(),
  }),
  getters: {
    userList(state) {
      return Array.from(state.userMapByName.values());
    },
  },
  actions: {
    async fetchUser(name: string) {
      const user = await authServiceClient().getUser({
        name,
      });
      this.userMapByName.set(user.name, user);
      return user;
    },
  },
});
