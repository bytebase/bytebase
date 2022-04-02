<template>
  <div
    v-for="item in filteredFormItemList"
    :key="item.label.key"
    class="col-span-2 col-start-2 w-64"
  >
    <div class="flex flex-row items-center space-x-1">
      <label for="instance" class="textlabel capitalize">
        {{ hidePrefix(item.label.key) }}
        <span v-if="item.required" class="text-red-600">*</span>
      </label>
    </div>
    <div class="flex flex-col space-y-1">
      <BBSelect
        :selected-item="getLabelValue(item.label.key)"
        :item-list="getLabelValueList(item.label)"
        :placeholder="getLabelPlaceholder(item.label.key)"
        @select-item="(value: string) => setLabelValue(item.label.key, value)"
      >
        <template #menuItem="{ item: value }">
          {{ value === "" ? $t("label.empty-label-value") : value }}
        </template>
      </BBSelect>
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { useLabelStore } from "@/store";
import { capitalize } from "lodash-es";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import {
  DatabaseLabel,
  Label,
  LabelKeyType,
  LabelValueType,
  Project,
} from "../../types";
import {
  isReservedLabel,
  parseLabelListInTemplate,
  hidePrefix,
  validateLabelsWithTemplate,
} from "../../utils";

const props = defineProps<{
  project: Project;
  labelList: DatabaseLabel[];
  filter: "required" | "optional";
}>();

const labelStore = useLabelStore();
const { t } = useI18n();

const prepare = () => {
  labelStore.fetchLabelList();
};
watchEffect(prepare);

const isDbNameTemplateMode = computed((): boolean => {
  return !!props.project.dbNameTemplate;
});

const availableLabelList = computed(() => {
  const allLabelList = labelStore.labelList;
  // ignore reserved labels (e.g. bb.environment)
  return allLabelList.filter((label) => !isReservedLabel(label));
});

const requiredLabelDict = computed((): Set<LabelKeyType> => {
  if (!isDbNameTemplateMode.value) {
    // all labels are optional if we have no template
    return new Set();
  }

  // otherwise parse the placeholders from the template
  const labels = parseLabelListInTemplate(
    props.project.dbNameTemplate,
    availableLabelList.value
  );
  const keys = labels.map((label) => label.key);
  return new Set(keys);
});

const formItemList = computed((): { label: Label; required: boolean }[] => {
  return availableLabelList.value.map((label) => {
    const required = requiredLabelDict.value.has(label.key);
    return {
      label,
      required,
    };
  });
});

const filteredFormItemList = computed(
  (): { label: Label; required: boolean }[] => {
    return formItemList.value.filter((item) =>
      props.filter === "required" ? item.required : !item.required
    );
  }
);

const getLabelPlaceholder = (key: LabelKeyType): string => {
  // provide "Select Tenant" if Tenant is optional
  // provide "Select {{TENANT}}" if Tenant is required in the template
  key = requiredLabelDict.value.has(key)
    ? `{{${hidePrefix(key).toUpperCase()}}}`
    : capitalize(hidePrefix(key));
  return t("create-db.select-label-value", { key });
};

const getLabelValue = (key: LabelKeyType): LabelValueType | undefined => {
  return props.labelList.find((label) => label.key === key)?.value || "";
};

const getLabelValueList = (label: Label): LabelValueType[] => {
  const valueList = [...label.valueList];
  if (!requiredLabelDict.value.has(label.key)) {
    // for optional labels
    // provide a "<empty value>" option ahead of other values
    valueList.unshift("");
  }
  return valueList;
};

const setLabelValue = (key: LabelKeyType, value: LabelValueType) => {
  const label = props.labelList.find((label) => label.key === key);
  if (label) {
    label.value = value;
  } else {
    props.labelList.push({ key, value });
  }
};

defineExpose({
  // called by parent component
  validate: (): boolean => {
    return validateLabelsWithTemplate(props.labelList, requiredLabelDict.value);
  },
});
</script>
