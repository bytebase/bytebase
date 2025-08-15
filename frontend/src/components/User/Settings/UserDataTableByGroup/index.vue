<template>
  <NDataTable
    key="member-by-group"
    :columns="columns"
    :data="userListByGroup"
    :row-key="(row) => row.name"
    :bordered="true"
    :loading="loading"
    :default-expanded-row-keys="expandedRowKeys"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { orderBy } from "lodash-es";
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { GroupMember_Role } from "@/types/proto-es/v1/group_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import GroupMemberNameCell from "./cells/GroupMemberNameCell.vue";
import GroupNameCell from "./cells/GroupNameCell.vue";
import GroupOperationsCell from "./cells/GroupOperationsCell.vue";

interface GroupRowData {
  type: "group";
  name: string;
  group: Group;
  children: UserRowData[];
}

interface UserRowData {
  type: "user";
  name: string;
  user: User;
  role: GroupMember_Role;
}

const props = withDefaults(
  defineProps<{
    groups: Group[];
    loading: boolean;
    showGroupRole?: boolean;
    onClickUser?: (user: User, event: MouseEvent) => void;
  }>(),
  {
    showGroupRole: true,
    groupRoleMap: () => new Map(),
    onClickUser: undefined,
  }
);

const emit = defineEmits<{
  (event: "update-group", group: Group): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const expandedRowKeys = computed(() => props.groups.map((group) => group.name));

const columns = computed(() => {
  return [
    {
      key: "group-members",
      title: `${t("settings.members.groups.self")} / ${t("common.users")}`,
      className: "flex items-center",
      render: (row: GroupRowData | UserRowData) => {
        if (row.type === "group") {
          return <GroupNameCell group={row.group} link={false} />;
        }

        return (
          <GroupMemberNameCell
            user={row.user}
            role={props.showGroupRole ? row.role : undefined}
            onClickUser={props.onClickUser}
          />
        );
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
          });
        } else {
          return "";
        }
      },
    },
  ] as DataTableColumn<GroupRowData | UserRowData>[];
});

const userListByGroup = computedAsync(async () => {
  const rowDataList: GroupRowData[] = [];

  for (const group of props.groups) {
    // Fetch user data for all members in this group
    const memberUserIds = group.members.map((m) => m.member);
    if (memberUserIds.length > 0) {
      await userStore.batchGetUsers(memberUserIds);
    }

    const members: UserRowData[] = [];
    for (const member of group.members) {
      const user = userStore.getUserByIdentifier(member.member);
      if (!user) {
        continue;
      }
      members.push({
        type: "user",
        name: `${group.name}-${user.name}`,
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
          (member) => (member.role === GroupMember_Role.OWNER ? 1 : 0),
          (member) => member.user.name,
        ],
        ["desc", "desc"]
      ),
    });
  }

  return rowDataList;
}, []);
</script>
