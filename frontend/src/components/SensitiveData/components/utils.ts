import { uniq } from "lodash-es";
import type { SelectOption } from "naive-ui";
import {
  Factor,
  Operator,
  EqualityOperatorList,
  CollectionOperatorList,
} from "@/plugins/cel";
import {
  useEnvironmentV1Store,
  useInstanceV1List,
  useProjectV1ListByCurrentUser,
} from "@/store";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

export const factorList: Factor[] = [
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "instance_id", // using `instance.resource_id`
  "database_name",
  "table_name",
];

const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    return {
      label: env.title,
      value: extractEnvironmentResourceName(env.name),
    };
  });
};

const getInstanceIdOptions = () => {
  const { instanceList } = useInstanceV1List(false);
  return instanceList.value.map<SelectOption>((ins) => {
    return {
      label: ins.title,
      value: extractInstanceResourceName(ins.name),
    };
  });
};

const getProjectIdOptions = () => {
  const { projectList } = useProjectV1ListByCurrentUser();
  return projectList.value.map<SelectOption>((proj) => ({
    label: proj.title,
    value: extractProjectResourceName(proj.name),
  }));
};

export const getFactorOptionsMap = () => {
  return factorList.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "environment_id":
        options = getEnvironmentIdOptions();
        break;
      case "instance_id":
        options = getInstanceIdOptions();
        break;
      case "project_id":
        options = getProjectIdOptions();
        break;
    }
    map.set(factor, options);
    return map;
  }, new Map<Factor, SelectOption[]>());
};

export const factorSupportDropdown: Factor[] = [
  "environment_id",
  "instance_id",
  "project_id",
];

export const factorOperatorOverrideMap = new Map<Factor, Operator[]>([
  ["project_id", uniq([...EqualityOperatorList, ...CollectionOperatorList])],
]);
