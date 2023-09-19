import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { authServiceClient } from "@/grpcweb";
import { Principal, PrincipalType, RoleType } from "@/types";
import {
  UpdateUserRequest,
  User,
  userRoleToJSON,
  UserType,
} from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { extractUserUID } from "@/utils";
import {
  getUserId,
  userNamePrefix,
  getUserEmailFromIdentifier,
} from "./v1/common";

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
    activeUserList(state) {
      const list = Array.from(state.userMapByName.values()).filter(
        (user) => user.state === State.ACTIVE
      );
      list.sort((a, b) => {
        return (
          parseInt(extractUserUID(a.name), 10) -
          parseInt(extractUserUID(b.name), 10)
        );
      });
      return list;
    },
  },
  actions: {
    async fetchUserList() {
      const { users } = await authServiceClient.listUsers({
        showDeleted: true,
      });
      for (const user of users) {
        this.userMapByName.set(user.name, user);
      }
      return users;
    },
    async fetchUser(name: string, silent = false) {
      const user = await authServiceClient.getUser(
        {
          name,
        },
        {
          silent,
        }
      );
      this.userMapByName.set(user.name, user);
      return user;
    },
    async createUser(user: User) {
      const createdUser = await authServiceClient.createUser({
        user,
      });
      this.userMapByName.set(createdUser.name, createdUser);
      return createdUser;
    },
    async updateUser(updateUserRequest: UpdateUserRequest) {
      const name = updateUserRequest.user?.name || "";
      const originData = await this.getOrFetchUserByName(name);
      if (!originData) {
        throw new Error(`user with name ${name} not found`);
      }
      const user = await authServiceClient.updateUser(updateUserRequest);
      this.userMapByName.set(user.name, user);
      return user;
    },
    async getOrFetchUserByName(name: string, silent = false) {
      const cachedData = this.userMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const user = await this.fetchUser(name, silent);
      this.userMapByName.set(user.name, user);
      return user;
    },
    getUserByName(name: string) {
      return this.userMapByName.get(name);
    },
    async getOrFetchUserById(uid: string, silent = false) {
      return await this.getOrFetchUserByName(
        getUserNameWithUserId(uid),
        silent
      );
    },
    getUserById(uid: string) {
      return this.userMapByName.get(getUserNameWithUserId(uid));
    },
    getUserByIdentifier(identifier: string) {
      return this.getUserByEmail(getUserEmailFromIdentifier(identifier));
    },
    getUserByEmail(email: string) {
      return [...this.userMapByName.values()].find(
        (user) => user.email === email
      );
    },
    async archiveUser(user: User) {
      await authServiceClient.deleteUser({
        name: user.name,
      });
      user.state = State.DELETED;
      return user;
    },
    async restoreUser(user: User) {
      const restoredUser = await authServiceClient.undeleteUser({
        name: user.name,
      });
      this.userMapByName.set(restoredUser.name, restoredUser);
      return restoredUser;
    },
  },
});

export const extractUserEmail = (emailResource: string) => {
  const matches = emailResource.match(/^user:(.+)$/);
  return matches?.[1] ?? "";
};

export const getUserNameWithUserId = (userUID: string) => {
  return `${userNamePrefix}${userUID}`;
};

export const getUpdateMaskFromUsers = (
  origin: User,
  update: User | Partial<User>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (!isUndefined(update.email) && !isEqual(origin.email, update.email)) {
    updateMask.push("email");
  }
  if (!isUndefined(update.password)) {
    updateMask.push("password");
  }
  if (
    !isUndefined(update.userRole) &&
    !isEqual(origin.userRole, update.userRole)
  ) {
    updateMask.push("role");
  }
  return updateMask;
};

export const convertUserTypeToPrincipalType = (
  userType: UserType
): PrincipalType => {
  if (userType === UserType.SYSTEM_BOT) {
    return "SYSTEM_BOT";
  } else if (userType === UserType.SERVICE_ACCOUNT) {
    return "SERVICE_ACCOUNT";
  } else {
    return "END_USER";
  }
};

export const convertUserToPrincipal = (user: User): Principal => {
  const userRole = userRoleToJSON(user.userRole) as RoleType;
  const userType = convertUserTypeToPrincipalType(user.userType);

  return {
    id: getUserId(user.name),
    name: user.title,
    email: user.email,
    role: userRole,
    type: userType,
    serviceKey: user.password,
  };
};
