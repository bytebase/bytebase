<template>
  <NDataTable
    :columns="columns"
    :data="members"
    :row-key="(row: ProjectBinding) => row.binding"
    :striped="true"
    :bordered="true"
    @update:checked-row-keys="handleMemberSelection"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn, DataTableRowKey } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedProject } from "@/types";
import type { ProjectBinding } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

const props = defineProps<{
  project: ComposedProject;
  members: ProjectBinding[];
  selectedMembers: string[];
}>();

const emit = defineEmits<{
  (event: "update-member", member: string): void;
  (event: "update-selected-members", members: string[]): void;
}>();

const { t } = useI18n();

const columns = computed(() => {
  return [
    {
      type: "selection",
      disabled: (projectMember: ProjectBinding) => {
        return projectMember.projectRoleBindings.length === 0;
      },
    },
    {
      key: "account",
      title: t("settings.members.table.account"),
      width: "32rem",
      render: (projectMember: ProjectBinding) => {
        return h(UserNameCell, {
          projectMember,
        });
      },
    },
    {
      key: "roles",
      title: t("settings.members.table.role"),
      render: (projectMember: ProjectBinding) => {
        return h(UserRolesCell, {
          projectRole: projectMember,
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (projectMember: ProjectBinding) => {
        return h(UserOperationsCell, {
          project: props.project,
          projectMember,
          "onUpdate-user": () => {
            emit("update-member", projectMember.binding);
          },
        });
      },
    },
  ] as DataTableColumn<ProjectBinding>[];
});

const handleMemberSelection = (rowKeys: DataTableRowKey[]) => {
  const members = rowKeys as string[];
  emit("update-selected-members", members);
};
</script>
