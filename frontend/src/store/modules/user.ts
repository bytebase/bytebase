import { computedAsync } from "@vueuse/core";
import { isEqual, isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { userServiceClient } from "@/grpcweb";
import {
  allUsersUser,
  SYSTEM_BOT_USER_NAME,
  isValidUserName,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type {
  UpdateUserRequest,
  User,
  StatUsersResponse_StatUser,
} from "@/types/proto/v1/user_service";
import { UserType } from "@/types/proto/v1/user_service";
import { ensureUserFullName } from "@/utils";
import { userNamePrefix, extractUserId } from "./v1/common";
import { usePermissionStore } from "./v1/permission";

export const useUserStore = defineStore("user", () => {
  const allUser = computed(() => allUsersUser());
  const userStats = ref<StatUsersResponse_StatUser[]>([]);
  const userRequestCache = new Map<string, Promise<User>>();

  const userMapByName = ref<Map<string, User>>(
    new Map([[allUser.value.name, allUser.value]])
  );

  const setUser = (user: User) => {
    userMapByName.value.set(user.name, user);
    usePermissionStore().invalidCacheByUser(user);
    return user;
  };

  const systemBotUser = computedAsync(() => {
    return getOrFetchUserByIdentifier(SYSTEM_BOT_USER_NAME);
  });

  const fetchUserList = async (params: {
    pageSize: number;
    pageToken?: string;
    filter?: string;
    showDeleted?: boolean;
  }): Promise<{
    users: User[];
    nextPageToken: string;
  }> => {
    const response = await userServiceClient.listUsers(params);
    for (const user of response.users) {
      setUser(user);
    }
    return response;
  };

  const refreshUserStat = async () => {
    const { stats } = await userServiceClient.statUsers({});
    userStats.value = stats;
  };

  const fetchUser = async (name: string, silent = false) => {
    const user = await userServiceClient.getUser(
      {
        name,
      },
      {
        silent,
      }
    );
    return setUser(user);
  };

  const createUser = async (user: User) => {
    const createdUser = await userServiceClient.createUser({
      user,
    });
    await refreshUserStat();
    return setUser(createdUser);
  };

  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByIdentifier(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }
    const user = await userServiceClient.updateUser(updateUserRequest);
    return setUser(user);
  };

  const archiveUser = async (user: User) => {
    await userServiceClient.deleteUser({
      name: user.name,
    });
    user.state = State.DELETED;
    await refreshUserStat();
    return user;
  };

  const restoreUser = async (user: User) => {
    const restoredUser = await userServiceClient.undeleteUser({
      name: user.name,
    });
    await refreshUserStat();
    return setUser(restoredUser);
  };

  const getOrFetchUserByIdentifier = async (
    identifier: string,
    silent = true
  ) => {
    const user = getUserByIdentifier(identifier);
    if (user) {
      return user;
    }

    const fullname = ensureUserFullName(identifier);
    if (!isValidUserName(fullname)) {
      return unknownUser();
    }
    const cached = userRequestCache.get(fullname);
    if (cached) return cached;
    const request = fetchUser(fullname, silent).then((user) => setUser(user));
    userRequestCache.set(fullname, request);
    return request;
  };

  const getUserByIdentifier = (identifier: string) => {
    if (!identifier) {
      return;
    }
    const id = extractUserId(identifier);
    if (Number.isNaN(Number(id))) {
      return [...userMapByName.value.values()].find(
        (user) => user.email === id
      );
    }
    return userMapByName.value.get(`${userNamePrefix}${id}`);
  };

  const activeUserCountWithoutBot = computed(() => {
    return userStats.value.reduce((count, stat) => {
      if (
        stat.state === State.ACTIVE &&
        stat.userType !== UserType.SYSTEM_BOT
      ) {
        count += stat.count;
      }
      return count;
    }, 0);
  });

  return {
    allUser,
    userStats,
    systemBotUser,
    fetchUserList,
    refreshUserStat,
    createUser,
    updateUser,
    getOrFetchUserByIdentifier,
    getUserByIdentifier,
    archiveUser,
    restoreUser,
    activeUserCountWithoutBot,
  };
});

export const batchGetOrFetchUsers = async (userNameList: string[]) => {
  const userStore = useUserStore();
  const distinctList = uniq(userNameList);
  await Promise.all(
    distinctList.map((userName) => {
      if (!isValidUserName(userName)) {
        return;
      }
      return userStore.getOrFetchUserByIdentifier(userName, true /* silent */);
    })
  );
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
