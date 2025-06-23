import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { computedAsync } from "@vueuse/core";
import { isEqual, isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { userServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import {
  isValidProjectName,
  allUsersUser,
  SYSTEM_BOT_USER_NAME,
  isValidUserName,
  unknownUser,
  userBindingPrefix,
} from "@/types";
import { State, stateToJSON } from "@/types/proto/v1/common";
import type { UpdateUserRequest, User } from "@/types/proto/v1/user_service";
import { UserType, userTypeToJSON } from "@/types/proto/v1/user_service";
import {
  GetUserRequestSchema,
  ListUsersRequestSchema,
  CreateUserRequestSchema,
  UpdateUserRequestSchema,
  DeleteUserRequestSchema,
  UndeleteUserRequestSchema,
  BatchGetUsersRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { convertNewUserToOld, convertOldUserToNew } from "@/utils/v1/user-conversions";
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
    const response = await userServiceClientConnect.getCurrentUser({});
    const user = convertNewUserToOld(response);
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
    const request = create(ListUsersRequestSchema, {
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      filter: getListUserFilter(params.filter ?? {}),
      showDeleted: params.filter?.state === State.DELETED ? true : false,
    });
    const response = await userServiceClientConnect.listUsers(request);
    const users = response.users.map(convertNewUserToOld);
    for (const user of users) {
      setUser(user);
    }
    return {
      users,
      nextPageToken: response.nextPageToken,
    };
  };

  const fetchUser = async (name: string, silent = false) => {
    const request = create(GetUserRequestSchema, {
      name,
    });
    const response = await userServiceClientConnect.getUser(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const user = convertNewUserToOld(response);
    return setUser(user);
  };

  const createUser = async (user: User) => {
    const newUser = convertOldUserToNew(user);
    const request = create(CreateUserRequestSchema, {
      user: newUser,
    });
    const response = await userServiceClientConnect.createUser(request);
    const createdUser = convertNewUserToOld(response);
    await actuatorStore.fetchServerInfo();
    return setUser(createdUser);
  };

  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByIdentifier(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }
    const newUser = convertOldUserToNew(updateUserRequest.user!);
    const request = create(UpdateUserRequestSchema, {
      user: newUser,
      updateMask: updateUserRequest.updateMask ? { paths: updateUserRequest.updateMask } : undefined,
      otpCode: updateUserRequest.otpCode,
      regenerateTempMfaSecret: updateUserRequest.regenerateTempMfaSecret,
      regenerateRecoveryCodes: updateUserRequest.regenerateRecoveryCodes,
    });
    const response = await userServiceClientConnect.updateUser(request);
    const user = convertNewUserToOld(response);
    return setUser(user);
  };

  const archiveUser = async (user: User) => {
    const request = create(DeleteUserRequestSchema, {
      name: user.name,
    });
    await userServiceClientConnect.deleteUser(request);
    user.state = State.DELETED;
    await actuatorStore.fetchServerInfo();
    return user;
  };

  const restoreUser = async (user: User) => {
    const request = create(UndeleteUserRequestSchema, {
      name: user.name,
    });
    const response = await userServiceClientConnect.undeleteUser(request);
    const restoredUser = convertNewUserToOld(response);
    await actuatorStore.fetchServerInfo();
    return setUser(restoredUser);
  };

  const batchGetUsers = async (userNameList: string[]) => {
    const distinctList = uniq(userNameList)
      .filter(
        (name) =>
          Boolean(name) &&
          (name.startsWith(userNamePrefix) ||
            name.startsWith(userBindingPrefix))
      )
      .map((name) => ensureUserFullName(name))
      .filter(
        (name) =>
          isValidUserName(name) && getUserByIdentifier(name) === undefined
      );
    if (distinctList.length === 0) {
      return [];
    }
    const request = create(BatchGetUsersRequestSchema, {
      names: distinctList,
    });
    const response = await userServiceClientConnect.batchGetUsers(request);
    const users = response.users.map(convertNewUserToOld);
    for (const user of users) {
      setUser(user);
    }
    return users;
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
    batchGetUsers,
    getOrFetchUserByIdentifier,
    getUserByIdentifier,
    archiveUser,
    restoreUser,
  };
});

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
