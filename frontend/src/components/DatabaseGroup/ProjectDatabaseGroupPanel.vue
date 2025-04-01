<template>
  <DatabaseGroupDataTable
    :database-group-list="filteredDbGroupList"
    :custom-click="true"
    :loading="!ready"
    :show-actions="true"
    @row-click="handleDatabaseGroupClick"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { useDBGroupListByProject } from "@/store";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";

const props = defineProps<{
  project: ComposedProject;
  filter: string;
}>();

const router = useRouter();
const { dbGroupList, ready } = useDBGroupListByProject(props.project.name);

const filteredDbGroupList = computed(() => {
  const filter = props.filter.trim().toLowerCase();
  if (!filter) {
    return dbGroupList.value;
  }
  return dbGroupList.value.filter((group) => {
    return group.databaseGroupName.toLowerCase().includes(filter);
  });
});

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
