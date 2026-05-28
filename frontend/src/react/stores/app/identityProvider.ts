import { clone, create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isEqual, isUndefined } from "lodash-es";
import { identityProviderServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  DeleteIdentityProviderRequestSchema,
  GetIdentityProviderRequestSchema,
  IdentityProviderSchema,
  ListIdentityProvidersRequestSchema,
  UpdateIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import type { AppSliceCreator, IdentityProviderSlice } from "./types";

const getUpdateMaskFromIdentityProvider = (
  origin: IdentityProvider,
  update: Partial<IdentityProvider>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (!isUndefined(update.domain) && !isEqual(origin.domain, update.domain)) {
    updateMask.push("domain");
  }
  if (!isUndefined(update.config) && !isEqual(origin.config, update.config)) {
    updateMask.push("config");
  }
  return updateMask;
};

export const createIdentityProviderSlice: AppSliceCreator<
  IdentityProviderSlice
> = (set, get) => ({
  identityProvidersByName: {},
  identityProviderRequests: {},

  identityProviderList: () => Object.values(get().identityProvidersByName),

  listIdentityProviders: async (parent) => {
    const response =
      await identityProviderServiceClientConnect.listIdentityProviders(
        createProto(ListIdentityProvidersRequestSchema, {
          parent: parent ?? "",
        })
      );
    set({
      identityProvidersByName: Object.fromEntries(
        response.identityProviders.map((idp) => [idp.name, idp])
      ),
    });
    return response.identityProviders;
  },

  fetchIdentityProvider: async (name, silent = false) => {
    const existing = get().identityProvidersByName[name];
    if (existing) return existing;
    const pending = get().identityProviderRequests[name];
    if (pending) return pending;

    const request = identityProviderServiceClientConnect
      .getIdentityProvider(
        createProto(GetIdentityProviderRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((identityProvider) => {
        set((state) => {
          const { [name]: _, ...identityProviderRequests } =
            state.identityProviderRequests;
          return {
            identityProvidersByName: {
              ...state.identityProvidersByName,
              [identityProvider.name]: identityProvider,
            },
            identityProviderRequests,
          };
        });
        return identityProvider;
      })
      .catch(() => {
        set((state) => {
          const { [name]: _, ...identityProviderRequests } =
            state.identityProviderRequests;
          return { identityProviderRequests };
        });
        return undefined;
      });
    set((state) => ({
      identityProviderRequests: {
        ...state.identityProviderRequests,
        [name]: request,
      },
    }));
    return request;
  },

  getIdentityProvider: (name) => get().identityProvidersByName[name],

  createIdentityProvider: async (identityProvider) => {
    const idp =
      await identityProviderServiceClientConnect.createIdentityProvider(
        createProto(CreateIdentityProviderRequestSchema, {
          identityProvider,
          identityProviderId: identityProvider.name,
        })
      );
    set((state) => ({
      identityProvidersByName: {
        ...state.identityProvidersByName,
        [idp.name]: idp,
      },
    }));
    return idp;
  },

  updateIdentityProvider: async (update) => {
    const origin = await get().fetchIdentityProvider(update.name || "");
    if (!origin) {
      throw new Error(`identity provider with name ${update.name} not found`);
    }
    const fullProvider = clone(IdentityProviderSchema, origin);
    if (update.title !== undefined) fullProvider.title = update.title;
    if (update.domain !== undefined) fullProvider.domain = update.domain;
    if (update.type !== undefined) fullProvider.type = update.type;
    if (update.config !== undefined) fullProvider.config = update.config;

    const idp =
      await identityProviderServiceClientConnect.updateIdentityProvider(
        createProto(UpdateIdentityProviderRequestSchema, {
          identityProvider: fullProvider,
          updateMask: {
            paths: getUpdateMaskFromIdentityProvider(origin, update),
          },
        })
      );
    set((state) => ({
      identityProvidersByName: {
        ...state.identityProvidersByName,
        [idp.name]: idp,
      },
    }));
    return idp;
  },

  deleteIdentityProvider: async (name) => {
    await identityProviderServiceClientConnect.deleteIdentityProvider(
      createProto(DeleteIdentityProviderRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...identityProvidersByName } =
        state.identityProvidersByName;
      return { identityProvidersByName };
    });
  },
});
