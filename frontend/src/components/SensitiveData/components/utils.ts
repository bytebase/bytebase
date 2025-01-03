import { uniq } from "lodash-es";
import type { SelectOption } from "naive-ui";
import type { Factor, Operator } from "@/plugins/cel";
import { EqualityOperatorList, CollectionOperatorList } from "@/plugins/cel";
import {
  useInstanceResourceList,
  useEnvironmentV1Store,
  useProjectV1List,
  useSettingV1Store,
} from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Algorithm } from "@/types/proto/v1/setting_service";
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
    const environmentName = extractEnvironmentResourceName(env.name);
    return {
      label: environmentName,
      value: environmentName,
    };
  });
};

export const getInstanceIdOptions = () => {
  const instanceList = useInstanceResourceList();
  return instanceList.value.map<SelectOption>((ins) => {
    const instanceId = extractInstanceResourceName(ins.name);
    return {
      label: instanceId,
      value: instanceId,
    };
  });
};

export const getProjectIdOptions = () => {
  const { projectList } = useProjectV1List();
  return projectList.value
    .filter((proj) => proj.name != DEFAULT_PROJECT_NAME)
    .map<SelectOption>((proj) => {
      const projectId = extractProjectResourceName(proj.name);
      return {
        label: `${projectId} (${proj.title})`,
        value: projectId,
      };
    });
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

export type MaskingType =
  | "full-mask"
  | "range-mask"
  | "md5-mask"
  | "inner-outer-mask";

export const getMaskingType = (
  algorithm: Algorithm | undefined
): MaskingType | undefined => {
  if (!algorithm) {
    return;
  }
  if (algorithm.fullMask) {
    return "full-mask";
  } else if (algorithm.rangeMask) {
    return "range-mask";
  } else if (algorithm.innerOuterMask) {
    return "inner-outer-mask";
  } else if (algorithm.md5Mask) {
    return "md5-mask";
  }
  return;
};
