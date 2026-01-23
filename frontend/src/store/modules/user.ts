import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { computedAsync } from "@vueuse/core";
import { isEqual, isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { userServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  allUsersUser,
  isValidProjectName,
  isValidUserName,
  SYSTEM_BOT_USER_NAME,
  unknownUser,
  userBindingPrefix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type {
  UpdateUserRequest,
  User,
} from "@/types/proto-es/v1/user_service_pb";
import {
  BatchGetUsersRequestSchema,
  CreateUserRequestSchema,
  DeleteUserRequestSchema,
  GetUserRequestSchema,
  ListUsersRequestSchema,
  UndeleteUserRequestSchema,
  UpdateUserRequestSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName, hasWorkspacePermissionV2 } from "@/utils";
import { serviceAccountToUser, useServiceAccountStore } from "./serviceAccount";
import { useActuatorV1Store } from "./v1/actuator";
import {
  extractUserId,
  serviceAccountNamePrefix,
  userNamePrefix,
  workloadIdentityNamePrefix,
} from "./v1/common";
import { usePermissionStore } from "./v1/permission";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "./workloadIdentity";

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
      `user_type in [${params.types.map((t) => `"${UserType[t]}"`).join(", ")}]`
    );
  }
  if (isValidProjectName(params.project)) {
    filter.push(`project == "${params.project}"`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }

  return filter.join(" && ");
};

