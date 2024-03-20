<template>
  <NDataTable
    :columns="columns"
    :data="members"
    :row-key="(row) => row.user.email"
    :striped="true"
    :bordered="true"
    @update:checked-row-keys="handleMemberSelection"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, DataTableRowKey, NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedProject } from "@/types";
import { ProjectMember } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

const props = defineProps<{
  project: ComposedProject;
  members: ProjectMember[];
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
      disabled: (projectMember: ProjectMember) => {
        return projectMember.projectRoleBindings.length === 0;
      },
    },
    {
      key: "account",
      title: t("settings.members.table.account"),
      width: "32rem",
      render: (projectMember: ProjectMember) => {
        return h(UserNameCell, {
          projectMember,
        });
      },
    },
    {
      key: "roles",
      title: t("settings.members.table.role"),
      render: (projectMember: ProjectMember) => {
        return h(UserRolesCell, {
          projectMember,
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (projectMember: ProjectMember) => {
        return h(UserOperationsCell, {
          project: props.project,
          projectMember,
          "onUpdate-user": () => {
            emit("update-member", projectMember.user.email);
          },
        });
      },
    },
  ] as DataTableColumn<ProjectMember>[];
});

const handleMemberSelection = (rowKeys: DataTableRowKey[]) => {
  const members = rowKeys as string[];
  emit("update-selected-members", members);
};
</script>
