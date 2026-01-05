import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { identityProviderServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  DeleteIdentityProviderRequestSchema,
  GetIdentityProviderRequestSchema,
  ListIdentityProvidersRequestSchema,
  UpdateIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";

export const useIdentityProviderStore = defineStore("idp", () => {
  const identityProviderMapByName = ref(new Map<string, IdentityProvider>());

  const identityProviderList = computed(() => {
    return Array.from(identityProviderMapByName.value.values());
  });

  const fetchIdentityProviderList = async () => {
    const request = create(ListIdentityProvidersRequestSchema, {});
    const response =
      await identityProviderServiceClientConnect.listIdentityProviders(request);
    for (const identityProvider of response.identityProviders) {
      identityProviderMapByName.value.set(
        identityProvider.name,
        identityProvider
      );
    }
    return response.identityProviders;
  };

  const getOrFetchIdentityProviderByName = async (
    name: string,
    silent = false
  ) => {
    const cachedData = identityProviderMapByName.value.get(name);
    if (cachedData) {
      return cachedData;
    }
    const request = create(GetIdentityProviderRequestSchema, { name });
    const identityProvider =
      await identityProviderServiceClientConnect.getIdentityProvider(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      });
    identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return identityProvider;
  };

  const getIdentityProviderByName = (name: string) => {
    return identityProviderMapByName.value.get(name);
  };

  const createIdentityProvider = async (createProvider: IdentityProvider) => {
    const request = create(CreateIdentityProviderRequestSchema, {
      identityProvider: createProvider,
      identityProviderId: createProvider.name,
    });
    const identityProvider =
      await identityProviderServiceClientConnect.createIdentityProvider(
        request
      );
    identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return identityProvider;
  };

  const patchIdentityProvider = async (update: Partial<IdentityProvider>) => {
    const originData = await getOrFetchIdentityProviderByName(
      update.name || ""
    );
    if (!originData) {
      throw new Error(`identity provider with name ${update.name} not found`);
    }

    const fullProvider = { ...originData, ...update } as IdentityProvider;
    const request = create(UpdateIdentityProviderRequestSchema, {
      identityProvider: fullProvider,
      updateMask: {
        paths: getUpdateMaskFromIdentityProviders(originData, update),
      },
    });
    const identityProvider =
      await identityProviderServiceClientConnect.updateIdentityProvider(
        request
      );
    identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return identityProvider;
  };

  const deleteIdentityProvider = async (name: string) => {
    const request = create(DeleteIdentityProviderRequestSchema, { name });
    await identityProviderServiceClientConnect.deleteIdentityProvider(request);
    identityProviderMapByName.value.delete(name);
  };

  return {
    identityProviderMapByName,
    identityProviderList,
    fetchIdentityProviderList,
    getOrFetchIdentityProviderByName,
    getIdentityProviderByName,
    createIdentityProvider,
    patchIdentityProvider,
    deleteIdentityProvider,
  };
});

const getUpdateMaskFromIdentityProviders = (
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
