<template>
  <NDataTable
    key="project-members"
    :columns="columns"
    :data="bindings"
    :row-key="(row: MemberBinding) => row.binding"
    :bordered="true"
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
import type { MemberBinding } from "../types";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

const props = defineProps<{
  allowEdit: boolean;
  bindings: MemberBinding[];
  selectedBindings: string[];
  selectDisabled: (projectMember: MemberBinding) => boolean;
}>();

const emit = defineEmits<{
  (event: "update-binding", binding: MemberBinding): void;
  (event: "update-selected-bindings", bindings: string[]): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const columns = computed(
  (): DataTableColumn<MemberBinding & { hide?: boolean }>[] => {
    return [
      {
        type: "selection",
        disabled: (projectMember: MemberBinding) => {
          return props.selectDisabled(projectMember);
        },
      },
      {
        type: "expand",
        hide: !props.bindings.some((binding) => binding.type === "groups"),
        expandable: (projectMember: MemberBinding) =>
          projectMember.type === "groups",
        renderExpand: (projectMember: MemberBinding) => {
          return (
            <div class="pl-20">
              {projectMember.group!.members.map((member) => {
                const user =
                  userStore.getUserByIdentifier(member.member) ?? unknownUser();
                return (
                  <GroupMemberNameCell
                    key={`${projectMember.group!.name}-${user.name}`}
                    user={user}
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
        render: (projectMember: MemberBinding) => {
          if (projectMember.type === "groups") {
            return <GroupNameCell group={projectMember.group!} />;
          }
          return <UserNameCell projectMember={projectMember} />;
        },
      },
      {
        key: "roles",
        title: t("settings.members.table.role"),
        resizable: true,
        render: (projectMember: MemberBinding) => {
          return h(UserRolesCell, {
            projectRole: projectMember,
          });
        },
      },
      {
        key: "operations",
        title: "",
        width: "4rem",
        render: (projectMember: MemberBinding) => {
          return h(UserOperationsCell, {
            allowEdit: props.allowEdit,
            projectMember,
            "onUpdate-binding": () => {
              emit("update-binding", projectMember);
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
