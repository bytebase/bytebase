import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import type { ResourceId } from "@/types";
import {
  isValidEnvironmentName,
  NULL_ENVIRONMENT_NAME,
  nullEnvironment,
  unknownEnvironment,
} from "@/types";
import type {
  EnvironmentSetting,
  EnvironmentSetting_Environment,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  EnvironmentSetting_EnvironmentSchema,
  EnvironmentSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
import { hasWorkspacePermissionV2 } from "@/utils";
import { environmentNamePrefix } from "./common";
import { useSettingV1Store } from "./setting";

interface EnvironmentState {
  environmentMapById: Map<ResourceId, Environment>;
}

const getEnvironmentByIdMap = (
  environments: Environment[]
): Map<ResourceId, Environment> => {
  return new Map(
    environments.map((environment) => [environment.id, environment])
  );
};

const convertToEnvironments = (
  environments: EnvironmentSetting_Environment[]
): Environment[] => {
  return environments.map<Environment>((env, i) => {
    return {
      ...create(EnvironmentSetting_EnvironmentSchema, {
        name: `${environmentNamePrefix}${env.id}`,
        id: env.id,
        title: env.title,
        color: env.color,
        tags: env.tags,
      }),
      order: i,
    };
  });
};

const convertEnvironment = (
  env: Environment
): EnvironmentSetting_Environment => {
  return create(EnvironmentSetting_EnvironmentSchema, {
    name: env.name,
    id: env.id,
    title: env.title,
    color: env.color,
    tags: env.tags,
  });
};

const convertEnvironments = (
  environments: Environment[]
): EnvironmentSetting_Environment[] => {
  return environments.map(convertEnvironment);
};

const updateEnvironmentSetting = async (
  environment: EnvironmentSetting
): Promise<Environment[]> => {
  const response = await useSettingV1Store().upsertSetting({
    name: Setting_SettingName.ENVIRONMENT,
    value: create(SettingValueSchema, {
      value: {
        case: "environment",
        value: environment,
      },
    }),
  });

  // Extract environments from proto-es response
  if (response.value?.value?.case === "environment") {
    const settingEnvironments = response.value.value.value.environments ?? [];
    return convertToEnvironments(settingEnvironments);
  }
  return [];
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
    async fetchEnvironments(silent = false) {
      if (!hasWorkspacePermissionV2("bb.settings.getEnvironment")) {
        return;
      }
      const setting = await useSettingV1Store().fetchSettingByName(
        Setting_SettingName.ENVIRONMENT,
        silent
      );
      if (setting?.value?.value?.case === "environment") {
        const settingEnvironments =
          setting.value.value.value.environments ?? [];
        const environments = convertToEnvironments(settingEnvironments);
        this.environmentMapById = getEnvironmentByIdMap(environments);
      }
    },
    async createEnvironment(
      environment: Partial<Environment>
    ): Promise<Environment> {
      const e = create(EnvironmentSetting_EnvironmentSchema, {
        name: "",
        id: environment.id ?? "",
        title: environment.title ?? "",
        color: environment.color ?? "",
        tags: environment.tags ?? {},
      });
      const newEnvironmentSettingValue = [
        ...convertEnvironments(this.environmentList),
        e,
      ];

      const newEnvironments = await updateEnvironmentSetting(
        create(EnvironmentSettingSchema, {
          environments: newEnvironmentSettingValue,
        })
      );

      const newEnvironmentMapById = getEnvironmentByIdMap(newEnvironments);
      this.environmentMapById = newEnvironmentMapById;
      const newEnvironment = newEnvironmentMapById.get(environment.id ?? "");
      if (!newEnvironment) {
        throw new Error(`environment with id ${environment.id} not found`);
      }
      return newEnvironment;
    },
    async updateEnvironment(
      update: Partial<Environment>
    ): Promise<Environment> {
      const originData = this.getEnvironmentByName(update.id || "");
      if (!originData) {
        throw new Error(`environment with id ${update.id} not found`);
      }
      const newEnvironments = await updateEnvironmentSetting(
        create(EnvironmentSettingSchema, {
          environments: convertEnvironments(
            this.environmentList.map((environment) => {
              if (environment.id === update.id) {
                environment.title = update.title ?? environment.title;
                environment.color = update.color ?? environment.color;
                environment.tags = update.tags ?? environment.tags;
                environment.order = update.order ?? environment.order;
              }
              return environment;
            })
          ),
        })
      );

      const newEnvironmentMapById = getEnvironmentByIdMap(newEnvironments);
      this.environmentMapById = newEnvironmentMapById;
      const newEnvironment = newEnvironmentMapById.get(update.id ?? "");
      if (!newEnvironment) {
        throw new Error(`environment with id ${update.id} not found`);
      }
      return newEnvironment;
    },
    async deleteEnvironment(name: string): Promise<void> {
      const id = name.replace(environmentNamePrefix, "");
      const newEnvironments = await updateEnvironmentSetting(
        create(EnvironmentSettingSchema, {
          environments: convertEnvironments(
            this.environmentList.filter((environment) => environment.id !== id)
          ),
        })
      );
      this.environmentMapById = getEnvironmentByIdMap(newEnvironments);
    },
    async reorderEnvironmentList(
      orderedEnvironmentList: Environment[]
    ): Promise<Environment[]> {
      const newEnvironments = await updateEnvironmentSetting(
        create(EnvironmentSettingSchema, {
          environments: convertEnvironments(orderedEnvironmentList),
        })
      );
      this.environmentMapById = getEnvironmentByIdMap(newEnvironments);
      return newEnvironments;
    },
    getEnvironmentByName(name: string, fallback: boolean = true) {
      if (name === NULL_ENVIRONMENT_NAME) {
        return nullEnvironment();
      }
      const id = name.replace(environmentNamePrefix, "");
      if (!id) {
        return unknownEnvironment();
      }
      const environment =
        this.environmentMapById.get(id) ?? unknownEnvironment();
      if (!isValidEnvironmentName(environment.name) && fallback) {
        return {
          ...environment,
          id,
          name,
          title: name,
        };
      }
      return environment;
    },
  },
});
