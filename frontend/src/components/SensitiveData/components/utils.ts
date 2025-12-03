import { create } from "@bufbuild/protobuf";
import { uniq } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { h } from "vue";
import { getRenderOptionFunc } from "@/components/CustomApproval/Settings/components/common";
import { InstanceV1Name } from "@/components/v2";
import type { Factor, Operator } from "@/plugins/cel";
import { CollectionOperatorList, EqualityOperatorList } from "@/plugins/cel";
import { useSettingV1Store } from "@/store";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Algorithm } from "@/types/proto-es/v1/setting_service_pb";
import {
  AlgorithmSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { extractInstanceResourceName } from "@/utils";
import { CEL_ATTRIBUTE_RESOURCE_PROJECT_ID } from "@/utils/cel-attributes";

export const getClassificationLevelOptions = () => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName(
    Setting_SettingName.DATA_CLASSIFICATION
  );
  if (!setting || !setting.value) {
    return [];
  }

  const config =
    setting.value.value.case === "dataClassification"
      ? setting.value.value.value.configs
      : [];
  if (config.length === 0) {
    return [];
  }

  return config[0].levels.map<SelectOption>((level) => ({
    label: level.title,
    value: level.id,
  }));
};

export const getInstanceIdOptions = (instanceList: Instance[]) => {
  return instanceList.map<SelectOption>((ins) => {
    const instanceId = extractInstanceResourceName(ins.name);
    return {
      label: `${ins.title} (${instanceId})`,
      value: instanceId,
      render: getRenderOptionFunc({
        name: ins.name,
        title: () =>
          h(InstanceV1Name, {
            instance: ins,
            link: false,
          }),
      }),
    };
  });
};

export const factorOperatorOverrideMap = new Map<Factor, Operator[]>([
  [
    CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
    uniq([...EqualityOperatorList, ...CollectionOperatorList]),
  ],
]);

export type MaskingType =
  | "full-mask"
  | "range-mask"
  | "md5-mask"
  | "inner-outer-mask";

export const getMaskingType = (
  algorithm: Algorithm | undefined
): MaskingType | undefined => {
  if (!algorithm || !algorithm.mask) {
    return;
  }
  switch (algorithm.mask.case) {
    case "fullMask":
      return "full-mask";
    case "rangeMask":
      return "range-mask";
    case "innerOuterMask":
      return "inner-outer-mask";
    case "md5Mask":
      return "md5-mask";
    default:
      return;
  }
};

// Create masking algorithm utilities
export const createFullMaskAlgorithm = (substitution = "*"): Algorithm => {
  return create(AlgorithmSchema, {
    mask: {
      case: "fullMask",
      value: { substitution },
    },
  });
};

export const createRangeMaskAlgorithm = (
  slices: { start: number; end: number; substitution: string }[]
): Algorithm => {
  return create(AlgorithmSchema, {
    mask: {
      case: "rangeMask",
      value: { slices },
    },
  });
};

export const createMd5MaskAlgorithm = (salt = ""): Algorithm => {
  return create(AlgorithmSchema, {
    mask: {
      case: "md5Mask",
      value: { salt },
    },
  });
};

export const createInnerOuterMaskAlgorithm = (
  prefixLen = 0,
  suffixLen = 0,
  substitution = "*"
): Algorithm => {
  return create(AlgorithmSchema, {
    mask: {
      case: "innerOuterMask",
      value: { prefixLen, suffixLen, substitution },
    },
  });
};
