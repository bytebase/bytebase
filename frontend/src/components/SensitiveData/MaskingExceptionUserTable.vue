<template>
  <NDataTable
    :size="size"
    :columns="accessTableColumns"
    :data="accessList"
    :row-key="(row: AccessUser) => row.key"
    :bordered="true"
    :striped="true"
    :max-height="'calc(100vh - 15rem)'"
    virtual-scroll
  />
</template>

<script lang="tsx" setup>
import { cloneDeep } from "lodash-es";
import { TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDatePicker, NPopconfirm, NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { MiniActionButton } from "@/components/v2";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import { MaskingLevel } from "@/types/proto/v1/common";
import { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import UserAvatar from "../User/UserAvatar.vue";
import MaskingLevelDropdown from "./components/MaskingLevelDropdown.vue";
import { type AccessUser } from "./types";

const props = defineProps<{
  size: "small" | "medium";
  accessList: AccessUser[];
  disabled: boolean;
  showDatabaseColumn: boolean;
}>();

const emit = defineEmits<{
  (event: "update:access", index: number, access: AccessUser): void;
  (event: "remove:access", index: number): void;
}>();

const { t } = useI18n();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const accessTableColumns = computed(
  (): DataTableColumn<AccessUser & { hide?: boolean }>[] => {
    return [
      {
        type: "expand",
        expandable: (_: AccessUser) => true,
        hide: props.showDatabaseColumn,
        renderExpand: (item: AccessUser) => {
          const expressions = item.rawExpression.split(" && ");
          if (
            expressions.length === 0 ||
            (expressions.length === 1 &&
              expressions[0].startsWith("request.time"))
          ) {
            expressions.push("all databases");
          }

          return (
            <ul class="list-disc pl-6 textinfolabel">
              {expressions.map((expression, i) => (
                <li key={`${item.key}.${i}`}>{expression}</li>
              ))}
            </ul>
          );
        },
      },
      {
        key: "member",
        title: t("common.members"),
        resizable: true,
        render: (item: AccessUser) => {
          if (item.type === "group") {
            return <GroupNameCell group={item.group!} />;
          }

          return (
            <div class="flex items-center gap-x-2">
              <UserAvatar size="SMALL" user={item.user} />
              <div class="flex flex-col">
                <RouterLink
                  to={{
                    name: WORKSPACE_ROUTE_USER_PROFILE,
                    params: {
                      principalEmail: item.user!.email,
                    },
                  }}
                  class="normal-link"
                >
                  {item.user!.title}
                </RouterLink>
                <span class="textinfolabel">{item.user!.email}</span>
              </div>
            </div>
          );
        },
      },
      {
        key: "resource",
        title: t("common.resource"),
        hide: !props.showDatabaseColumn,
        render: (item: AccessUser) => {
          const expressions = item.rawExpression.split(" && ");
          if (
            expressions.length === 0 ||
            (expressions.length === 1 &&
              expressions[0].startsWith("request.time"))
          ) {
            expressions.push("all databases");
          }

          return (
            <ul class="list-disc pl-6 textinfolabel">
              {expressions.map((expression, i) => (
                <li key={`${item.key}.${i}`}>{expression}</li>
              ))}
            </ul>
          );
        },
      },
      {
        key: "export",
        title: t("settings.sensitive-data.action.export"),
        width: "5rem",
        render: (item: AccessUser, row: number) => {
          return (
            <NCheckbox
              checked={item.supportActions.has(
                MaskingExceptionPolicy_MaskingException_Action.EXPORT
              )}
              disabled={!hasPermission.value || props.disabled}
              onUpdateChecked={(e) =>
                onAccessControlUpdate(row, (item) =>
                  toggleAction(
                    item,
                    MaskingExceptionPolicy_MaskingException_Action.EXPORT,
                    e
                  )
                )
              }
            />
          );
        },
      },
      {
        key: "query",
        title: t("settings.sensitive-data.action.query"),
        width: "5rem",
        render: (item: AccessUser, row: number) => {
          return (
            <NCheckbox
              checked={item.supportActions.has(
                MaskingExceptionPolicy_MaskingException_Action.QUERY
              )}
              disabled={!hasPermission.value || props.disabled}
              onUpdate:checked={(e) =>
                onAccessControlUpdate(row, (item) =>
                  toggleAction(
                    item,
                    MaskingExceptionPolicy_MaskingException_Action.QUERY,
                    e
                  )
                )
              }
            />
          );
        },
      },
      {
        key: "level",
        title: t("settings.sensitive-data.masking-level.self"),
        render: (item: AccessUser, row: number) => {
          return (
            <MaskingLevelDropdown
              disabled={!hasPermission.value || props.disabled}
              level={item.maskingLevel}
              levelList={[MaskingLevel.PARTIAL, MaskingLevel.NONE]}
              onUpdate:level={(e) =>
                onAccessControlUpdate(row, (item) => (item.maskingLevel = e))
              }
            />
          );
        },
      },
      {
        key: "expire",
        title: t("common.expiration"),
        render: (item: AccessUser, row: number) => {
          return (
            <NDatePicker
              value={item.expirationTimestamp}
              style={"width: 100%"}
              type={"datetime"}
              isDateDisabled={(date: number) => date < Date.now()}
              clearable={true}
              disabled={!hasPermission.value || props.disabled}
              onUpdate:value={(val: number | undefined) =>
                onAccessControlUpdate(
                  row,
                  (item) => (item.expirationTimestamp = val)
                )
              }
            />
          );
        },
      },
      {
        key: "operation",
        title: "",
        hide: !hasPermission.value,
        width: "4rem",
        render: (_: AccessUser, row: number) => {
          return (
            <NPopconfirm onPositiveClick={() => emit("remove:access", row)}>
              {{
                trigger: () => {
                  return (
                    <MiniActionButton
                      disabled={!hasPermission.value || props.disabled}
                    >
                      {{
                        default: () => <TrashIcon class="w-4 h-4" />,
                      }}
                    </MiniActionButton>
                  );
                },
                default: () => (
                  <div class="whitespace-nowrap">
                    {t(
                      "settings.sensitive-data.column-detail.remove-user-permission"
                    )}
                  </div>
                ),
              }}
            </NPopconfirm>
          );
        },
      },
    ].filter((column) => !column.hide) as DataTableColumn<AccessUser>[];
  }
);

const toggleAction = (
  item: AccessUser,
  action: MaskingExceptionPolicy_MaskingException_Action,
  checked: boolean
) => {
  if (checked) {
    item.supportActions.add(action);
  } else {
    item.supportActions.delete(action);
  }
};

const onAccessControlUpdate = async (
  index: number,
  callback: (item: AccessUser) => void
) => {
  const item = cloneDeep(props.accessList[index]);
  if (!item) {
    return;
  }
  callback(item);
  emit("update:access", index, item);
};
</script>
