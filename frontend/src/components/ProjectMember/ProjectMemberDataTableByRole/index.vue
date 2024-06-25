<template>
  <NDataTable
    key="project-members-by-role"
    :columns="columns"
    :data="userListByRole"
    :row-key="(row) => row.name"
    :striped="true"
    :bordered="true"
    default-expand-all
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useRoleStore } from "@/store";
import type { ComposedProject } from "@/types";
import { PRESET_WORKSPACE_ROLES } from "@/types";
import { displayRoleTitle, sortRoles } from "@/utils";
import UserNameCell from "../ProjectMemberDataTable/cells/UserNameCell.vue";
import UserOperationsCell from "../ProjectMemberDataTable/cells/UserOperationsCell.vue";
import type { ProjectBinding } from "../types";

interface RoleRowData {
  type: "role";
  name: string;
  children: BindingRowData[];
}

interface BindingRowData {
  type: "binding";
  name: string;
  member: ProjectBinding;
}

const props = defineProps<{
  project: ComposedProject;
  bindings: ProjectBinding[];
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: ProjectBinding): void;
}>();

const { t } = useI18n();
const roleStore = useRoleStore();

const columns = computed(() => {
  return [
    {
      key: "role-members",
      title: `${t("common.role.self")} / ${t("common.members")}`,
      className: "flex items-center",
      render: (row: RoleRowData | BindingRowData) => {
        if (row.type === "role") {
          return h(
            "div",
            {
              class: "flex items-center",
            },
            [
              h(
                "span",
                {
                  class: "font-medium",
                },
                displayRoleTitle(row.name)
              ),
              h(
                "span",
                {
                  class: "ml-1 font-normal text-control-light",
                },
                `(${row.children.length})`
              ),
            ]
          );
        }

        if (row.member.type === "groups") {
          return <GroupNameCell group={row.member.group!} />;
        }

        return <UserNameCell projectMember={row.member} />;
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (row: RoleRowData | BindingRowData) => {
        if (row.type === "role") {
          return "";
        } else {
          return h(UserOperationsCell, {
            project: props.project,
            projectMember: row.member,
            "onUpdate-binding": () => {
              emit("update-binding", row.member);
            },
          });
        }
      },
    },
  ] as DataTableColumn<RoleRowData | BindingRowData>[];
});

const userListByRole = computed(() => {
  const roles = sortRoles(
    roleStore.roleList
      .map((role) => role.name)
      .filter((role) => !PRESET_WORKSPACE_ROLES.includes(role))
  );
  const rowDataList: RoleRowData[] = [];

  for (const role of roles) {
    const members = props.bindings.filter((member) => {
      return (
        member.workspaceLevelProjectRoles.includes(role) ||
        member.projectRoleBindings.find((binding) => binding.role === role)
      );
    });

    if (members.length > 0) {
      rowDataList.push({
        type: "role",
        name: role,
        children: members.map((member) => {
          return {
            type: "binding",
            name: member.binding,
            member,
          };
        }),
      });
    }
  }

  return rowDataList;
});
</script>
