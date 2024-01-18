<template>
  <div class="w-full">
    <DatabaseGroupTable
      :database-group-list="databaseGroupList"
      :show-edit="allowEdit"
      @edit="handleConfigureDatabaseGroup"
    />
  </div>

  <DatabaseGroupPanel
    :show="state.showDatabaseGroupPanel"
    :project="project"
    :resource-type="'DATABASE_GROUP'"
    :database-group="state.editingDatabaseGroup"
    @close="state.showDatabaseGroupPanel = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, onMounted, reactive } from "vue";
import { useDBGroupStore } from "@/store";
import { ComposedProject } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";

interface LocalState {
  showDatabaseGroupPanel: boolean;
  editingDatabaseGroup?: DatabaseGroup;
}

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  showDatabaseGroupPanel: false,
});

const databaseGroupList = computed(() => {
  return dbGroupStore.getDBGroupListByProjectName(props.project.name);
});

onMounted(async () => {
  await dbGroupStore.getOrFetchDBGroupListByProjectName(props.project.name);
});

const handleConfigureDatabaseGroup = (databaseGroup: DatabaseGroup) => {
  state.editingDatabaseGroup = cloneDeep(databaseGroup);
  state.showDatabaseGroupPanel = true;
};
</script>
