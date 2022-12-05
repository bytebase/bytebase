<template>
  <template v-for="label in PRESET_LABEL_KEYS" :key="label.key">
    <dd class="flex items-center text-sm md:mr-4">
      <slot name="label" :label="label"></slot>
      <DatabaseLabelPropItem
        :label-key="label"
        :value="getLabelValue(label)"
        :required="isRequired(label)"
        :database="database"
        :allow-edit="allowEdit"
        @update:value="(value) => onUpdateValue(label, value)"
      />
    </dd>
  </template>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, withDefaults, watch, reactive } from "vue";
import type {
  Database,
  DatabaseLabel,
  LabelKeyType,
  LabelValueType,
} from "@/types";
import { parseLabelListInTemplate, PRESET_LABEL_KEYS } from "@/utils";
import DatabaseLabelPropItem from "./DatabaseLabelPropItem.vue";

const props = withDefaults(
  defineProps<{
    labelList: DatabaseLabel[];
    database: Database;
    allowEdit?: boolean;
  }>(),
  {
    allowEdit: false,
  }
);

const emit = defineEmits<{
  (event: "update:labelList", labels: DatabaseLabel[]): void;
}>();

const state = reactive({
  labelList: cloneDeep(props.labelList),
});

watch(
  () => props.labelList,
  (list) => (state.labelList = cloneDeep(list))
);

const requiredLabelList = computed((): string[] => {
  const { project } = props.database;
  if (project.tenantMode !== "TENANT") return [];
  if (!project.dbNameTemplate) return [];

  return parseLabelListInTemplate(project.dbNameTemplate);
});

const isRequired = (key: LabelKeyType) => {
  return requiredLabelList.value.includes(key);
};

const getLabelValue = (key: LabelKeyType): LabelValueType | undefined => {
  return state.labelList.find((label) => label.key === key)?.value || "";
};

const setLabelValue = (key: LabelKeyType, value: LabelValueType) => {
  const index = state.labelList.findIndex((label) => label.key === key);

  if (index < 0) {
    if (value) {
      // push new value
      state.labelList.push({ key, value });
    }
  } else {
    if (value) {
      state.labelList[index].value = value;
    } else {
      // remove empty value from the list
      state.labelList.splice(index, 1);
    }
  }
};

const onUpdateValue = (key: LabelKeyType, value: LabelValueType) => {
  setLabelValue(key, value);
  emit("update:labelList", state.labelList);
};
</script>
