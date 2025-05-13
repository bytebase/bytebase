<template>
  <NDataTable
    key="project-members-by-role"
    :columns="columns"
    :data="userListByRole"
    :row-key="(row) => row.name"
    :striped="true"
    :bordered="true"
    :max-height="'calc(100vh - 15rem)'"
    virtual-scroll
    default-expand-all
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import type { User } from "@/types/proto/v1/user_service";
import { displayRoleTitle } from "@/utils";
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
  scope: "workspace" | "project";
  allowEdit: boolean;
  bindingsByRole: Map<string, Map<string, MemberBinding>>;
  onClickUser?: (user: User, event: MouseEvent) => void;
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: MemberBinding): void;
  (event: "revoke-binding", binding: MemberBinding): void;
}>();

const { t } = useI18n();

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
          const deleted = row.member.group?.deleted ?? false;
          return (
            <GroupNameCell
              group={row.member.group!}
              link={!deleted}
              deleted={deleted}
            />
          );
        }

        return (
          <UserNameCell binding={row.member} onClickUser={props.onClickUser} />
        );
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
            scope: props.scope,
            allowEdit: props.allowEdit,
            binding: row.member,
            "onUpdate-binding": () => {
              emit("update-binding", row.member);
            },
            "onRevoke-binding": () => {
              emit("revoke-binding", row.member);
            },
          });
        }
      },
    },
  ] as DataTableColumn<RoleRowData | BindingRowData>[];
});

const userListByRole = computed(() => {
  const rowDataList: RoleRowData[] = [];

  for (const [role, memberBindings] of props.bindingsByRole.entries()) {
    const children: BindingRowData[] = [];

    for (const memberBinding of memberBindings.values()) {
      children.push({
        type: "binding",
        name: `${role}-${memberBinding.binding}`,
        member: memberBinding,
      });
    }
    if (children.length > 0) {
      rowDataList.push({
        type: "role",
        name: role,
        children,
      });
    }
  }

  return rowDataList;
});
</script>
