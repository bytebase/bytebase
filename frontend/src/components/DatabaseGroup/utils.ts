import type { SelectOption } from "naive-ui";
import type { Factor } from "@/plugins/cel";
import { useEnvironmentV1Store, useInstanceV1List } from "@/store";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
} from "@/utils";

export const FactorList: Factor[] = [
  "resource.environment_name",
  "resource.database_name",
  "resource.instance_id",
];

export const factorSupportDropdown: Factor[] = [
  "resource.environment_name",
  "resource.instance_id",
];

export const DatabaseGroupFactorOptionsMap = () => {
  return FactorList.reduce((map, factor) => {
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
    const environmentName = extractEnvironmentResourceName(env.name);
    return {
      label: environmentName,
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
      value: instanceId,
    };
  });
};
