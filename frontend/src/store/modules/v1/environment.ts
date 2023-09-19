import { isEqual, isUndefined, orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { environmentServiceClient } from "@/grpcweb";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { ResourceId, unknownEnvironment } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";

interface EnvironmentState {
  environmentMapByName: Map<ResourceId, Environment>;
}

export const useEnvironmentV1Store = defineStore("environment_v1", {
  state: (): EnvironmentState => ({
    environmentMapByName: new Map(),
  }),
  getters: {
    environmentList(state) {
      return orderBy(
        Array.from(state.environmentMapByName.values()),
        (env) => env.order,
        "asc"
      );
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
    async reorderEnvironmentList(orderedEnvironmentList: Environment[]) {
      const updatedEnvironmentList = await Promise.all(
        orderedEnvironmentList.map((environment, order) => {
          return environmentServiceClient.updateEnvironment({
            environment: {
              ...environment,
              order,
            },
            updateMask: ["order"],
          });
        })
      );
      updatedEnvironmentList.forEach((environment) => {
        this.environmentMapByName.set(environment.name, environment);
      });
      return updatedEnvironmentList;
    },
    async getOrFetchEnvironmentByName(name: string, silent = false) {
      const cachedData = this.environmentMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const environment = await environmentServiceClient.getEnvironment(
        {
          name,
        },
        { silent }
      );
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    async getOrFetchEnvironmentByUID(uid: string) {
      const name = `${environmentNamePrefix}${uid}`;
      return this.getOrFetchEnvironmentByName(name);
    },
    getEnvironmentByName(name: string) {
      return this.environmentMapByName.get(name);
    },
    getEnvironmentByUID(uid: string) {
      return (
        this.environmentList.find((env) => env.uid == uid) ??
        unknownEnvironment()
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
