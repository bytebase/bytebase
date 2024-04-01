<template>
  <NSelect
    :value="value ?? null"
    :options="options"
    :placeholder="placeholder ?? $t('database-group.select')"
    :virtual-scroll="true"
    :filter="filterByName"
    :filterable="true"
    class="bb-databaseGroup-select"
    style="width: 12rem"
    v-bind="$attrs"
    :render-label="renderLabel"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption, SelectRenderLabel } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, h, watch } from "vue";
import { useSlots } from "vue";
import { useDBGroupStore } from "@/store";
import type { ComposedDatabaseGroup } from "@/types";

interface DatabaseGroupSelectOption extends SelectOption {
  value: string;
  databaseGroup: ComposedDatabaseGroup;
}

const slots = useSlots();
const props = withDefaults(
  defineProps<{
    value?: string;
    project: string;
    placeholder?: string;
  }>(),
  {
    value: undefined,
    project: undefined,
    placeholder: undefined,
  }
);

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const dbGroupStore = useDBGroupStore();

const databaseGroupList = computed(() => {
  return dbGroupStore.getDBGroupListByProjectName(props.project);
});

watch(
  () => props.project,
  async (project) =>
    await dbGroupStore.getOrFetchDBGroupListByProjectName(project),
  { immediate: true }
);

const filterByName = (pattern: string, option: SelectOption) => {
  const { databaseGroup } = option as DatabaseGroupSelectOption;
  return databaseGroup.databaseGroupName
    .toLowerCase()
    .includes(pattern.toLowerCase());
};

const options = computed(() => {
  return databaseGroupList.value.map<DatabaseGroupSelectOption>(
    (databaseGroup) => {
      return {
        databaseGroup,
        value: databaseGroup.name,
        label: databaseGroup.databaseGroupName,
      };
    }
  );
});

const renderLabel: SelectRenderLabel = (option) => {
  const { databaseGroup } = option as DatabaseGroupSelectOption;
  if (!databaseGroup) {
    return;
  }

  if (slots.default) {
    return slots.default({ databaseGroup });
  }

  const children = [h("div", {}, [databaseGroup.databaseGroupName])];
  return h(
    "div",
    {
      class: "w-full flex flex-row justify-start items-center truncate",
    },
    children
  );
};
</script>
