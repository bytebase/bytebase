<template>
  <NDataTable
    :columns="columns"
    :data="userListByGroup"
    :row-key="(row) => row.name"
    :bordered="true"
    :default-expanded-row-keys="expandedRowKeys"
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import UserRolesCell from "@/components/ProjectMember/ProjectMemberDataTable/cells/UserRolesCell.vue";
import type { ProjectRole } from "@/components/ProjectMember/types";
import { useUserStore } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import { filterUserListByKeyword } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { UserGroup, UserGroupMember_Role } from "@/types/proto/v1/user_group";
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

const props = withDefaults(
  defineProps<{
    groups: UserGroup[];
    filter?: string;
    showDescription?: boolean;
    showGroupRole?: boolean;
    groupRoleMap?: Map<string, ProjectRole>;
    allowDelete: boolean;
    allowEdit: boolean;
  }>(),
  {
    filter: "",
    showDescription: true,
    showGroupRole: true,
    groupRoleMap: () => new Map(),
  }
);

const emit = defineEmits<{
  (event: "update-group", group: UserGroup): void;
  (event: "delete-group", group: UserGroup): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const expandedRowKeys = computed(() => props.groups.map((group) => group.name));

const columns = computed(() => {
  return [
    {
      key: "group-members",
      title: `${t("settings.members.groups.self")} / ${t("common.members")}`,
      className: "flex items-center",
      render: (row: GroupRowData | UserRowData) => {
        if (row.type === "group") {
          return (
            <div>
              <div class="flex items-center">
                <span class="font-medium">{row.group.title}</span>
                <span class="ml-1 font-normal text-control-light">
                  (
                  {t("settings.members.groups.n-members", {
                    n: row.children.length,
                  })}
                  )
                </span>
                {props.groupRoleMap.has(row.group.name) && (
                  <UserRolesCell
                    class="ml-3"
                    projectRole={props.groupRoleMap.get(row.group.name)!}
                  />
                )}
              </div>
              {props.showDescription && (
                <span class="textinfolabel text-sm">
                  {row.group.description}
                </span>
              )}
            </div>
          );
        }

        return h(UserNameCell, {
          user: row.user,
          role: row.role,
          showGroupRole: props.showGroupRole,
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
            allowDelete: props.allowDelete,
            allowEdit: props.allowEdit,
            "onUpdate-group": () => {
              emit("update-group", row.group);
            },
            "onDelete-group": () => {
              emit("delete-group", row.group);
            },
          });
        } else {
          return "";
        }
      },
    },
  ] as DataTableColumn<GroupRowData | UserRowData>[];
});

const filteredUserList = computed(() => {
  return filterUserListByKeyword(userStore.activeUserList, props.filter);
});

const userListByGroup = computed(() => {
  const rowDataList: GroupRowData[] = [];

  for (const group of props.groups) {
    const members: UserRowData[] = [];
    for (const member of group.members) {
      const user = filteredUserList.value.find(
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
      children: orderBy(
        members,
        [
          (member) => (member.role === UserGroupMember_Role.OWNER ? 1 : 0),
          (member) => member.user.name,
        ],
        ["desc", "desc"]
      ),
    });
  }

  return rowDataList;
});
</script>
