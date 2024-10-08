<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.database-grouping" />

    <DatabaseGroupDataTable
      :database-group-list="dbGroupList"
      :custom-click="true"
      :loading="!ready"
      :show-actions="true"
      @row-click="handleDatabaseGroupClick"
    />
  </div>

  <DatabaseGroupPanel
    :show="state.showDatabaseGroupPanel"
    :project="project"
    :database-group="state.editingDatabaseGroup"
    @close="state.showDatabaseGroupPanel = false"
  />

  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.database-grouping"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { useDBGroupListByProject } from "@/store";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";
import { FeatureAttention, FeatureModal } from "../FeatureGuard";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";

interface LocalState {
  showDatabaseGroupPanel: boolean;
  showFeatureModal: boolean;
  editingDatabaseGroup?: ComposedDatabaseGroup;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const router = useRouter();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showDatabaseGroupPanel: false,
});
const { dbGroupList, ready } = useDBGroupListByProject(props.project.name);

const handleDatabaseGroupClick = (
  event: MouseEvent,
  databaseGroup: ComposedDatabaseGroup
) => {
  const url = router.resolve({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      databaseGroupName: databaseGroup.databaseGroupName,
    },
  }).fullPath;
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
