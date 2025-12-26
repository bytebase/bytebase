<template>
  <NDataTable
    key="iam-members"
    :columns="columns"
    :data="data"
    :row-key="(row: BindingRowData | UserRoleData) => row.key"
    :bordered="true"
    :striped="true"
    :cascade="false"
    allow-checking-not-loaded
    :checked-row-keys="selectedBindings"
    v-model:expanded-row-keys="expandedRowKeys"
    @load="onGroupLoad"
    @update:checked-row-keys="handleMemberSelection"
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import type {
  DataTableColumn,
  DataTableRowData,
  DataTableRowKey,
} from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import GroupMemberNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupMemberNameCell.vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { UserNameCell } from "@/components/v2/Model/cells";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { MemberBinding } from "../types";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

interface BindingRowData {
  isLeaf: boolean;
  data: MemberBinding;
  key: string;
  type: "binding";
  children?: UserRoleData[];
}

interface UserRoleData {
  data: User;
  key: string;
  type: "user";
}

const props = defineProps<{
  scope: "workspace" | "project";
  allowEdit: boolean;
  bindings: MemberBinding[];
  selectedBindings: string[];
  selectDisabled: (memberBinding: MemberBinding) => boolean;
  onClickUser?: (user: User, event: MouseEvent) => void;
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: MemberBinding): void;
  (event: "revoke-binding", binding: MemberBinding): void;
  (event: "update-selected-bindings", bindings: string[]): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const expandedRowKeys = ref<string[]>([]);

const data = computed((): BindingRowData[] => {
  return props.bindings.map((binding) => {
    return {
      data: binding,
      key: binding.binding,
      type: "binding",
      isLeaf:
        binding.type !== "groups" || (binding.group?.members.length ?? 0) === 0,
    };
  });
});

const onGroupLoad = async (row: DataTableRowData) => {
  const binding = (row as BindingRowData).data;
  if (binding.type !== "groups" || !binding.group?.members) {
    return;
  }

  await userStore.batchGetOrFetchUsers(
    binding.group.members.map((m) => m.member)
  );

  const children: UserRoleData[] = [];
  for (const member of binding.group.members) {
    const user = userStore.getUserByIdentifier(member.member);
    if (!user) {
      continue;
    }
    children.push({
      data: user,
      key: `${binding.group?.name}-${user.name}`,
      type: "user",
    });
  }

  row.children = orderBy(children, [(data) => data.data.name], ["desc"]);
};

const columns = computed(
  (): DataTableColumn<
    (BindingRowData | UserRoleData) & { hide?: boolean }
  >[] => {
    return [
      {
        type: "selection",
        hide: !props.allowEdit,
        disabled: (rowData: BindingRowData | UserRoleData) => {
          return (
            rowData.type == "user" ||
            props.selectDisabled((rowData as BindingRowData).data)
          );
        },
        cellProps: (rowData: BindingRowData | UserRoleData) => {
          if (rowData.type === "user") {
            return { style: { visibility: "hidden" } };
          }
          return {};
        },
      },
      {
        key: "account",
        title: t("settings.members.table.account"),
        resizable: true,
        className: "flex items-center",
        render: (rowData: BindingRowData | UserRoleData) => {
          if (rowData.type === "user") {
            return (
              <GroupMemberNameCell
                user={rowData.data}
                onClickUser={props.onClickUser}
              />
            );
          }

          const binding = (rowData as BindingRowData).data;
          if (binding.type === "groups") {
            const deleted = binding.group?.deleted ?? false;
            return (
              <GroupNameCell
                group={binding.group!}
                link={!deleted}
                deleted={deleted}
              />
            );
          }
          return (
            <UserNameCell
              user={binding.user ?? unknownUser()}
              onClickUser={props.onClickUser}
              allowEdit={false}
              showMfaEnabled={false}
            />
          );
        },
      },
      {
        key: "roles",
        title: t("settings.members.table.role"),
        resizable: true,
        render: (rowData: BindingRowData | UserRoleData) => {
          if (rowData.type === "user") {
            return null;
          }

          const binding = (rowData as BindingRowData).data;
          return h(UserRolesCell, {
            role: binding,
            key: binding.binding,
          });
        },
      },
      {
        key: "operations",
        title: "",
        width: "4rem",
        render: (rowData: BindingRowData | UserRoleData) => {
          if (rowData.type === "user") {
            return null;
          }

          const binding = (rowData as BindingRowData).data;
          return h(UserOperationsCell, {
            scope: props.scope,
            key: binding.binding,
            allowEdit: props.allowEdit,
            binding: binding,
            "onUpdate-binding": () => {
              emit("update-binding", binding);
            },
            "onRevoke-binding": () => {
              emit("revoke-binding", binding);
            },
          });
        },
      },
    ].filter((column) => !column.hide) as DataTableColumn<
      BindingRowData | UserRoleData
    >[];
  }
);

const handleMemberSelection = (rowKeys: DataTableRowKey[]) => {
  const members = rowKeys as string[];
  emit("update-selected-bindings", members);
};
</script>
