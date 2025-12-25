<template>
  <NDataTable
    key="member-by-group"
    :columns="columns"
    :data="userListByGroup"
    :row-key="(row) => row.name"
    :bordered="true"
    :loading="loading"
    :cascade="false"
    allow-checking-not-loaded
    :expanded-row-keys="expandedKeys"
    @update:expanded-row-keys="$emit('update:expanded-keys', $event as string[])"
    @load="onExpand"
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import type { DataTableColumn, DataTableRowData } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h, watchEffect } from "vue";
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
  isLeaf: boolean;
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
    expandedKeys?: string[];
    onClickUser?: (user: User, event: MouseEvent) => void;
  }>(),
  {
    showGroupRole: true,
    groupRoleMap: () => new Map(),
    onClickUser: undefined,
    expandedKeys: () => [],
  }
);

const emit = defineEmits<{
  (event: "update-group", group: Group): void;
  (event: "remove-group", group: Group): void;
  (event: "update:expanded-keys", keys: string[]): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const onExpand = async (row: DataTableRowData) => {
  if (row.type !== "group") {
    return;
  }
  await onGroupLoad(row as GroupRowData);
};

const onGroupLoad = async (row: GroupRowData) => {
  const { group } = row;
  const memberUserIds = group.members.map((m) => m.member);
  await userStore.batchGetOrFetchUsers(memberUserIds);

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

  row.children = orderBy(
    members,
    [
      (member) => (member.role === GroupMember_Role.OWNER ? 1 : 0),
      (member) => member.user.name,
    ],
    ["desc", "desc"]
  );
};

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
            "onRemove-group": () => {
              emit("remove-group", row.group);
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

  for (const group of props.groups) {
    rowDataList.push({
      type: "group",
      group,
      isLeaf: group.members.length === 0,
      name: group.name,
      children: [],
    });
  }

  return rowDataList;
});

watchEffect(async () => {
  for (const expandKey of props.expandedKeys) {
    const data = userListByGroup.value.find((row) => row.name === expandKey);
    if (!data) {
      continue;
    }
    await onGroupLoad(data);
  }
});
</script>
