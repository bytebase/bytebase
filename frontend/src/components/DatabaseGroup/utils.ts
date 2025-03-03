import { uniq } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { type OptionConfig } from "@/components/ExprEditor/context";
import type { Factor } from "@/plugins/cel";
import { useEnvironmentV1Store } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedProject } from "@/types";
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

export const getDatabaseGroupOptionConfigMap = (project: ComposedProject) => {
  return FactorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "resource.environment_name":
        options = getEnvironmentOptions();
        break;
      case "resource.instance_id":
        options = getInstanceIdOptions(project);
        break;
    }
    map.set(factor, {
      remote: false,
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
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

const getInstanceIdOptions = (project: ComposedProject) => {
  const { databaseList } = useDatabaseV1List(project.name);
  return uniq(
    databaseList.value.map((d) => d.instanceResource.name)
  ).map<SelectOption>((instanceName) => {
    const instanceId = extractInstanceResourceName(instanceName);
    return {
      label: instanceId,
      value: instanceId,
    };
  });
};
