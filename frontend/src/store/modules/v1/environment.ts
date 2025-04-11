import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { environmentServiceClient, settingServiceClient } from "@/grpcweb";
import type { ResourceId } from "@/types";
import { unknownEnvironment } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import {
  EnvironmentSetting,
  EnvironmentSetting_Environment,
} from "@/types/proto/v1/setting_service";
import { environmentNamePrefix } from "./common";

interface EnvironmentState {
  environmentMapById: Map<ResourceId, Environment>;
}

const getEnvironmentByIdMap = (
  environments: Environment[]
): Map<ResourceId, Environment> => {
  return new Map(
    environments.map((environment) => [environment.name, environment])
  );
};

const convertToEnvironments = (
  environments: EnvironmentSetting_Environment[]
): Environment[] => {
  return environments.map<Environment>((env, i) => {
    return {
      name: `${environmentNamePrefix}${env.id}`,
      title: env.title,
      order: i,
      color: env.color,
      tier: env.tags["protected"] === "protected"
        ? EnvironmentTier.PROTECTED
        : EnvironmentTier.UNPROTECTED,
      state: State.ACTIVE,
    };
  });
};

const convertEnvironments = (
  environments: Environment[]
): EnvironmentSetting_Environment[] => {
  return environments.map((env) => {
    const res: EnvironmentSetting_Environment = {
      id: env.name.replace(environmentNamePrefix, ""),
      title: env.title,
      color: env.color,
      tags: {},
    };
    if (env.tier === EnvironmentTier.PROTECTED) {
      res.tags.protected = "protected";
    }
    return res;
  });
};

const getEnvironmentSetting = async (
  silent = false
): Promise<Environment[]> => {
  const setting = await settingServiceClient.getSetting(
    {
      name: "settings/bb.workspace.environment",
    },
    { silent }
  );
  const settingEnvironments =
    setting.value?.environmentSetting?.environments ?? [];
  return convertToEnvironments(settingEnvironments);
};

const updateEnvironmentSetting = async (
  environment: EnvironmentSetting
): Promise<Environment[]> => {
  const setting = await settingServiceClient.updateSetting({
    setting: {
      name: "settings/bb.workspace.environment",
      value: {
        environmentSetting: environment,
      },
    },
    updateMask: ["environment_setting"],
  });
  const settingEnvironments =
    setting.value?.environmentSetting?.environments ?? [];
  return convertToEnvironments(settingEnvironments);
};

export const useEnvironmentV1Store = defineStore("environment_v1", {
  state: (): EnvironmentState => ({
    environmentMapById: new Map(),
  }),
  getters: {
    environmentList(state) {
      return orderBy(
        Array.from(state.environmentMapById.values()),
        (env) => env.order,
        "asc"
      );
    },
  },
  actions: {
    async fetchEnvironments(_showDeleted = false, silent = false) {
      const environments = await getEnvironmentSetting(silent);
      this.environmentMapById = getEnvironmentByIdMap(environments);
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
    async createEnvironment(
      environment: Partial<Environment>
    ): Promise<Environment> {
      const e: EnvironmentSetting_Environment = {
        id: environment.name?.replace(environmentNamePrefix, "") ?? "",
        title: environment.title ?? "",
        color: environment.color ?? "",
        tags: {},
      };
      if (environment.tier === EnvironmentTier.PROTECTED) {
        e.tags.protected = "protected";
      }
      const newEnvironmentSettingValue = [
        ...convertEnvironments(this.environmentList),
        e,
      ];

      const newEnvironments = await updateEnvironmentSetting({
        environments: newEnvironmentSettingValue,
      });

      const newEnvironmentMapById = getEnvironmentByIdMap(newEnvironments);
      this.environmentMapById = newEnvironmentMapById;
      const newEnvironment = newEnvironmentMapById.get(environment.name ?? "");
      if (!newEnvironment) {
        throw new Error(`environment with name ${environment.name} not found`);
      }
      return newEnvironment;
    },
    async updateEnvironment(
      update: Partial<Environment>
    ): Promise<Environment> {
      const originData = await this.getOrFetchEnvironmentByName(
        update.name || ""
      );
      if (!originData) {
        throw new Error(`environment with name ${update.name} not found`);
      }
      const newEnvironments = await updateEnvironmentSetting({
        environments: convertEnvironments(
          this.environmentList.map((environment) => {
            if (environment.name === update.name) {
              environment.title = update.title ?? environment.title;
              environment.color = update.color ?? environment.color;
              environment.tier = update.tier ?? environment.tier;
              environment.order = update.order ?? environment.order;
            }
            return environment;
          })
        ),
      });

      const newEnvironmentMapById = getEnvironmentByIdMap(newEnvironments);
      this.environmentMapById = newEnvironmentMapById;
      const newEnvironment = newEnvironmentMapById.get(update.name || "");
      if (!newEnvironment) {
        throw new Error(`environment with name ${update.name} not found`);
      }
      return newEnvironment;
    },
    async deleteEnvironment(name: string): Promise<void> {
      const newEnvironments = await updateEnvironmentSetting({
        environments: convertEnvironments(
          this.environmentList.filter(
            (environment) => environment.name !== name
          )
        ),
      });
      this.environmentMapById = getEnvironmentByIdMap(newEnvironments);
    },
    async undeleteEnvironment(name: string) {
      const environment = await environmentServiceClient.undeleteEnvironment({
        name,
      });
      this.environmentMapById.set(environment.name, environment);
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
        this.environmentMapById.set(environment.name, environment);
      });
      return updatedEnvironmentList;
    },
    async getOrFetchEnvironmentByName(
      name: string,
      silent = false
    ): Promise<Environment | undefined> {
      const cachedData = this.environmentMapById.get(name);
      if (cachedData) {
        return cachedData;
      }
      await this.fetchEnvironments(false, silent);
      const environment = this.environmentMapById.get(name);
      return environment;
    },
    getEnvironmentByName(name: string) {
      return this.environmentMapById.get(name) ?? unknownEnvironment();
    },
  },
});

export const useEnvironmentV1List = (showDeleted = false) => {
  const store = useEnvironmentV1Store();
  return computed(() => store.getEnvironmentList(showDeleted));
};

export const defaultEnvironmentTier = EnvironmentTier.UNPROTECTED;
