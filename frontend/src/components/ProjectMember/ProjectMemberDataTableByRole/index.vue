<template>
  <NDataTable
    :columns="columns"
    :data="userListByRole"
    :row-key="(row) => row.name"
    :striped="true"
    :bordered="true"
    default-expand-all
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoleStore } from "@/store";
import type { ComposedProject } from "@/types";
import { PRESET_WORKSPACE_ROLES } from "@/types";
import { displayRoleTitle, sortRoles } from "@/utils";
import type { ProjectMember } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";

interface RoleRowData {
  type: "role";
  name: string;
  children: UserRowData[];
}

interface UserRowData {
  type: "user";
  name: string;
  member: ProjectMember;
}

const props = defineProps<{
  project: ComposedProject;
  members: ProjectMember[];
}>();

const emit = defineEmits<{
  (event: "update-member", member: string): void;
}>();

const { t } = useI18n();
const roleStore = useRoleStore();

const columns = computed(() => {
  return [
    {
      key: "role-members",
      title: `${t("common.role.self")} / ${t("common.members")}`,
      className: "flex items-center",
      render: (row: RoleRowData | UserRowData) => {
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

        return h(UserNameCell, {
          projectMember: row.member,
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (row: RoleRowData | UserRowData) => {
        if (row.type === "role") {
          return "";
        } else {
          return h(UserOperationsCell, {
            project: props.project,
            projectMember: row.member,
            "onUpdate-user": () => {
              emit("update-member", row.member.user.email);
            },
          });
        }
      },
    },
  ] as DataTableColumn<RoleRowData | UserRowData>[];
});

const userListByRole = computed(() => {
  const roles = sortRoles(
    roleStore.roleList
      .map((role) => role.name)
      .filter((role) => !PRESET_WORKSPACE_ROLES.includes(role))
  );
  const rowDataList: RoleRowData[] = [];

  for (const role of roles) {
    const members = props.members.filter((member) => {
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
            type: "user",
            name: member.user.name,
            member,
          };
        }),
      });
    }
  }

  return rowDataList;
});
</script>
