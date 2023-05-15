import { defineStore } from "pinia";
import { computed } from "vue";
import { environmentServiceClient } from "@/grpcweb";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import { ResourceId, EnvironmentId } from "@/types";
import { isEqual, isUndefined } from "lodash-es";
import { State } from "@/types/proto/v1/common";
import { environmentNamePrefix } from "@/store/modules/v1/common";

interface EnvironmentState {
  environmentMapByName: Map<ResourceId, Environment>;
}

export const useEnvironmentV1Store = defineStore("environment_v1", {
  state: (): EnvironmentState => ({
    environmentMapByName: new Map(),
  }),
  getters: {
    environmentList(state) {
      return Array.from(state.environmentMapByName.values());
    },
  },
  actions: {
    async fetchEnvironments(showDeleted = false) {
      const { environments } = await environmentServiceClient.listEnvironments({
        showDeleted,
      });
      for (const env of environments) {
        this.environmentMapByName.set(env.name, env);
      }
      return environments;
    },
    getEnvironmentList(showDeleted = false): Environment[] {
      return this.environmentList.filter((environment: Environment) => {
        if (environment.state == State.DELETED && !showDeleted) {
          return false;
        }
        return true;
      });
    },
    async createEnvironment(environment: Partial<Environment>) {
      const createdEnvironment =
        await environmentServiceClient.createEnvironment({
          environment,
          environmentId: environment.name,
        });
      this.environmentMapByName.set(
        createdEnvironment.name,
        createdEnvironment
      );
      return createdEnvironment;
    },
    async updateEnvironment(update: Partial<Environment>) {
      const originData = await this.getOrFetchEnvironmentByName(
        update.name || ""
      );
      if (!originData) {
        throw new Error(`environment with name ${update.name} not found`);
      }

      const environment = await environmentServiceClient.updateEnvironment({
        environment: update,
        updateMask: getUpdateMaskFromEnvironments(originData, update),
      });
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    async deleteEnvironment(name: string) {
      await environmentServiceClient.deleteEnvironment({
        name,
      });
      const cachedData = this.getEnvironmentByName(name);
      if (cachedData) {
        this.environmentMapByName.set(name, {
          ...cachedData,
          state: State.DELETED,
        });
      }
    },
    async undeleteEnvironment(name: string) {
      const environment = await environmentServiceClient.undeleteEnvironment({
        name,
      });
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    async getOrFetchEnvironmentByName(name: string) {
      const cachedData = this.environmentMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const environment = await environmentServiceClient.getEnvironment({
        name,
      });
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    async getOrFetchEnvironmentByUID(uid: EnvironmentId) {
      const name = `${environmentNamePrefix}${uid}`;
      return this.getOrFetchEnvironmentByName(name);
    },
    getEnvironmentByName(name: string) {
      return this.environmentMapByName.get(name);
    },
    getEnvironmentByUID(uid: EnvironmentId) {
      if (typeof uid === "string") {
        uid = parseInt(uid, 10);
      }
      return (
        this.environmentList.find((env) => parseInt(env.uid, 10) == uid) ??
        Environment.fromJSON({})
      );
    },
  },
});

const getUpdateMaskFromEnvironments = (
  origin: Environment,
  update: Partial<Environment>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (!isUndefined(update.order) && !isEqual(origin.order, update.order)) {
    updateMask.push("order");
  }
  if (!isUndefined(update.tier) && !isEqual(origin.tier, update.tier)) {
    updateMask.push("tier");
  }
  return updateMask;
};

export const useEnvironmentV1List = (showDeleted = false) => {
  const store = useEnvironmentV1Store();
  return computed(() => store.getEnvironmentList(showDeleted));
};

export const defaultEnvironmentTier = EnvironmentTier.UNPROTECTED;
