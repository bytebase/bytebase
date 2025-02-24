import { uniq } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { getRenderOptionFunc } from "@/components/CustomApproval/Settings/components/common";
import type { Factor, Operator } from "@/plugins/cel";
import { EqualityOperatorList, CollectionOperatorList } from "@/plugins/cel";
import { useInstanceResourceList, useSettingV1Store } from "@/store";
import type { Algorithm } from "@/types/proto/v1/setting_service";
import { extractInstanceResourceName } from "@/utils";

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

export const getInstanceIdOptions = () => {
  const instanceList = useInstanceResourceList();
  return instanceList.value.map<SelectOption>((ins) => {
    const instanceId = extractInstanceResourceName(ins.name);
    return {
      label: `${ins.title} (${instanceId})`,
      value: instanceId,
      render: getRenderOptionFunc(ins),
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
