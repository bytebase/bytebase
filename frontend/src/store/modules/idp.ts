import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { identityProviderServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import type { IdentityProvider as NewIdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import {
  CreateIdentityProviderRequestSchema,
  DeleteIdentityProviderRequestSchema,
  GetIdentityProviderRequestSchema,
  ListIdentityProvidersRequestSchema,
  UpdateIdentityProviderRequestSchema,
} from "@/types/proto-es/v1/idp_service_pb";
import {
  convertOldIdentityProviderToNew,
  convertNewIdentityProviderToOld,
} from "@/utils/v1/idp-conversions";
import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { ref, computed } from "vue";

export const useIdentityProviderStore = defineStore("idp", () => {
  // Internal state uses proto-es types
  const _identityProviderMapByName = ref(new Map<string, NewIdentityProvider>());
  
  // Computed property provides old proto type for compatibility
  const identityProviderMapByName = computed(() => {
    const map = new Map<string, IdentityProvider>();
    _identityProviderMapByName.value.forEach((value, key) => {
      map.set(key, convertNewIdentityProviderToOld(value));
    });
    return map;
  });

  const identityProviderList = computed(() => {
    return Array.from(identityProviderMapByName.value.values());
  });

  const fetchIdentityProviderList = async () => {
    const request = create(ListIdentityProvidersRequestSchema, {});
    const response = await identityProviderServiceClientConnect.listIdentityProviders(request);
    for (const identityProvider of response.identityProviders) {
      _identityProviderMapByName.value.set(
        identityProvider.name,
        identityProvider
      );
    }
    return response.identityProviders.map(convertNewIdentityProviderToOld);
  };

  const getOrFetchIdentityProviderByName = async (name: string, silent = false) => {
    const cachedData = _identityProviderMapByName.value.get(name);
    if (cachedData) {
      return convertNewIdentityProviderToOld(cachedData);
    }
    const request = create(GetIdentityProviderRequestSchema, { name });
    const identityProvider = await identityProviderServiceClientConnect.getIdentityProvider(
      request,
      { contextValues: createContextValues().set(silentContextKey, silent) }
    );
    _identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return convertNewIdentityProviderToOld(identityProvider);
  };

  const getIdentityProviderByName = (name: string) => {
    const cachedData = _identityProviderMapByName.value.get(name);
    return cachedData ? convertNewIdentityProviderToOld(cachedData) : undefined;
  };

  const createIdentityProvider = async (createProvider: IdentityProvider) => {
    const newProvider = convertOldIdentityProviderToNew(createProvider);
    const request = create(CreateIdentityProviderRequestSchema, {
      identityProvider: newProvider,
      identityProviderId: createProvider.name,
    });
    const identityProvider = await identityProviderServiceClientConnect.createIdentityProvider(request);
    _identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return convertNewIdentityProviderToOld(identityProvider);
  };

  const patchIdentityProvider = async (update: Partial<IdentityProvider>) => {
    const originData = await getOrFetchIdentityProviderByName(
      update.name || ""
    );
    if (!originData) {
      throw new Error(`identity provider with name ${update.name} not found`);
    }

    const fullProvider = { ...originData, ...update } as IdentityProvider;
    const newProvider = convertOldIdentityProviderToNew(fullProvider);
    const request = create(UpdateIdentityProviderRequestSchema, {
      identityProvider: newProvider,
      updateMask: { paths: getUpdateMaskFromIdentityProviders(originData, update) },
    });
    const identityProvider = await identityProviderServiceClientConnect.updateIdentityProvider(request);
    _identityProviderMapByName.value.set(
      identityProvider.name,
      identityProvider
    );
    return convertNewIdentityProviderToOld(identityProvider);
  };

  const deleteIdentityProvider = async (name: string) => {
    const request = create(DeleteIdentityProviderRequestSchema, { name });
    await identityProviderServiceClientConnect.deleteIdentityProvider(request);
    _identityProviderMapByName.value.delete(name);
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
