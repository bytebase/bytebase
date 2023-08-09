<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-between items-center">
      <span>{{ $t("database-group.self") }}</span>
      <div class="flex flex-row gap-x-2">
        <NButton @click="handleCreateSchemaGroup">
          <span class="mr-1">{{
            $t("database-group.table-group.create")
          }}</span>
          <FeatureBadge feature="bb.feature.database-grouping" />
        </NButton>
        <NButton @click="handleCreateDatabaseGroup">
          <span class="mr-1">{{ $t("database-group.create") }}</span>
          <FeatureBadge feature="bb.feature.database-grouping" />
        </NButton>
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

  <FeatureModal
    feature="bb.feature.database-grouping"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, onMounted, reactive } from "vue";
import { hasFeature, useDBGroupStore } from "@/store";
import { ComposedProject } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";
import { ResourceType } from "./common/ExprEditor/context";

interface LocalState {
  showFeatureModal: boolean;
  showDatabaseGroupPanel: boolean;
  resourceType: ResourceType;
  editingDatabaseGroup?: DatabaseGroup;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  showFeatureModal: false,
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
  if (!hasFeature("bb.feature.database-grouping")) {
    state.showFeatureModal = true;
    return;
  }

  state.resourceType = "DATABASE_GROUP";
  state.editingDatabaseGroup = undefined;
  state.showDatabaseGroupPanel = true;
};

const handleCreateSchemaGroup = () => {
  if (!hasFeature("bb.feature.database-grouping")) {
    state.showFeatureModal = true;
    return;
  }

  state.resourceType = "SCHEMA_GROUP";
  state.editingDatabaseGroup = undefined;
  state.showDatabaseGroupPanel = true;
};

const handleConfigureDatabaseGroup = (databaseGroup: DatabaseGroup) => {
  state.editingDatabaseGroup = cloneDeep(databaseGroup);
  state.showDatabaseGroupPanel = true;
};
</script>
