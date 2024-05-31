<template>
  <NDataTable
    key="project-members"
    :columns="columns"
    :data="bindings"
    :row-key="(row: ProjectBinding) => row.binding"
    :bordered="true"
    :checked-row-keys="selectedBindings"
    @update:checked-row-keys="handleMemberSelection"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn, DataTableRowKey } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import GroupMemberNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupMemberNameCell.vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useUserStore } from "@/store";
import { type ComposedProject, unknownUser } from "@/types";
import type { ProjectBinding } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

const props = defineProps<{
  project: ComposedProject;
  bindings: ProjectBinding[];
  selectedBindings: string[];
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: ProjectBinding): void;
  (event: "update-selected-bindings", bindings: string[]): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const columns = computed(
  (): DataTableColumn<ProjectBinding & { hide?: boolean }>[] => {
    return [
      {
        type: "selection",
        disabled: (projectMember: ProjectBinding) => {
          return projectMember.projectRoleBindings.length === 0;
        },
      },
      {
        type: "expand",
        hide: !props.bindings.some((binding) => binding.type === "groups"),
        expandable: (projectMember: ProjectBinding) =>
          projectMember.type === "groups",
        renderExpand: (projectMember: ProjectBinding) => {
          return (
            <div class="pl-20">
              {projectMember.group!.members.map((member) => {
                const user =
                  userStore.getUserByIdentifier(member.member) ?? unknownUser();
                return (
                  <GroupMemberNameCell
                    key={`${projectMember.group!.name}-${user.name}`}
                    user={user}
                  />
                );
              })}
            </div>
          );
        },
      },
      {
        key: "account",
        title: t("settings.members.table.account"),
        width: "32rem",
        resizable: true,
        render: (projectMember: ProjectBinding) => {
          if (projectMember.type === "groups") {
            return <GroupNameCell group={projectMember.group!} />;
          }
          return <UserNameCell projectMember={projectMember} />;
        },
      },
      {
        key: "roles",
        title: t("settings.members.table.role"),
        resizable: true,
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
            "onUpdate-binding": () => {
              emit("update-binding", projectMember);
            },
          });
        },
      },
    ].filter((column) => !column.hide) as DataTableColumn<ProjectBinding>[];
  }
);

const handleMemberSelection = (rowKeys: DataTableRowKey[]) => {
  const members = rowKeys as string[];
  emit("update-selected-bindings", members);
};
</script>
