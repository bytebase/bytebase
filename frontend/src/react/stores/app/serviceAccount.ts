import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { serviceAccountServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
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
import { type User, UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type {
  AccountFilter,
  AppSliceCreator,
  ServiceAccountSlice,
} from "./types";

const serviceAccountNamePrefix = "serviceAccounts/";

export const extractServiceAccountId = (identifier: string) => {
  const matches = identifier.match(
    /^(?:serviceAccount:|serviceAccounts\/)(.+)$/
  );
  return matches?.[1] ?? identifier;
};

export const ensureServiceAccountFullName = (identifier: string) => {
  const id = extractServiceAccountId(identifier);
  return `${serviceAccountNamePrefix}${id}`;
};

export const buildAccountListFilter = (params: AccountFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.contains("${search}") || email.contains("${search}"))`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }
  return filter.join(" && ");
};

export const serviceAccountToUser = (sa: ServiceAccount): User => {
  return createProto(UserSchema, {
    name: `users/${sa.email}`,
    email: sa.email,
    title: sa.title,
    state: sa.state,
    serviceKey: sa.serviceKey,
  });
};

export const createServiceAccountSlice: AppSliceCreator<ServiceAccountSlice> = (
  set,
  get
) => ({
  serviceAccountsByName: {},
  serviceAccountRequests: {},

  listServiceAccounts: async (params) => {
    const response =
      await serviceAccountServiceClientConnect.listServiceAccounts(
        createProto(ListServiceAccountsRequestSchema, {
          parent: params.parent,
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          showDeleted: params.showDeleted,
          filter: buildAccountListFilter(params.filter ?? {}),
        })
      );
    set((state) => ({
      serviceAccountsByName: {
        ...state.serviceAccountsByName,
        ...Object.fromEntries(
          response.serviceAccounts.map((sa) => [sa.name, sa])
        ),
      },
    }));
    return {
      serviceAccounts: response.serviceAccounts,
      nextPageToken: response.nextPageToken,
    };
  },

  fetchServiceAccount: async (name, silent = false) => {
    const validName = ensureServiceAccountFullName(name);
    const existing = get().serviceAccountsByName[validName];
    if (existing) return existing;
    const pending = get().serviceAccountRequests[validName];
    if (pending) return pending;

    const request = serviceAccountServiceClientConnect
      .getServiceAccount(
        createProto(GetServiceAccountRequestSchema, { name: validName }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((serviceAccount) => {
        set((state) => {
          const { [validName]: _, ...serviceAccountRequests } =
            state.serviceAccountRequests;
          return {
            serviceAccountsByName: {
              ...state.serviceAccountsByName,
              [serviceAccount.name]: serviceAccount,
            },
            serviceAccountRequests,
          };
        });
        return serviceAccount;
      })
      .catch(() => {
        set((state) => {
          const { [validName]: _, ...serviceAccountRequests } =
            state.serviceAccountRequests;
          return { serviceAccountRequests };
        });
        return undefined;
      });
    set((state) => ({
      serviceAccountRequests: {
        ...state.serviceAccountRequests,
        [validName]: request,
      },
    }));
    return request;
  },

  getServiceAccount: (name) => {
    const validName = ensureServiceAccountFullName(name);
    const email = extractServiceAccountId(validName);
    return (
      get().serviceAccountsByName[validName] ??
      createProto(ServiceAccountSchema, {
        name: validName,
        email,
        state: State.ACTIVE,
        title: email.split("@")[0],
      })
    );
  },

  createServiceAccount: async (serviceAccountId, serviceAccount, parent) => {
    const sa = await serviceAccountServiceClientConnect.createServiceAccount(
      createProto(CreateServiceAccountRequestSchema, {
        parent,
        serviceAccountId,
        serviceAccount: createProto(
          ServiceAccountSchema,
          serviceAccount as ServiceAccount
        ),
      })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },

  updateServiceAccount: async (serviceAccount, updateMask) => {
    const sa = await serviceAccountServiceClientConnect.updateServiceAccount(
      createProto(UpdateServiceAccountRequestSchema, {
        serviceAccount: createProto(
          ServiceAccountSchema,
          serviceAccount as ServiceAccount
        ),
        updateMask,
      })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },

  deleteServiceAccount: async (name) => {
    const validName = ensureServiceAccountFullName(name);
    await serviceAccountServiceClientConnect.deleteServiceAccount(
      createProto(DeleteServiceAccountRequestSchema, { name: validName })
    );
    set((state) => {
      const cached = state.serviceAccountsByName[validName];
      if (!cached) return {};
      return {
        serviceAccountsByName: {
          ...state.serviceAccountsByName,
          [validName]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  undeleteServiceAccount: async (name) => {
    const sa = await serviceAccountServiceClientConnect.undeleteServiceAccount(
      createProto(UndeleteServiceAccountRequestSchema, { name })
    );
    set((state) => ({
      serviceAccountsByName: { ...state.serviceAccountsByName, [sa.name]: sa },
    }));
    return sa;
  },
});
