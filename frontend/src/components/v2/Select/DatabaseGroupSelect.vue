<template>
  <NSelect
    :value="state.selectedDatabaseGroup"
    :options="dbGroupOptions"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="'Select Database Group'"
    @update:value="$emit('update:selected', $event)"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NSelect, type SelectOption } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useDBGroupListByProject } from "@/store";
import type { DatabaseGroup } from "@/types/proto/v1/database_group_service";

interface LocalState {
  selectedDatabaseGroup?: string;
}

const props = withDefaults(
  defineProps<{
    project: string;
    selected?: string;
    disabled?: boolean;
    clearable?: boolean;
    selectFirstAsDefault?: boolean;
  }>(),
  {
    clearable: false,
    selected: undefined,
    selectFirstAsDefault: true,
  }
);

const emit = defineEmits<{
  (event: "update:selected", name: string | undefined): void;
}>();

const state = reactive<LocalState>({
  selectedDatabaseGroup: undefined,
});
const { dbGroupList } = useDBGroupListByProject(props.project);

const dbGroupOptions = computed(() => {
  return dbGroupList.value.map<SelectOption>((dbGroup) => ({
    value: dbGroup.name,
    label: dbGroup.databaseGroupName,
  }));
});

watch(
  [() => props.project, () => props.selected],
  () => {
    state.selectedDatabaseGroup = dbGroupList.value.find(
      (item) => item.name === props.selected
    )?.name;
    if (!state.selectedDatabaseGroup && props.selectFirstAsDefault) {
      state.selectedDatabaseGroup = head(dbGroupList.value)?.name;
    }
    emit("update:selected", state.selectedDatabaseGroup);
  },
  { immediate: true, deep: true }
);
</script>
