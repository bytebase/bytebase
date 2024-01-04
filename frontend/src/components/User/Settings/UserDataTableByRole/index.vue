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
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { PresetRoleType } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { displayRoleTitle } from "@/utils";
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
  user: User;
}

const props = defineProps<{
  userList: User[];
}>();

const emit = defineEmits<{
  (event: "update-user", user: User): void;
}>();

const { t } = useI18n();

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
          user: row.user,
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
            user: row.user,
            "onUpdate-user": () => {
              emit("update-user", row.user);
            },
          });
        }
      },
    },
  ] as DataTableColumn<RoleRowData | UserRowData>[];
});

const userListByRole = computed(() => {
  const roles = Object.values(PresetRoleType);
  const rowDataList: RoleRowData[] = [];

  for (const role of roles) {
    const users = props.userList.filter((user) => {
      return user.roles.includes(role);
    });

    if (users.length > 0) {
      rowDataList.push({
        type: "role",
        name: role,
        children: users.map((user) => {
          return {
            type: "user",
            name: user.name,
            user,
          };
        }),
      });
    }
  }

  return rowDataList;
});
</script>
