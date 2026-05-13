import type { Factor } from "@/plugins/cel";
import type { ResourceSelectOption } from "@/types/v2-shared";
import {
  getDatabaseIdOptionConfig,
  getEnvironmentIdOptions,
  getInstanceIdOptionConfig,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
} from "@/utils/cel-attributes";
import { type OptionConfig } from "@/utils/expr";

export const FactorList: Factor[] = [
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
];

export const getDatabaseGroupOptionConfigMap = (project: string) => {
  return FactorList.reduce((map, factor) => {
    let options: ResourceSelectOption<unknown>[] = [];
    switch (factor) {
      case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
        options = getEnvironmentIdOptions();
        break;
      case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
        map.set(factor, getInstanceIdOptionConfig());
        return map;
      case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME:
        map.set(factor, getDatabaseIdOptionConfig(project));
        return map;
    }
    map.set(factor, {
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
};
