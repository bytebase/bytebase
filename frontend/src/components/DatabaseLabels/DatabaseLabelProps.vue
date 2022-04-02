<template>
  <template v-for="label in availableLabelList" :key="label.key">
    <dd class="flex items-center text-sm md:mr-4">
      <slot name="label" :label="label"></slot>
      <DatabaseLabelPropItem
        :label="label"
        :value="getLabelValue(label.key)"
        :label-list="availableLabelList"
        :required-label-list="requiredLabelList"
        :database="database"
        :allow-edit="allowEdit"
        @update:value="(value) => onUpdateValue(label.key, value)"
      />
    </dd>
  </template>
</template>

<script lang="ts" setup>
import { useLabelStore } from "@/store";
import { cloneDeep } from "lodash-es";
import { computed, withDefaults, watch, watchEffect, reactive } from "vue";
import {
  Database,
  DatabaseLabel,
  LabelKeyType,
  LabelValueType,
} from "../../types";
import { isReservedLabel, parseLabelListInTemplate } from "../../utils";
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

const labelStore = useLabelStore();

const prepareLabelList = () => {
  labelStore.fetchLabelList();
};
watchEffect(prepareLabelList);

watch(
  () => props.labelList,
  (list) => (state.labelList = cloneDeep(list))
);

const availableLabelList = computed(() => {
  const allList = labelStore.labelList;
  return allList.filter((label) => !isReservedLabel(label));
});

const requiredLabelList = computed(() => {
  const { project } = props.database;
  if (project.tenantMode !== "TENANT") return [];
  if (!project.dbNameTemplate) return [];

  return parseLabelListInTemplate(
    project.dbNameTemplate,
    availableLabelList.value
  );
});

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
