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
      {{ getDatabaseGroupTitle(item.name) }}
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useDBGroupStore } from "@/store";
import { DatabaseGroup } from "@/types/proto/v1/project_service";

interface LocalState {
  selectedDatabaseGroup?: DatabaseGroup;
}

const props = defineProps<{
  selectedId?: string;
  projectId: string;
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
  return dbGroupStore.getDBGroupListByProjectName(props.projectId);
});

const getDatabaseGroupTitle = (databaseGroupName: string): string => {
  return databaseGroupName.split("/").pop() || "";
};

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
  () => props.selectedId,
  (selectedId) => {
    invalidateSelectionIfNeeded();
    state.selectedDatabaseGroup = dbGroupList.value.find(
      (item) => item.name === selectedId
    );
  },
  { immediate: true }
);
</script>
