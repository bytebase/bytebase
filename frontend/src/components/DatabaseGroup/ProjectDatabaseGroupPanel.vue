<template>
  <DatabaseGroupDataTable
    :database-group-list="filteredDbGroupList"
    :custom-click="true"
    :loading="!ready"
    :show-actions="allowDelete"
    @row-click="handleDatabaseGroupClick"
    @delete="handleDelete"
  />
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import {
  useDBGroupListByProject,
  useDBGroupStore,
  useGracefulRequest,
} from "@/store";
import { getProjectNameAndDatabaseGroupName } from "@/store/modules/v1/common";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  project: Project;
  filter: string;
}>();

const router = useRouter();
const { t } = useI18n();
const dialog = useDialog();
const dbGroupStore = useDBGroupStore();

const { dbGroupList, ready } = useDBGroupListByProject(
  computed(() => props.project.name)
);

const allowDelete = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.databaseGroups.delete");
});

const filteredDbGroupList = computed(() => {
  const filter = props.filter.trim().toLowerCase();
  if (!filter) {
    return dbGroupList.value;
  }
  return dbGroupList.value.filter((group) => {
    return (
      group.name.toLowerCase().includes(filter) ||
      group.title.toLowerCase().includes(filter)
    );
  });
});

const handleDatabaseGroupClick = (
  event: MouseEvent,
  databaseGroup: DatabaseGroup
) => {
  const [projectId, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    databaseGroup.name
  );
  const url = router.resolve({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      projectId,
      databaseGroupName,
    },
  }).fullPath;
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const handleDelete = (databaseGroup: DatabaseGroup) => {
  dialog.warning({
    title: t("database-group.delete-group", { name: databaseGroup.title }),
    content: t("common.cannot-undo-this-action"),
    negativeText: t("common.cancel"),
    positiveText: t("common.delete"),
    onPositiveClick: () => {
      useGracefulRequest(() =>
        dbGroupStore.deleteDatabaseGroup(databaseGroup.name)
      );
    },
  });
};
</script>
