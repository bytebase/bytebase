<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-between items-center">
      <span>Database groups</span>
      <div class="flex flex-row gap-x-2">
        <NButton @click="handleCreateSchemaGroup">New table group</NButton>
        <NButton @click="handleCreateDatabaseGroup">New database group</NButton>
      </div>
    </div>
    <div class="mt-4">
      <DatabaseGroupTable
        :database-group-list="databaseGroupList"
        :show-edit="true"
        @edit="handleConfigureDatabaseGroup"
      />
    </div>
  </div>

  <DatabaseGroupPanel
    v-if="state.showDatabaseGroupPanel"
    :project="project"
    :resource-type="state.resourceType"
    :database-group="state.editingDatabaseGroup"
    @close="state.showDatabaseGroupPanel = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, onMounted, reactive } from "vue";
import { useDBGroupStore } from "@/store";
import { ComposedProject } from "@/types";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { ResourceType } from "./common/ExprEditor/context";

interface LocalState {
  showDatabaseGroupPanel: boolean;
  resourceType: ResourceType;
  editingDatabaseGroup?: DatabaseGroup;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  showDatabaseGroupPanel: false,
  resourceType: "DATABASE_GROUP",
});

const databaseGroupList = computed(() => {
  return dbGroupStore.getDBGroupListByProjectName(props.project.name);
});

onMounted(async () => {
  await dbGroupStore.getOrFetchDBGroupListByProjectName(props.project.name);
});

const handleCreateDatabaseGroup = () => {
  state.resourceType = "DATABASE_GROUP";
  state.editingDatabaseGroup = undefined;
  state.showDatabaseGroupPanel = true;
};

const handleConfigureDatabaseGroup = (databaseGroup: DatabaseGroup) => {
  state.editingDatabaseGroup = cloneDeep(databaseGroup);
  state.showDatabaseGroupPanel = true;
};

const handleCreateSchemaGroup = () => {
  state.resourceType = "SCHEMA_GROUP";
  state.editingDatabaseGroup = undefined;
  state.showDatabaseGroupPanel = true;
};
</script>
