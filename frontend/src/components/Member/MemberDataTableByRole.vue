<template>
  <NDataTable
    key="project-members-by-role"
    :columns="columns"
    :data="userListByRole"
    :row-key="(row) => row.name"
    :striped="true"
    :bordered="true"
    default-expand-all
  />
</template>

<script lang="tsx" setup>
import { Building2Icon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTooltip } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { displayRoleTitle, isBindingPolicyExpired } from "@/utils";
import UserNameCell from "./MemberDataTable/cells/UserNameCell.vue";
import UserOperationsCell from "./MemberDataTable/cells/UserOperationsCell.vue";
import type { MemberBinding } from "./types";

type Scope = "workspace" | "project";

interface RoleRowData {
  type: "role";
  name: string;
  scope: Scope;
  binding?: Binding;
  children: BindingRowData[];
}

interface BindingRowData {
  type: "binding";
  name: string;
  member: MemberBinding;
}

const props = defineProps<{
  scope: Scope;
  allowEdit: boolean;
  bindings: MemberBinding[];
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
          return (
            <div class="flex items-center space-x-1">
              {row.scope === "workspace" && (
                <NTooltip
                  v-slots={{
                    trigger: () => <Building2Icon class="w-4 h-auto" />,
                    default: () => t("project.members.workspace-level-roles"),
                  }}
                />
              )}
              <span
                class={`font-medium ${row.binding && isBindingPolicyExpired(row.binding) ? "line-through" : ""}`}
              >
                {displayRoleTitle(row.name)}
              </span>
              <span class="font-normal text-control-light">{`(${row.children.length})`}</span>
            </div>
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

const getRoleDataId = (roleData: RoleRowData): string => {
  const sections = [roleData.scope, roleData.name];
  if (roleData.binding && isBindingPolicyExpired(roleData.binding)) {
    sections.push("expired");
  }
  return sections.join(".");
};

const userListByRole = computed(() => {
  const map: Map<string, RoleRowData> = new Map();

  for (const memberBinding of props.bindings) {
    for (const role of memberBinding.workspaceLevelRoles) {
      const roleData: RoleRowData = {
        type: "role",
        name: role,
        scope: "workspace",
        children: [],
      };
      const id = getRoleDataId(roleData);
      if (!map.has(id)) {
        map.set(id, roleData);
      }
      map.get(id)?.children.push({
        type: "binding",
        name: `${role}-${memberBinding.binding}`,
        member: memberBinding,
      });
    }

    for (const binding of memberBinding.projectRoleBindings) {
      const roleData: RoleRowData = {
        type: "role",
        name: binding.role,
        scope: "project",
        children: [],
        binding,
      };
      const id = getRoleDataId(roleData);
      if (!map.has(id)) {
        map.set(id, roleData);
      }
      map.get(id)?.children.push({
        type: "binding",
        name: `${binding.role}-${memberBinding.binding}`,
        member: memberBinding,
      });
    }
  }

  return [...map.values()];
});
</script>
