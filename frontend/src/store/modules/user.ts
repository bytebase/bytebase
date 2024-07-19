import { isEqual, isUndefined, orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { authServiceClient } from "@/grpcweb";
import {
  type PrincipalType,
  PresetRoleType,
  PRESET_WORKSPACE_ROLES,
  ALL_USERS_USER_EMAIL,
  allUsersUser,
  SYSTEM_BOT_USER_NAME,
} from "@/types";
import type { ComposedUser } from "@/types";
import type { UpdateUserRequest, User } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { roleListInIAM } from "@/utils";
import { userNamePrefix, getUserEmailFromIdentifier } from "./v1/common";
import { usePermissionStore } from "./v1/permission";
import { useWorkspaceV1Store } from "./v1/workspace";

export const useUserStore = defineStore("user", () => {
  const userMapByName = ref<Map<string, ComposedUser>>(new Map());
  const workspaceStore = useWorkspaceV1Store();

  const setUser = (user: User) => {
    const roles = roleListInIAM(workspaceStore.workspaceIamPolicy, user.email);
    const composedUser: ComposedUser = {
      ...user,
      roles,
    };
    userMapByName.value.set(user.name, composedUser);

    // invalid permission cache
    usePermissionStore().invalidCacheByUser(composedUser);
    return composedUser;
  };

  const userList = computed(() => {
    return orderBy(
      Array.from(userMapByName.value.values()),
      [
        (user) =>
          user.userType === UserType.SYSTEM_BOT
            ? 0
            : user.userType === UserType.SERVICE_ACCOUNT
              ? 1
              : 2,
        (user) => user.roles.includes(PresetRoleType.WORKSPACE_ADMIN),
        (user) => user.roles.includes(PresetRoleType.WORKSPACE_DBA),
      ],
      ["asc", "desc", "desc"]
    );
  });
  // The active user list and exclude allUsers.
  const activeUserList = computed(() => {
    return userList.value.filter(
      (user) =>
        user.state === State.ACTIVE && user.email !== ALL_USERS_USER_EMAIL
    );
  });

  const systemBotUser = computed(() => {
    return activeUserList.value.find(
      (user) => user.name === SYSTEM_BOT_USER_NAME
    );
  });

  const workspaceLevelProjectMembers = computed(() => {
    return activeUserList.value.filter((user) =>
      user.roles.some((role) => !PRESET_WORKSPACE_ROLES.includes(role))
    );
  });

  const fetchUserList = async () => {
    const { users } = await authServiceClient.listUsers({
      showDeleted: true,
    });
    const response: ComposedUser[] = [];
    for (const user of users) {
      response.push(setUser(user));
    }
    return response;
  };
  const fetchUser = async (name: string, silent = false) => {
    const user = await authServiceClient.getUser(
      {
        name,
      },
      {
        silent,
      }
    );
    return setUser(user);
  };
  const createUser = async (user: ComposedUser) => {
    const createdUser = await authServiceClient.createUser({
      user,
    });
    await workspaceStore.patchIamPolicy({
      member: `user:${createdUser.email}`,
      roles: user.roles,
    });

    return setUser(createdUser);
  };
  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByName(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }
    const user = await authServiceClient.updateUser(updateUserRequest);
    return setUser(user);
  };
  const updateUserRoles = async (user: ComposedUser) => {
    await workspaceStore.patchIamPolicy({
      member: `user:${user.email}`,
      roles: user.roles,
    });
    return setUser(user);
  };
  const getOrFetchUserByName = async (name: string, silent = false) => {
    const cachedData = userMapByName.value.get(name);
    if (cachedData) {
      return cachedData;
    }
    const user = await fetchUser(name, silent);
    return setUser(user);
  };
  const getUserByName = (name: string) => {
    return userMapByName.value.get(name);
  };
  const getOrFetchUserById = async (uid: string, silent = false) => {
    return await getOrFetchUserByName(getUserNameWithUserId(uid), silent);
  };
  const getUserById = (uid: string) => {
    return getUserByName(getUserNameWithUserId(uid));
  };
  const getUserByIdentifier = (identifier: string) => {
    return getUserByEmail(getUserEmailFromIdentifier(identifier));
  };
  const getUserByEmail = (email: string) => {
    if (email === ALL_USERS_USER_EMAIL) {
      return allUsersUser();
    }
    return [...userMapByName.value.values()].find(
      (user) => user.email === email
    );
  };
  const archiveUser = async (user: User) => {
    await authServiceClient.deleteUser({
      name: user.name,
    });
    user.state = State.DELETED;
    return user;
  };
  const restoreUser = async (user: User) => {
    const restoredUser = await authServiceClient.undeleteUser({
      name: user.name,
    });
    return setUser(restoredUser);
  };

  return {
    userMapByName,
    userList,
    activeUserList,
    systemBotUser,
    workspaceLevelProjectMembers,
    fetchUserList,
    fetchUser,
    createUser,
    updateUser,
    updateUserRoles,
    getOrFetchUserByName,
    getUserByName,
    getOrFetchUserById,
    getUserById,
    getUserByIdentifier,
    getUserByEmail,
    archiveUser,
    restoreUser,
  };
});

export const extractUserEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:user:|users\/)(.+)$/);
  return matches?.[1] ?? emailResource;
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
  if (!isUndefined(update.password) && update.password !== "") {
    updateMask.push("password");
  }
  if (!isUndefined(update.phone) && !isEqual(origin.phone, update.phone)) {
    updateMask.push("phone");
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
