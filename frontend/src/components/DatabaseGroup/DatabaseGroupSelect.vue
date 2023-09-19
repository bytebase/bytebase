<template>
  <BBSelect
    :selected-item="state.selectedDatabaseGroup"
    :item-list="dbGroupList"
    :disabled="disabled"
    :placeholder="'Select Database Group'"
    :show-prefix-item="true"
    @select-item="(item: DatabaseGroup) => $emit('select-database-group-id', item.name)"
  >
    <template #menuItem="{ item }">
      {{ item.databasePlaceholder }}
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useDBGroupStore } from "@/store";
import { ComposedDatabaseGroup } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";

interface LocalState {
  selectedDatabaseGroup?: ComposedDatabaseGroup;
}

const props = defineProps<{
  projectId: string;
  selectedId?: string;
  environmentId?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "select-database-group-id", name?: string): void;
}>();

const state = reactive<LocalState>({
  selectedDatabaseGroup: undefined,
});

const dbGroupStore = useDBGroupStore();

const dbGroupList = computed(() => {
  return dbGroupStore
    .getDBGroupListByProjectName(props.projectId)
    .filter((dbGroup) =>
      props.environmentId
        ? dbGroup.environment.uid === props.environmentId
        : true
    );
});

const invalidateSelectionIfNeeded = () => {
  if (
    state.selectedDatabaseGroup &&
    !dbGroupList.value.find(
      (item: DatabaseGroup) => item.name == state.selectedDatabaseGroup?.name
    )
  ) {
    state.selectedDatabaseGroup = undefined;
    emit("select-database-group-id", undefined);
  }
};

watch(
  () => props,
  () => {
    invalidateSelectionIfNeeded();
    state.selectedDatabaseGroup = dbGroupList.value.find(
      (item) => item.name === props.selectedId
    );
  },
  { immediate: true, deep: true }
);
</script>
