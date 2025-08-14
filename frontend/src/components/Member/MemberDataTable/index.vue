<template>
  <NDataTable
    key="iam-members"
    :columns="columns"
    :data="bindings"
    :row-key="(row: MemberBinding) => row.binding"
    :bordered="true"
    :striped="true"
    :checked-row-keys="selectedBindings"
    @update:checked-row-keys="handleMemberSelection"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn, DataTableRowKey } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import GroupMemberNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupMemberNameCell.vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { useUserStore } from "@/store";
import { unknownUser } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { MemberBinding } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

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

const columns = computed(
  (): DataTableColumn<MemberBinding & { hide?: boolean }>[] => {
    return [
      {
        type: "selection",
        hide: !props.allowEdit,
        disabled: (memberBinding: MemberBinding) => {
          return props.selectDisabled(memberBinding);
        },
      },
      {
        type: "expand",
        hide: !props.bindings.some((binding) => binding.type === "groups"),
        expandable: (memberBinding: MemberBinding) =>
          memberBinding.type === "groups" &&
          !!memberBinding.group?.members.length,
        renderExpand: (memberBinding: MemberBinding) => {
          // Fetch user data for group members when expanded
          if (memberBinding.group?.members) {
            const memberUserIds = memberBinding.group.members.map(
              (m) => m.member
            );
            userStore.batchGetUsers(memberUserIds);
          }

          return (
            <div class="pl-20 space-y-2">
              {memberBinding.group?.members.map((member) => {
                const user =
                  userStore.getUserByIdentifier(member.member) ?? unknownUser();
                return (
                  <GroupMemberNameCell
                    key={`${memberBinding.group?.name}-${user.name}`}
                    user={user}
                    onClickUser={props.onClickUser}
                  />
                );
              })}
            </div>
          );
        },
      },
      {
        key: "account",
        title: t("settings.members.table.account"),
        width: "32rem",
        resizable: true,
        render: (memberBinding: MemberBinding) => {
          if (memberBinding.type === "groups") {
            const deleted = memberBinding.group?.deleted ?? false;
            return (
              <GroupNameCell
                group={memberBinding.group!}
                link={!deleted}
                deleted={deleted}
              />
            );
          }
          return (
            <UserNameCell
              binding={memberBinding}
              onClickUser={props.onClickUser}
            />
          );
        },
      },
      {
        key: "roles",
        title: t("settings.members.table.role"),
        resizable: true,
        render: (memberBinding: MemberBinding) => {
          return h(UserRolesCell, {
            role: memberBinding,
            key: memberBinding.binding,
          });
        },
      },
      {
        key: "operations",
        title: "",
        width: "4rem",
        render: (memberBinding: MemberBinding) => {
          return h(UserOperationsCell, {
            scope: props.scope,
            key: memberBinding.binding,
            allowEdit: props.allowEdit,
            binding: memberBinding,
            "onUpdate-binding": () => {
              emit("update-binding", memberBinding);
            },
            "onRevoke-binding": () => {
              emit("revoke-binding", memberBinding);
            },
          });
        },
      },
    ].filter((column) => !column.hide) as DataTableColumn<MemberBinding>[];
  }
);

const handleMemberSelection = (rowKeys: DataTableRowKey[]) => {
  const members = rowKeys as string[];
  emit("update-selected-bindings", members);
};
</script>
