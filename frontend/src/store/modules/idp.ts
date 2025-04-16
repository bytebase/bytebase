import { identityProviderClient } from "@/grpcweb";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";

interface IdentityProviderState {
  identityProviderMapByName: Map<string, IdentityProvider>;
}

export const useIdentityProviderStore = defineStore("idp", {
  state: (): IdentityProviderState => ({
    identityProviderMapByName: new Map(),
  }),
  getters: {
    identityProviderList(state) {
      return Array.from(state.identityProviderMapByName.values());
    },
  },
  actions: {
    async fetchIdentityProviderList() {
      const { identityProviders } =
        await identityProviderClient.listIdentityProviders({
        });
      for (const identityProvider of identityProviders) {
        this.identityProviderMapByName.set(
          identityProvider.name,
          identityProvider
        );
      }
      return identityProviders;
    },
    async getOrFetchIdentityProviderByName(name: string, silent = false) {
      const cachedData = this.identityProviderMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const identityProvider = await identityProviderClient.getIdentityProvider(
        { name },
        { silent }
      );
      this.identityProviderMapByName.set(
        identityProvider.name,
        identityProvider
      );
      return identityProvider;
    },
    getIdentityProviderByName(name: string) {
      const cachedData = this.identityProviderMapByName.get(name);
      return cachedData;
    },
    async createIdentityProvider(create: IdentityProvider) {
      const identityProvider =
        await identityProviderClient.createIdentityProvider({
          identityProvider: create,
          identityProviderId: create.name,
        });
      this.identityProviderMapByName.set(
        identityProvider.name,
        identityProvider
      );
      return identityProvider;
    },
    async patchIdentityProvider(update: Partial<IdentityProvider>) {
      const originData = await this.getOrFetchIdentityProviderByName(
        update.name || ""
      );
      if (!originData) {
        throw new Error(`identity provider with name ${update.name} not found`);
      }

      const identityProvider =
        await identityProviderClient.updateIdentityProvider({
          identityProvider: update,
          updateMask: getUpdateMaskFromIdentityProviders(originData, update),
        });
      this.identityProviderMapByName.set(
        identityProvider.name,
        identityProvider
      );
      return identityProvider;
    },
    async deleteIdentityProvider(name: string) {
      await identityProviderClient.deleteIdentityProvider({
        name,
      });
      const cachedData = this.getIdentityProviderByName(name);
      if (cachedData) {
        this.identityProviderMapByName.delete(name);
      }
    },
  },
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
