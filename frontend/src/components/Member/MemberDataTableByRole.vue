<template>
  <NDataTable
    key="project-members-by-role"
    :columns="columns"
    :data="userListByRole"
    :row-key="(row) => row.id"
    :bordered="true"
    :row-class-name="rowClassName"
    default-expand-all
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import { Building2Icon, Trash2Icon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NPopconfirm, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { UserNameCell } from "@/components/v2/Model/cells";
import { PresetRoleType, unknownUser } from "@/types";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { displayRoleTitle, isBindingPolicyExpired } from "@/utils";
import UserOperationsCell from "./MemberDataTable/cells/UserOperationsCell.vue";
import type { MemberBinding } from "./types";

type Scope = "workspace" | "project";

interface RoleRowData {
  id: string;
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
  (event: "revoke-role", role: string, expired: boolean): void;
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
            <div class="flex items-center gap-x-1">
              {row.scope === "workspace" && (
                <NTooltip
                  v-slots={{
                    trigger: () => <Building2Icon class="w-4 h-auto mr-1" />,
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
          <UserNameCell
            user={row.member.user ?? unknownUser()}
            onClickUser={props.onClickUser}
            allowEdit={false}
            showMfaEnabled={false}
          />
        );
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (row: RoleRowData | BindingRowData) => {
        if (row.type === "role") {
          return (
            row.scope === "project" &&
            props.allowEdit && (
              <NPopconfirm
                positiveButtonProps={{
                  type: "error",
                }}
                onPositiveClick={() =>
                  emit("revoke-role", row.name, row.id.endsWith(".expired"))
                }
                v-slots={{
                  trigger: () => (
                    <Trash2Icon class=" text-red-600 w-4 h-auto ml-auto mr-3 cursor-pointer" />
                  ),
                  default: () => t("settings.members.revoke-access-alert"),
                }}
              />
            )
          );
        } else {
          return (
            <UserOperationsCell
              class="ml-auto"
              scope={props.scope}
              allowEdit={props.allowEdit}
              binding={row.member}
              onUpdate-binding={() => {
                emit("update-binding", row.member);
              }}
              onRevoke-binding={() => {
                emit("revoke-binding", row.member);
              }}
            />
          );
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

  const getRoleRowData = (
    memberBinding: MemberBinding,
    data: { name: string; scope: Scope; binding?: Binding | undefined }
  ) => {
    const roleData: RoleRowData = {
      id: "",
      type: "role",
      children: [],
      ...data,
    };
    const id = getRoleDataId(roleData);
    roleData.id = id;
    if (!map.has(id)) {
      map.set(id, roleData);
    }
    map.get(id)?.children.push({
      type: "binding",
      name: `${id}-${memberBinding.binding}`,
      member: memberBinding,
    });
  };

  for (const memberBinding of props.bindings) {
    for (const role of memberBinding.workspaceLevelRoles) {
      getRoleRowData(memberBinding, {
        name: role,
        scope: "workspace",
      });
    }

    for (const binding of memberBinding.projectRoleBindings) {
      getRoleRowData(memberBinding, {
        name: binding.role,
        scope: "project",
        binding,
      });
    }
  }

  return orderBy(
    [...map.values()],
    [
      (row) => (row.scope === "workspace" ? 0 : 1),
      (row) => {
        if (
          Object.values(PresetRoleType).includes(row.name as PresetRoleType)
        ) {
          return Object.values(PresetRoleType).indexOf(
            row.name as PresetRoleType
          );
        }
        return Number.MAX_VALUE;
      },
    ],
    ["asc", "asc"]
  );
});

const rowClassName = (row: RoleRowData) => {
  return row.type === "role" ? "n-data-table-tr--striped" : "";
};
</script>
