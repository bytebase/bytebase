import type { SelectOption } from "naive-ui";
import type { Factor } from "@/plugins/cel";
import { useEnvironmentV1Store, useInstanceV1List } from "@/store";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
} from "@/utils";

export type ResourceType = "DATABASE_GROUP" | "SCHEMA_GROUP";

export const FactorList: Map<ResourceType, Factor[]> = new Map([
  [
    "DATABASE_GROUP",
    [
      "resource.environment_name",
      "resource.database_name",
      "resource.instance_id",
    ],
  ],
  ["SCHEMA_GROUP", ["resource.table_name"]],
]);

export const factorSupportDropdown: Factor[] = [
  "resource.environment_name",
  "resource.instance_id",
];

export const getFactorOptionsMap = (resourceType: ResourceType) => {
  const factorList = FactorList.get(resourceType) || [];
  return factorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "resource.environment_name":
        options = getEnvironmentOptions();
        break;
      case "resource.instance_id":
        options = getInstanceIdOptions();
        break;
    }
    map.set(factor, options);
    return map;
  }, new Map<Factor, SelectOption[]>());
};

const getEnvironmentOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    const environmentId = extractEnvironmentResourceName(env.name);
    return {
      label: environmentId,
      value: env.name,
    };
  });
};

const getInstanceIdOptions = () => {
  const { instanceList } = useInstanceV1List();
  return instanceList.value.map<SelectOption>((instance) => {
    const instanceId = extractInstanceResourceName(instance.name);
    return {
      label: instanceId,
      value: instance.name,
    };
  });
};
