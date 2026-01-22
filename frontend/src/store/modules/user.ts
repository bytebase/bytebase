import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { computedAsync } from "@vueuse/core";
import { isEqual, isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import {
  serviceAccountServiceClientConnect,
  userServiceClientConnect,
  workloadIdentityServiceClientConnect,
} from "@/connect";
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
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import {
  CreateServiceAccountRequestSchema,
  DeleteServiceAccountRequestSchema,
  GetServiceAccountRequestSchema,
  ListServiceAccountsRequestSchema,
  ServiceAccountSchema,
  UndeleteServiceAccountRequestSchema,
  UpdateServiceAccountRequestSchema,
} from "@/types/proto-es/v1/service_account_service_pb";
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
  UserSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  CreateWorkloadIdentityRequestSchema,
  DeleteWorkloadIdentityRequestSchema,
  GetWorkloadIdentityRequestSchema,
  ListWorkloadIdentitiesRequestSchema,
  UndeleteWorkloadIdentityRequestSchema,
  UpdateWorkloadIdentityRequestSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { ensureUserFullName, hasWorkspacePermissionV2 } from "@/utils";
import { useActuatorV1Store } from "./v1/actuator";
import { extractUserId, userNamePrefix } from "./v1/common";
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

const serviceAccountToUser = (sa: ServiceAccount): User => {
  return create(UserSchema, {
    name: `users/${sa.email}`,
    email: sa.email,
    title: sa.title,
    state: sa.state,
    userType: UserType.SERVICE_ACCOUNT,
    serviceKey: sa.serviceKey,
  });
};

const workloadIdentityToUser = (wi: WorkloadIdentity): User => {
  return create(UserSchema, {
    name: `users/${wi.email}`,
    email: wi.email,
    title: wi.title,
    state: wi.state,
    userType: UserType.WORKLOAD_IDENTITY,
    workloadIdentityConfig: wi.workloadIdentityConfig,
  });
};

const extractEmailPrefix = (email: string, suffix: string): string => {
  if (email.endsWith(suffix)) {
    return email.slice(0, -suffix.length);
  }
  return email.split("@")[0];
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
        const request = create(ListServiceAccountsRequestSchema, {
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          showDeleted,
        });
        const response =
          await serviceAccountServiceClientConnect.listServiceAccounts(request);
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
        const request = create(ListWorkloadIdentitiesRequestSchema, {
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          showDeleted,
        });
        const response =
          await workloadIdentityServiceClientConnect.listWorkloadIdentities(
            request
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
      const request = create(GetServiceAccountRequestSchema, {
        name: `serviceAccounts/${email}`,
      });
      const response =
        await serviceAccountServiceClientConnect.getServiceAccount(request, {
          contextValues: createContextValues().set(silentContextKey, silent),
        });
      return setUser(serviceAccountToUser(response));
    }

    if (
      email.endsWith("@workload.bytebase.com") ||
      email.includes(".workload.bytebase.com")
    ) {
      const request = create(GetWorkloadIdentityRequestSchema, {
        name: `workloadIdentities/${email}`,
      });
      const response =
        await workloadIdentityServiceClientConnect.getWorkloadIdentity(
          request,
          {
            contextValues: createContextValues().set(silentContextKey, silent),
          }
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
    let response: User;

    if (user.userType === UserType.SERVICE_ACCOUNT) {
      const serviceAccountId = extractEmailPrefix(
        user.email,
        "@service.bytebase.com"
      );
      const request = create(CreateServiceAccountRequestSchema, {
        serviceAccountId,
        serviceAccount: create(ServiceAccountSchema, {
          title: user.title,
        }),
      });
      const sa =
        await serviceAccountServiceClientConnect.createServiceAccount(request);
      response = serviceAccountToUser(sa);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const workloadIdentityId = extractEmailPrefix(
        user.email,
        "@workload.bytebase.com"
      );
      const request = create(CreateWorkloadIdentityRequestSchema, {
        workloadIdentityId,
        workloadIdentity: create(WorkloadIdentitySchema, {
          title: user.title,
          workloadIdentityConfig: user.workloadIdentityConfig,
        }),
      });
      const wi =
        await workloadIdentityServiceClientConnect.createWorkloadIdentity(
          request
        );
      response = workloadIdentityToUser(wi);
    } else {
      const request = create(CreateUserRequestSchema, {
        user: user,
      });
      response = await userServiceClientConnect.createUser(request);
    }

    await actuatorStore.fetchServerInfo();
    return setUser(response);
  };

  const updateUser = async (updateUserRequest: UpdateUserRequest) => {
    const name = updateUserRequest.user?.name || "";
    const originData = await getOrFetchUserByIdentifier(name);
    if (!originData) {
      throw new Error(`user with name ${name} not found`);
    }

    let response: User;
    const updateMaskPaths = updateUserRequest.updateMask?.paths ?? [];

    if (
      originData.userType === UserType.SERVICE_ACCOUNT &&
      updateMaskPaths.includes("service_key")
    ) {
      const request = create(UpdateServiceAccountRequestSchema, {
        serviceAccount: create(ServiceAccountSchema, {
          name: `serviceAccounts/${originData.email}`,
          title: updateUserRequest.user?.title ?? originData.title,
        }),
        updateMask: updateUserRequest.updateMask,
      });
      const sa =
        await serviceAccountServiceClientConnect.updateServiceAccount(request);
      response = serviceAccountToUser(sa);
    } else if (originData.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(UpdateWorkloadIdentityRequestSchema, {
        workloadIdentity: create(WorkloadIdentitySchema, {
          name: `workloadIdentities/${originData.email}`,
          title: updateUserRequest.user?.title ?? originData.title,
          workloadIdentityConfig:
            updateUserRequest.user?.workloadIdentityConfig ??
            originData.workloadIdentityConfig,
        }),
        updateMask: updateUserRequest.updateMask,
      });
      const wi =
        await workloadIdentityServiceClientConnect.updateWorkloadIdentity(
          request
        );
      response = workloadIdentityToUser(wi);
    } else {
      const request = create(UpdateUserRequestSchema, {
        user: updateUserRequest.user,
        updateMask: updateUserRequest.updateMask,
        otpCode: updateUserRequest.otpCode,
        regenerateTempMfaSecret: updateUserRequest.regenerateTempMfaSecret,
        regenerateRecoveryCodes: updateUserRequest.regenerateRecoveryCodes,
      });
      response = await userServiceClientConnect.updateUser(request);
    }

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

  const archiveUser = async (user: User) => {
    if (user.userType === UserType.SERVICE_ACCOUNT) {
      const request = create(DeleteServiceAccountRequestSchema, {
        name: `serviceAccounts/${user.email}`,
      });
      await serviceAccountServiceClientConnect.deleteServiceAccount(request);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(DeleteWorkloadIdentityRequestSchema, {
        name: `workloadIdentities/${user.email}`,
      });
      await workloadIdentityServiceClientConnect.deleteWorkloadIdentity(
        request
      );
    } else {
      const request = create(DeleteUserRequestSchema, {
        name: user.name,
      });
      await userServiceClientConnect.deleteUser(request);
    }

    user.state = State.DELETED;
    await actuatorStore.fetchServerInfo();
    return user;
  };

  const restoreUser = async (user: User) => {
    let response: User;

    if (user.userType === UserType.SERVICE_ACCOUNT) {
      const request = create(UndeleteServiceAccountRequestSchema, {
        name: `serviceAccounts/${user.email}`,
      });
      const sa =
        await serviceAccountServiceClientConnect.undeleteServiceAccount(
          request
        );
      response = serviceAccountToUser(sa);
    } else if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const request = create(UndeleteWorkloadIdentityRequestSchema, {
        name: `workloadIdentities/${user.email}`,
      });
      const wi =
        await workloadIdentityServiceClientConnect.undeleteWorkloadIdentity(
          request
        );
      response = workloadIdentityToUser(wi);
    } else {
      const request = create(UndeleteUserRequestSchema, {
        name: user.name,
      });
      response = await userServiceClientConnect.undeleteUser(request);
    }

    await actuatorStore.fetchServerInfo();
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

    const regularUsers: string[] = [];
    const serviceAccountEmails: string[] = [];
    const workloadIdentityEmails: string[] = [];

    for (const name of pendingFetch) {
      const email = extractUserId(name);
      if (
        email.endsWith("@service.bytebase.com") ||
        email.includes(".service.bytebase.com")
      ) {
        serviceAccountEmails.push(email);
      } else if (
        email.endsWith("@workload.bytebase.com") ||
        email.includes(".workload.bytebase.com")
      ) {
        workloadIdentityEmails.push(email);
      } else {
        regularUsers.push(name);
      }
    }

    try {
      const promises: Promise<void>[] = [];

      if (regularUsers.length > 0) {
        const userPromise = (async () => {
          const request = create(BatchGetUsersRequestSchema, {
            names: regularUsers,
          });
          const response = await userServiceClientConnect.batchGetUsers(
            request,
            {
              contextValues: createContextValues().set(silentContextKey, true),
            }
          );
          for (const user of response.users) {
            setUser(user);
          }
        })();
        promises.push(userPromise);
      }

      for (const email of serviceAccountEmails) {
        const saPromise = (async () => {
          try {
            const request = create(GetServiceAccountRequestSchema, {
              name: `serviceAccounts/${email}`,
            });
            const response =
              await serviceAccountServiceClientConnect.getServiceAccount(
                request,
                {
                  contextValues: createContextValues().set(
                    silentContextKey,
                    true
                  ),
                }
              );
            setUser(serviceAccountToUser(response));
          } catch {
            // Ignore errors for individual fetches
          }
        })();
        promises.push(saPromise);
      }

      for (const email of workloadIdentityEmails) {
        const wiPromise = (async () => {
          try {
            const request = create(GetWorkloadIdentityRequestSchema, {
              name: `workloadIdentities/${email}`,
            });
            const response =
              await workloadIdentityServiceClientConnect.getWorkloadIdentity(
                request,
                {
                  contextValues: createContextValues().set(
                    silentContextKey,
                    true
                  ),
                }
              );
            setUser(workloadIdentityToUser(response));
          } catch {
            // Ignore errors for individual fetches
          }
        })();
        promises.push(wiPromise);
      }

      await Promise.all(promises);
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
