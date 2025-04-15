import { computedAsync } from "@vueuse/core";
import { isEqual, isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { userServiceClient } from "@/grpcweb";
import {
  isValidProjectName,
  allUsersUser,
  SYSTEM_BOT_USER_NAME,
  isValidUserName,
  unknownUser,
} from "@/types";
import { State, stateToJSON } from "@/types/proto/v1/common";
import type { UpdateUserRequest, User } from "@/types/proto/v1/user_service";
import { UserType, userTypeToJSON } from "@/types/proto/v1/user_service";
import { ensureUserFullName } from "@/utils";
import { useActuatorV1Store } from "./v1/actuator";
import { userNamePrefix, extractUserId } from "./v1/common";
import { usePermissionStore } from "./v1/permission";

export interface UserFilter {
  query?: string;
  types?: UserType[];
  project?: string;
  state?: State;
}

const getListUserFilter = (params: UserFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.matches("${search}") || email.matches("${search}"))`);
  }
  if (params.types) {
    filter.push(
      `user_type in [${params.types.map((t) => `"${userTypeToJSON(t)}"`).join(", ")}]`
    );
  }
  if (isValidProjectName(params.project)) {
    filter.push(`project == "${params.project}"`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${stateToJSON(params.state)}"`);
  }

  return filter.join(" && ");
};

export const useUserStore = defineStore("user", () => {
  const actuatorStore = useActuatorV1Store();
  const allUser = computed(() => allUsersUser());
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

  const fetchCurrentUser = async () => {
    const user = await userServiceClient.getCurrentUser({});
    setUser(user);
    return user;
  };

  const fetchUserList = async (params: {
    pageSize: number;
    pageToken?: string;
    filter?: UserFilter;
  }): Promise<{
    users: User[];
    nextPageToken: string;
  }> => {
    const response = await userServiceClient.listUsers({
      ...params,
      filter: getListUserFilter(params.filter ?? {}),
      showDeleted: params.filter?.state === State.DELETED ? true : false,
    });
    for (const user of response.users) {
      setUser(user);
    }
    return response;
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
    await actuatorStore.fetchServerInfo();
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
    await actuatorStore.fetchServerInfo();
    return user;
  };

  const restoreUser = async (user: User) => {
    const restoredUser = await userServiceClient.undeleteUser({
      name: user.name,
    });
    await actuatorStore.fetchServerInfo();
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

  return {
    allUser,
    systemBotUser,
    fetchCurrentUser,
    fetchUserList,
    createUser,
    updateUser,
    getOrFetchUserByIdentifier,
    getUserByIdentifier,
    archiveUser,
    restoreUser,
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
