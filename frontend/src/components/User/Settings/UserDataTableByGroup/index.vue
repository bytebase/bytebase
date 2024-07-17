<template>
  <NDataTable
    key="member-by-group"
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
import { useUserStore } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import type { ComposedUser } from "@/types";
import { UserGroup, UserGroupMember_Role } from "@/types/proto/v1/user_group";
import GroupMemberNameCell from "./cells/GroupMemberNameCell.vue";
import GroupNameCell from "./cells/GroupNameCell.vue";
import GroupOperationsCell from "./cells/GroupOperationsCell.vue";

interface GroupRowData {
  type: "group";
  name: string;
  group: UserGroup;
  children: UserRowData[];
}

interface UserRowData {
  type: "user";
  name: string;
  user: ComposedUser;
  role: UserGroupMember_Role;
}

const props = withDefaults(
  defineProps<{
    groups: UserGroup[];
    showGroupRole?: boolean;
    allowEdit: boolean;
  }>(),
  {
    showGroupRole: true,
    groupRoleMap: () => new Map(),
  }
);

const emit = defineEmits<{
  (event: "update-group", group: UserGroup): void;
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
          return <GroupNameCell group={row.group} link={false} />;
        }

        return (
          <GroupMemberNameCell
            user={row.user}
            role={props.showGroupRole ? row.role : undefined}
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
            allowEdit: props.allowEdit,
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

const userListByGroup = computed(() => {
  const rowDataList: GroupRowData[] = [];

  for (const group of props.groups) {
    const members: UserRowData[] = [];
    for (const member of group.members) {
      const user = userStore.activeUserList.find(
        (user) => user.email === getUserEmailFromIdentifier(member.member)
      );
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
