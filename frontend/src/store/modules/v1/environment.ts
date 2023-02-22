import { defineStore } from "pinia";
import { environmentServiceClient } from "@/grpcweb";
import { Environment } from "@/types/proto/v1/environment_service";
import { ResourceId } from "@/types";
import { isEqual, isUndefined } from "lodash-es";
import { State } from "@/types/proto/v1/common";

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
      const { environments } =
        await environmentServiceClient().listEnvironments({
          showDeleted,
        });
      for (const env of environments) {
        this.environmentMapByName.set(env.name, env);
      }
      return environments;
    },
    async createEnvironment(environment: Partial<Environment>) {
      const createdEnvironment =
        await environmentServiceClient().createEnvironment({
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

      const environment = await environmentServiceClient().updateEnvironment({
        environment: update,
        updateMask: getUpdateMaskFromEnvironments(originData, update),
      });
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    async deleteEnvironment(name: string) {
      await environmentServiceClient().deleteEnvironment({
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
      const environment = await environmentServiceClient().undeleteEnvironment({
        name,
      });
      this.environmentMapByName.set(environment.name, environment);
    },
    async getOrFetchEnvironmentByName(name: string) {
      const cachedData = this.environmentMapByName.get(name);
      if (cachedData) {
        return cachedData;
      }
      const environment = await environmentServiceClient().getEnvironment({
        name,
      });
      this.environmentMapByName.set(environment.name, environment);
      return environment;
    },
    getEnvironmentByName(name: string) {
      return this.environmentMapByName.get(name);
    },
  },
});

const getUpdateMaskFromEnvironments = (
  origin: Environment,
  update: Partial<Environment>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("environment.title");
  }
  if (!isUndefined(update.order) && !isEqual(origin.order, update.order)) {
    updateMask.push("environment.order");
  }
  if (!isUndefined(update.tier) && !isEqual(origin.tier, update.tier)) {
    updateMask.push("environment.tier");
  }
  return updateMask;
};
