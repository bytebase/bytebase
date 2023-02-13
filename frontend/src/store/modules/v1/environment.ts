import { defineStore } from "pinia";
import { environmentServiceClient } from "@/grpcweb";
import { Environment } from "@/types/proto/v1/environment_service";

interface EnvironmentState {
  environmentMapByName: Map<string, Environment>;
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
    getEnvironmentByName(name: string) {
      return this.environmentMapByName.get(name);
    },
  },
});
