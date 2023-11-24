<template>
  <NSelect
    :value="state.selectedDatabaseGroup"
    :options="dbGroupOptions"
    :disabled="disabled"
    :placeholder="'Select Database Group'"
    @update:value="$emit('update:selected', $event)"
  />
</template>

<script lang="ts" setup>
import { SelectOption } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useDBGroupStore } from "@/store";
import { DatabaseGroup } from "@/types/proto/v1/project_service";

interface LocalState {
  selectedDatabaseGroup?: string;
}

const props = defineProps<{
  project: string;
  selected?: string;
  environment?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:selected", name: string | undefined): void;
}>();

const state = reactive<LocalState>({
  selectedDatabaseGroup: undefined,
});

const dbGroupStore = useDBGroupStore();

const dbGroupList = computed(() => {
  return dbGroupStore
    .getDBGroupListByProjectName(props.project)
    .filter((dbGroup) =>
      props.environment ? dbGroup.environment.uid === props.environment : true
    );
});
const dbGroupOptions = computed(() => {
  return dbGroupList.value.map<SelectOption>((dbGroup) => ({
    value: dbGroup.name,
    label: dbGroup.databaseGroupName,
  }));
});

const invalidateSelectionIfNeeded = () => {
  if (
    state.selectedDatabaseGroup &&
    !dbGroupList.value.find(
      (item: DatabaseGroup) => item.name == state.selectedDatabaseGroup
    )
  ) {
    state.selectedDatabaseGroup = undefined;
    emit("update:selected", undefined);
  }
};

watch(
  [() => props.project, () => props.selected, () => props.environment],
  () => {
    invalidateSelectionIfNeeded();
    state.selectedDatabaseGroup = dbGroupList.value.find(
      (item) => item.name === props.selected
    )?.name;
  },
  { immediate: true, deep: true }
);
</script>
