<template>
  <div
    v-for="item in filteredFormItemList"
    :key="item.label"
    class="w-full"
    v-bind="$attrs"
  >
    <div class="flex flex-row items-center space-x-1">
      <label for="instance" class="textlabel capitalize">
        {{ hidePrefix(item.label) }}
        <span v-if="item.required" class="text-red-600">*</span>
      </label>
    </div>
    <div class="flex flex-col space-y-1">
      <BBTextField
        :required="item.required"
        :value="getLabelValue(item.label)"
        :placeholder="getLabelPlaceholder(item.label)"
        class="textfield"
        @input="
          setLabelValue(item.label, ($event.target as HTMLInputElement).value)
        "
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { capitalize } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  DatabaseLabel,
  LabelKeyType,
  LabelValueType,
  Project,
} from "@/types";
import {
  hidePrefix,
  validateLabelsWithTemplate,
  parseLabelListInTemplate,
  PRESET_LABEL_KEYS,
} from "@/utils";
import { BBTextField } from "@/bbkit";

const props = defineProps<{
  project: Project;
  labelList: DatabaseLabel[];
  filter: "required" | "optional";
}>();

const { t } = useI18n();

const isDbNameTemplateMode = computed((): boolean => {
  return !!props.project.dbNameTemplate;
});

const requiredLabelDict = computed((): Set<LabelKeyType> => {
  if (!isDbNameTemplateMode.value) {
    // all labels are optional if we have no template
    return new Set();
  }

  // otherwise parse the placeholders from the template
  const keys = parseLabelListInTemplate(props.project.dbNameTemplate);
  return new Set(keys);
});

const formItemList = computed(
  (): { label: LabelKeyType; required: boolean }[] => {
    return PRESET_LABEL_KEYS.map((label) => {
      const required = requiredLabelDict.value.has(label);
      return {
        label,
        required,
      };
    });
  }
);

const filteredFormItemList = computed(
  (): { label: LabelKeyType; required: boolean }[] => {
    return formItemList.value.filter((item) =>
      props.filter === "required" ? item.required : !item.required
    );
  }
);

const getLabelPlaceholder = (key: LabelKeyType): string => {
  // provide "Input Tenant" if Tenant is optional
  // provide "Input {{TENANT}}" if Tenant is required in the template
  key = requiredLabelDict.value.has(key)
    ? `{{${hidePrefix(key).toUpperCase()}}}`
    : capitalize(hidePrefix(key));
  return t("create-db.input-label-value", { key });
};

const getLabelValue = (key: LabelKeyType): LabelValueType | undefined => {
  return props.labelList.find((label) => label.key === key)?.value || "";
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
