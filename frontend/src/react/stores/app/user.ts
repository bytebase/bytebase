import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { userServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { userNamePrefix } from "@/react/lib/resourceName";
import { extractUserEmail } from "@/store/modules/v1/common";
import {
  allUsersUser,
  isValidProjectName,
  isValidUserName,
  unknownUser,
  userBindingPrefix,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchGetUsersRequestSchema,
  CreateUserRequestSchema,
  DeleteUserRequestSchema,
  GetUserRequestSchema,
  ListUsersRequestSchema,
  UndeleteUserRequestSchema,
  UpdateEmailRequestSchema,
  UpdateUserRequestSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { ensureUserFullName } from "@/utils/v1/user";
import type { AppSliceCreator, UserFilter, UserSlice } from "./types";

const UNKNOWN_PROJECT_NAME_LEGACY = "projects/-";

export const buildUserFilter = (params: UserFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.contains("${search}") || email.contains("${search}"))`);
  }
  if (
    isValidProjectName(params.project) &&
    params.project !== UNKNOWN_PROJECT_NAME_LEGACY
  ) {
    filter.push(`project == "${params.project}"`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }
  return filter.join(" && ");
};

export const createUserSlice: AppSliceCreator<UserSlice> = (set, get) => {
  const adjustActivatedUserCount = (delta: number) => {
    set((state) => {
      if (!state.serverInfo) {
        return {};
      }
      return {
        serverInfo: {
          ...state.serverInfo,
          activatedUserCount: Math.max(
            0,
            state.serverInfo.activatedUserCount + delta
          ),
        },
      };
    });
  };

  return {
    usersByName: { [allUsersUser().name]: allUsersUser() },
    userRequests: {},

    listUsers: async (params) => {
      if (!get().hasWorkspacePermission("bb.users.list")) {
        return { users: [], nextPageToken: "" };
      }
      const response = await userServiceClientConnect.listUsers(
        createProto(ListUsersRequestSchema, {
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          filter: buildUserFilter(params.filter ?? {}),
          showDeleted:
            params.showDeleted ?? params.filter?.state === State.DELETED,
        })
      );
      set((state) => ({
        usersByName: {
          ...state.usersByName,
          ...Object.fromEntries(
            response.users.map((user) => [user.name, user])
          ),
        },
      }));
      return { users: response.users, nextPageToken: response.nextPageToken };
    },

    fetchUser: async (name, silent = false) => {
      const validName = ensureUserFullName(name);
      const existing = get().usersByName[validName];
      if (existing) return existing;
      const pending = get().userRequests[validName];
      if (pending) return pending;

      const request = userServiceClientConnect
        .getUser(createProto(GetUserRequestSchema, { name: validName }), {
          contextValues: createContextValues().set(silentContextKey, silent),
        })
        .then((user) => {
          set((state) => {
            const { [validName]: _, ...userRequests } = state.userRequests;
            return {
              usersByName: { ...state.usersByName, [user.name]: user },
              userRequests,
            };
          });
          return user;
        })
        .catch(() => {
          set((state) => {
            const { [validName]: _, ...userRequests } = state.userRequests;
            return { userRequests };
          });
          return undefined;
        });
      set((state) => ({
        userRequests: { ...state.userRequests, [validName]: request },
      }));
      return request;
    },

    batchGetOrFetchUsers: async (names) => {
      const validNames = uniq(names).filter(
        (name) =>
          Boolean(name) &&
          (name.startsWith(userNamePrefix) ||
            name.startsWith(userBindingPrefix))
      );
      const missing = validNames
        .filter((name) => get().getUserByIdentifier(name) === undefined)
        .map((name) => ensureUserFullName(name));

      if (missing.length > 0) {
        try {
          const response = await userServiceClientConnect.batchGetUsers(
            createProto(BatchGetUsersRequestSchema, { names: missing }),
            { contextValues: createContextValues().set(silentContextKey, true) }
          );
          set((state) => ({
            usersByName: {
              ...state.usersByName,
              ...Object.fromEntries(
                response.users.map((user) => [user.name, user])
              ),
            },
          }));
        } catch {
          // Match the legacy store: return cached users plus unknown fallbacks.
        }
      }

      return validNames.map(
        (name) => get().getUserByIdentifier(name) ?? unknownUser(name)
      );
    },

    getOrFetchUserByIdentifier: async ({
      identifier,
      silent = true,
      fallback = true,
    }) => {
      const user = get().getUserByIdentifier(identifier);
      if (user) return user;

      const fullname = ensureUserFullName(identifier);
      if (!isValidUserName(fullname)) {
        return unknownUser();
      }
      const fetched = await get().fetchUser(fullname, silent);
      return fetched ?? unknownUser(fallback ? fullname : "");
    },

    getUserByIdentifier: (identifier) => {
      if (!identifier) return undefined;
      const id = extractUserEmail(identifier);
      if (Number.isNaN(Number(id))) {
        return Object.values(get().usersByName).find(
          (user) => user.email === id
        );
      }
      return get().usersByName[`${userNamePrefix}${id}`];
    },

    createUser: async (user) => {
      const response = await userServiceClientConnect.createUser(
        createProto(CreateUserRequestSchema, { user })
      );
      set((state) => ({
        usersByName: { ...state.usersByName, [response.name]: response },
      }));
      adjustActivatedUserCount(1);
      return response;
    },

    updateUser: async (request) => {
      const name = request.user?.name || "";
      if (!request.allowMissing) {
        const originData = await get().getOrFetchUserByIdentifier({
          identifier: name,
          fallback: false,
        });
        if (!isValidUserName(originData.name)) {
          throw new Error(`user with name ${name} not found`);
        }
      }

      const response = await userServiceClientConnect.updateUser(
        createProto(UpdateUserRequestSchema, {
          user: request.user,
          updateMask: request.updateMask,
          otpCode: request.otpCode,
          regenerateTempMfaSecret: request.regenerateTempMfaSecret,
          regenerateRecoveryCodes: request.regenerateRecoveryCodes,
          allowMissing: request.allowMissing,
        })
      );
      set((state) => ({
        usersByName: { ...state.usersByName, [response.name]: response },
        currentUser:
          state.currentUser?.name === response.name
            ? response
            : state.currentUser,
      }));
      return response;
    },

    updateEmail: async (oldEmail, newEmail) => {
      const originData = await get().getOrFetchUserByIdentifier({
        identifier: oldEmail,
        fallback: false,
      });
      if (!isValidUserName(originData.name)) {
        throw new Error(`user with email ${oldEmail} not found`);
      }

      const oldName = originData.name;
      const response = await userServiceClientConnect.updateEmail(
        createProto(UpdateEmailRequestSchema, {
          name: ensureUserFullName(oldEmail),
          email: newEmail,
        })
      );
      set((state) => {
        const usersByName = { ...state.usersByName };
        if (oldName !== response.name) {
          delete usersByName[oldName];
        }
        usersByName[response.name] = response;
        return {
          usersByName,
          currentUser:
            state.currentUser?.name === oldName ? response : state.currentUser,
        };
      });
      return response;
    },

    archiveUser: async (name) => {
      const validName = ensureUserFullName(name);
      await userServiceClientConnect.deleteUser(
        createProto(DeleteUserRequestSchema, { name: validName })
      );
      set((state) => {
        const cached = state.usersByName[validName];
        if (!cached) return {};
        return {
          usersByName: {
            ...state.usersByName,
            [validName]: { ...cached, state: State.DELETED },
          },
        };
      });
      adjustActivatedUserCount(-1);
    },

    restoreUser: async (name) => {
      const response = await userServiceClientConnect.undeleteUser(
        createProto(UndeleteUserRequestSchema, {
          name: ensureUserFullName(name),
        })
      );
      set((state) => ({
        usersByName: { ...state.usersByName, [response.name]: response },
      }));
      adjustActivatedUserCount(1);
      return response;
    },
  };
};
