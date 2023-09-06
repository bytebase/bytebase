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
  useSettingV1Store,
} from "@/store";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

export const getClassificationLevelOptions = () => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName(
    "bb.workspace.data-classification"
  );
  if (!setting) {
    return [];
  }
  const config = setting.value?.dataClassificationSettingValue?.configs ?? [];
  if (config.length === 0) {
    return [];
  }

  return config[0].levels.map<SelectOption>((level) => ({
    label: level.title,
    value: level.id,
  }));
};

export const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<SelectOption>((env) => {
    return {
      label: env.title,
      value: extractEnvironmentResourceName(env.name),
    };
  });
};

export const getInstanceIdOptions = () => {
  const { instanceList } = useInstanceV1List(false);
  return instanceList.value.map<SelectOption>((ins) => {
    return {
      label: ins.title,
      value: extractInstanceResourceName(ins.name),
    };
  });
};

export const getProjectIdOptions = () => {
  const { projectList } = useProjectV1ListByCurrentUser();
  return projectList.value.map<SelectOption>((proj) => ({
    label: proj.title,
    value: extractProjectResourceName(proj.name),
  }));
};

export const factorSupportDropdown: Factor[] = [
  "environment_id",
  "instance_id",
  "project_id",
  "classification_level",
];

export const factorOperatorOverrideMap = new Map<Factor, Operator[]>([
  ["project_id", uniq([...EqualityOperatorList, ...CollectionOperatorList])],
]);
