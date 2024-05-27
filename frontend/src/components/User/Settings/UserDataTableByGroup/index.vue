<template>
  <NDataTable
    :columns="columns"
    :data="userListByGroup"
    :row-key="(row) => row.name"
    :bordered="true"
    :default-expanded-row-keys="expandedRowKeys"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, useDialog } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useUserGroupStore, pushNotification } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import type { User } from "@/types/proto/v1/auth_service";
import {
  UserGroup,
  type UserGroupMember_Role,
} from "@/types/proto/v1/user_group";
import GroupOperationsCell from "./cells/GroupOperationsCell.vue";
import UserNameCell from "./cells/UserNameCell.vue";

interface GroupRowData {
  type: "group";
  name: string;
  group: UserGroup;
  children: UserRowData[];
}

interface UserRowData {
  type: "user";
  name: string;
  user: User;
  role: UserGroupMember_Role;
}

const props = defineProps<{
  userList: User[];
}>();

const emit = defineEmits<{
  (event: "update-group", group: UserGroup): void;
}>();

const { t } = useI18n();
const groupStore = useUserGroupStore();
const $dialog = useDialog();

const expandedRowKeys = computed(() =>
  groupStore.groupList.map((group) => group.name)
);

const columns = computed(() => {
  return [
    {
      key: "group-members",
      title: `${t("settings.members.groups.self")} / ${t("common.members")}`,
      className: "flex items-center",
      render: (row: GroupRowData | UserRowData) => {
        if (row.type === "group") {
          return h("div", {}, [
            h(
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
                  row.group.title
                ),
                h(
                  "span",
                  {
                    class: "ml-1 font-normal text-control-light",
                  },
                  `(${row.children.length})`
                ),
              ]
            ),
            h(
              "span",
              { class: "textinfolabel text-sm" },
              row.group.description
            ),
          ]);
        }

        return h(UserNameCell, {
          user: row.user,
          role: row.role,
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (row: GroupRowData | UserRowData) => {
        if (row.type === "group") {
          return h(GroupOperationsCell, {
            group: row.group,
            "onUpdate-group": () => {
              emit("update-group", row.group);
            },
            "onDelete-group": () => {
              $dialog.warning({
                title: t("common.warning"),
                content: t("settings.members.groups.delete-warning", {
                  name: row.group.title,
                }),
                style: "z-index: 100000",
                negativeText: t("common.cancel"),
                positiveText: t("common.continue-anyway"),
                onPositiveClick: () => {
                  groupStore.deleteGroup(row.group.name).then(() => {
                    pushNotification({
                      module: "bytebase",
                      style: "SUCCESS",
                      title: t("common.deleted"),
                    });
                  });
                },
              });
            },
          });
        } else {
          return "";
        }
      },
    },
  ] as DataTableColumn<GroupRowData | UserRowData>[];
});

const userListByGroup = computed(() => {
  const rowDataList: GroupRowData[] = [];

  for (const group of groupStore.groupList) {
    const members: UserRowData[] = [];
    for (const member of group.members) {
      const user = props.userList.find(
        (user) => user.email === getUserEmailFromIdentifier(member.member)
      );
      if (!user) {
        continue;
      }
      members.push({
        type: "user",
        name: user.name,
        user,
        role: member.role,
      });
    }
    rowDataList.push({
      type: "group",
      group,
      name: group.name,
      children: members,
    });
  }

  return rowDataList;
});
</script>
