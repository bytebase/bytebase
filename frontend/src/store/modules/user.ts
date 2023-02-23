import { defineStore } from "pinia";
import { authServiceClient } from "@/grpcweb";
import {
  User,
  userRoleToJSON,
  userTypeToJSON,
} from "@/types/proto/v1/auth_service";
import { isEqual, isUndefined } from "lodash-es";
import { getUserId, userNamePrefix } from "./v1/common";
import { Principal, PrincipalType, RoleType } from "@/types";

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
    async fetchUserList() {
      const { users } = await authServiceClient().listUsers({});
      for (const user of users) {
        this.userMapByName.set(user.name, user);
      }
      return users;
    },
    async fetchUser(name: string) {
      const user = await authServiceClient().getUser({
        name,
      });
      this.userMapByName.set(user.name, user);
      return user;
    },
    async createUser(create: User) {
      const user = await authServiceClient().createUser({
        user: create,
      });
      this.userMapByName.set(user.name, user);
      return user;
    },
    async updateUser(update: Partial<User>) {
      const originData = await this.getOrFetchUserByName(update.name || "");
      if (!originData) {
        throw new Error(`user with name ${update.name} not found`);
      }

      const user = await authServiceClient().updateUser({
        user: update,
        updateMask: getUpdateMaskFromUsers(originData, update),
      });
      this.userMapByName.set(user.name, user);
      return user;
    },
    async getOrFetchUserByName(name: string) {
      const cachedData = this.userMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const user = await authServiceClient().getUser({
        name,
      });
      this.userMapByName.set(user.name, user);
      return user;
    },
    getUserByName(name: string) {
      return this.userMapByName.get(name);
    },
    async getOrFetchUserById(id: number) {
      const name = `${userNamePrefix}${id}`;
      return await this.getOrFetchUserByName(name);
    },
    getUserById(id: number) {
      const name = `${userNamePrefix}${id}`;
      return this.userMapByName.get(name);
    },
  },
});

const getUpdateMaskFromUsers = (
  origin: User,
  update: Partial<User>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("user.title");
  }
  if (!isUndefined(update.email) && !isEqual(origin.email, update.email)) {
    updateMask.push("user.email");
  }
  if (!isUndefined(update.password)) {
    updateMask.push("user.password");
  }
  if (
    !isUndefined(update.userRole) &&
    !isEqual(origin.userRole, update.userRole)
  ) {
    updateMask.push("user.role");
  }
  return updateMask;
};

export const convertUserToPrincipal = (user: User): Principal => {
  const userRole = userRoleToJSON(user.userRole) as RoleType;
  const userType = userTypeToJSON(user.userType) as PrincipalType;

  return {
    id: getUserId(user.name),
    name: user.title,
    email: user.email,
    role: userRole,
    type: userType,
    serviceKey: user.password,
  };
};
