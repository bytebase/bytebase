<template>
  <span class="textlabel relative inline-flex items-center">
    <template v-if="!editable">
      {{ value || $t("label.empty-label-value") }}
    </template>
    <template v-else>
      <select
        class="absolute inset-0 opacity-0 m-0 p-0"
        :value="value"
        @change="onChange"
      >
        <option :value="''" :selected="!!value">
          {{ $t("label.empty-label-value") }}
        </option>
        <option
          v-for="(labelValue, i) in label.valueList"
          :key="i"
          :value="labelValue"
        >
          {{ labelValue }}
        </option>
      </select>
      {{ value || $t("label.empty-label-value") }}
      <heroicons-outline:chevron-down class="w-4 h-4 ml-0.5" />
    </template>
  </span>
</template>

<script lang="ts" setup>
import { computed, defineProps, defineEmits } from "vue";
import { Database, Label, LabelValueType } from "../../types";

const props = defineProps<{
  label: Label;
  labelList: Label[];
  requiredLabelList: Label[];
  value: LabelValueType | undefined;
  database: Database;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (e: "update:value", value: LabelValueType): void;
}>();

const editable = computed((): boolean => {
  if (!props.allowEdit) return false;

  // Not editable if this is a required label in `dbNameTemplate`
  // e.g. tenant in "{{DB_NAME}}_{{TENANT}}"
  const isRequired = !!props.requiredLabelList.find(
    (label) => label.key === props.label.key
  );
  return !isRequired;
});

const onChange = (e: Event) => {
  const select = e.target as HTMLSelectElement;
  const { value } = select;
  emit("update:value", value);
};
</script>