export const useUserStore = defineStore("user", () => {
  const actuatorStore = useActuatorV1Store();
  const serviceAccountStore = useServiceAccountStore();
  const workloadIdentityStore = useWorkloadIdentityStore();
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

  const fetchCurrentUser = async () => {
    const response = await userServiceClientConnect.getCurrentUser({});
    setUser(response);
    return response;
  };

  const fetchUserList = async (params: {
    pageSize: number;
    pageToken?: string;
    filter?: UserFilter;
  }): Promise<{
    users: User[];
    nextPageToken: string;
  }> => {
    if (!hasWorkspacePermissionV2("bb.users.list")) {
      return {
        users: [],
        nextPageToken: "",
      };
    }

    const requestedTypes = params.filter?.types ?? [
      UserType.USER,
      UserType.SERVICE_ACCOUNT,
      UserType.WORKLOAD_IDENTITY,
      UserType.SYSTEM_BOT,
    ];
    const showDeleted = params.filter?.state === State.DELETED;
    const allUsers: User[] = [];

    const needsUserService = requestedTypes.some(
      (t) => t === UserType.USER || t === UserType.SYSTEM_BOT
    );
    const needsServiceAccountService = requestedTypes.includes(
      UserType.SERVICE_ACCOUNT
    );
    const needsWorkloadIdentityService = requestedTypes.includes(
      UserType.WORKLOAD_IDENTITY
    );

    const promises: Promise<void>[] = [];

    if (needsUserService) {
      const userPromise = (async () => {
        const request = create(ListUsersRequestSchema, {
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          filter: getListUserFilter(params.filter ?? {}),
          showDeleted,
        });
        const response = await userServiceClientConnect.listUsers(request);
        for (const user of response.users) {
          setUser(user);
          allUsers.push(user);
        }
      })();
      promises.push(userPromise);
    }

    if (needsServiceAccountService) {
      const saPromise = (async () => {
        const response = await serviceAccountStore.listServiceAccounts(
          params.pageSize,
          params.pageToken,
          showDeleted
        );
        for (const sa of response.serviceAccounts) {
          const user = serviceAccountToUser(sa);
          setUser(user);
          allUsers.push(user);
        }
      })();
      promises.push(saPromise);
    }

    if (needsWorkloadIdentityService) {
      const wiPromise = (async () => {
        const response = await workloadIdentityStore.listWorkloadIdentities(
          params.pageSize,
          params.pageToken,
          showDeleted
        );
        for (const wi of response.workloadIdentities) {
          const user = workloadIdentityToUser(wi);
          setUser(user);
          allUsers.push(user);
        }
      })();
      promises.push(wiPromise);
    }

    await Promise.all(promises);

    return {
      users: allUsers,
      nextPageToken: "",
    };
  };

  const fetchUser = async (name: string, silent = false) => {
    const email = extractUserId(name);

    if (
      email.endsWith("@service.bytebase.com") ||
      email.includes(".service.bytebase.com")
    ) {
      const response = await serviceAccountStore.getOrFetchServiceAccount(
        `${serviceAccountNamePrefix}${email}`,
        silent
      );
      return setUser(serviceAccountToUser(response));
    }

    if (
      email.endsWith("@workload.bytebase.com") ||
      email.includes(".workload.bytebase.com")
    ) {
      const response = await workloadIdentityStore.getOrFetchWorkloadIdentity(
        `${workloadIdentityNamePrefix}${email}`,
        silent
      );
      return setUser(workloadIdentityToUser(response));
    }

    const request = create(GetUserRequestSchema, {
      name,
    });
    const response = await userServiceClientConnect.getUser(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    return setUser(response);
  };

  const createUser = async (user: User) => {
    const request = create(CreateUserRequestSchema, {
      user: user,
    });
    const response = await userServiceClientConnect.createUser(request);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.USER,
      },
    ]);
    return setUser(response);
  };

  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByIdentifier(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }

    const request = create(UpdateUserRequestSchema, {
      user: updateUserRequest.user,
      updateMask: updateUserRequest.updateMask,
      otpCode: updateUserRequest.otpCode,
      regenerateTempMfaSecret: updateUserRequest.regenerateTempMfaSecret,
      regenerateRecoveryCodes: updateUserRequest.regenerateRecoveryCodes,
    });
    const response = await userServiceClientConnect.updateUser(request);

    return setUser(response);
  };

  const updateEmail = async (oldEmail: string, newEmail: string) => {
    const originData = await getOrFetchUserByIdentifier(oldEmail);
    if (!originData) {
      throw new Error(`user with email ${oldEmail} not found`);
    }
    const response = await userServiceClientConnect.updateEmail({
      name: `users/${oldEmail}`,
      email: newEmail,
    });
    return setUser(response);
  };

  const archiveUser = async (name: string) => {
    const request = create(DeleteUserRequestSchema, {
      name,
    });
    await userServiceClientConnect.deleteUser(request);
    actuatorStore.updateUserStat([
      {
        count: -1,
        state: State.ACTIVE,
        userType: UserType.USER,
      },
      {
        count: 1,
        state: State.DELETED,
        userType: UserType.USER,
      },
    ]);

    const user = userMapByName.value.get(name);
    if (user) {
      user.state = State.DELETED;
    }
  };

  const restoreUser = async (name: string) => {
    const request = create(UndeleteUserRequestSchema, {
      name,
    });
    const response = await userServiceClientConnect.undeleteUser(request);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.USER,
      },
      {
        count: -1,
        state: State.DELETED,
        userType: UserType.USER,
      },
    ]);
    return setUser(response);
  };

  const batchGetOrFetchUsers = async (userNameList: string[]) => {
    const validList = uniq(userNameList).filter(
      (name) =>
        Boolean(name) &&
        (name.startsWith(userNamePrefix) || name.startsWith(userBindingPrefix))
    );
    const pendingFetch = validList
      .filter((name) => {
        return getUserByIdentifier(name) === undefined;
      })
      .map((name) => ensureUserFullName(name));

    if (pendingFetch.length === 0) {
      return validList.map(
        (name) => getUserByIdentifier(name) ?? unknownUser(name)
      );
    }

    try {
      if (pendingFetch.length > 0) {
        const request = create(BatchGetUsersRequestSchema, {
          names: pendingFetch,
        });
        const response = await userServiceClientConnect.batchGetUsers(request, {
          contextValues: createContextValues().set(silentContextKey, true),
        });
        for (const user of response.users) {
          setUser(user);
        }
      }
    } finally {
      return validList.map(
        (name) => getUserByIdentifier(name) ?? unknownUser(name)
      );
    }
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
    const request = fetchUser(fullname, silent)
      .then((user) => setUser(user))
      .catch(() => unknownUser(fullname));
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

  const systemBotUser = computedAsync(() => {
    return getOrFetchUserByIdentifier(SYSTEM_BOT_USER_NAME);
  });

  return {
    allUser,
    systemBotUser,
    fetchCurrentUser,
    fetchUserList,
    createUser,
    updateUser,
    updateEmail,
    batchGetOrFetchUsers,
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
