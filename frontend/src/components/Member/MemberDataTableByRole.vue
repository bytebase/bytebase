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
import { displayRoleTitle, sortRoles } from "@/utils";
import UserNameCell from "./MemberDataTable/cells/UserNameCell.vue";
import UserOperationsCell from "./MemberDataTable/cells/UserOperationsCell.vue";
import type { MemberBinding } from "./types";

interface RoleRowData {
  type: "role";
  name: string;
  children: BindingRowData[];
}

interface BindingRowData {
  type: "binding";
  name: string;
  member: MemberBinding;
}

const props = defineProps<{
  allowEdit: boolean;
  bindings: MemberBinding[];
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: MemberBinding): void;
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
            allowEdit: props.allowEdit,
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
  const roles = sortRoles(roleStore.roleList.map((role) => role.name));
  const rowDataList: RoleRowData[] = [];

  for (const role of roles) {
    const members = props.bindings.filter((member) => {
      return (
        member.workspaceLevelRoles.includes(role) ||
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
