<template>
  <NDataTable
    :size="size"
    :columns="accessTableColumns"
    :data="state.accessList"
    :row-key="(row: AccessUser) => row.key"
    :bordered="true"
    :striped="true"
    :max-height="'calc(100vh - 15rem)'"
    virtual-scroll
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import { TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDatePicker, NDataTable, useDialog } from "naive-ui";
import { computed, reactive, h, watchEffect } from "vue";
import type { VNodeChild } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import { useRouter } from "vue-router";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { MiniActionButton } from "@/components/v2";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import {
  usePolicyV1Store,
  usePolicyByParentAndType,
  useUserStore,
  useGroupStore,
  useDatabaseV1Store,
  pushNotification,
  extractGroupEmail,
} from "@/store";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto/v1/org_policy_service";
import type {
  Policy,
  MaskingExceptionPolicy_MaskingException,
} from "@/types/proto/v1/org_policy_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { autoDatabaseRoute, hasWorkspacePermissionV2 } from "@/utils";
import { convertFromCELString } from "@/utils/issue/cel";
import UserAvatar from "../User/UserAvatar.vue";
import MaskingLevelDropdown from "./components/MaskingLevelDropdown.vue";
import { type AccessUser } from "./types";

interface LocalState {
  processing: boolean;
  accessList: AccessUser[];
}

const props = defineProps<{
  size: "small" | "medium";
  disabled: boolean;
  // project full name
  project: string;
  showDatabaseColumn: boolean;
  filterException: (
    exception: MaskingExceptionPolicy_MaskingException
  ) => boolean;
}>();

const state = reactive<LocalState>({
  processing: false,
  accessList: [],
});
const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useGroupStore();
const databaseStore = useDatabaseV1Store();
const policyStore = usePolicyV1Store();
const $dialog = useDialog();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const policy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  }))
);

const getDatabaseAccessResource = (access: AccessUser): VNodeChild => {
  if (!access.databaseResource) {
    return <div class="textinfo">{t("database.all")}</div>;
  }
  const database = databaseStore.getDatabaseByName(
    access.databaseResource.databaseFullName
  );

  return (
    <div class="space-y-1">
      <div class="flex items-center gap-x-1 text-sm textinfo">
        <span>{`${t("common.instance")}:`}</span>
        <InstanceV1Name instance={database.instanceResource} />
      </div>
      <div class="flex items-center gap-x-1 text-sm textinfo">
        <span>{`${t("common.database")}:`}</span>
        <div
          class="normal-link hover:underline"
          onClick={() => {
            const query: Record<string, string> = {};
            if (access.databaseResource?.schema) {
              query.schema = access.databaseResource.schema;
            }
            if (access.databaseResource?.table) {
              query.table = access.databaseResource.table;
            }
            router.push({
              ...autoDatabaseRoute(router, database),
              query,
            });
          }}
        >
          <DatabaseV1Name database={database} link={false} />
        </div>
      </div>
      {access.databaseResource.schema && (
        <div class="text-sm textinfo">{`${t("common.schema")}: ${access.databaseResource.schema}`}</div>
      )}
      {access.databaseResource.table && (
        <div class="text-sm textinfo">{`${t("common.table")}: ${access.databaseResource.table}`}</div>
      )}
      {access.databaseResource.column && (
        <div class="text-sm textinfo">{`${t("database.column")}: ${access.databaseResource.column}`}</div>
      )}
    </div>
  );
};

const expirationTimeRegex = /request.time < timestamp\("(.+)?"\)/;

const getAccessUsers = async (
  exception: MaskingExceptionPolicy_MaskingException
): Promise<AccessUser | undefined> => {
  let expirationTimestamp: number | undefined;
  const expression = exception.condition?.expression ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    expirationTimestamp = new Date(matches[1]).getTime();
  }
  const conditionExpression = await convertFromCELString(expression);

  const access: AccessUser = {
    type: "user",
    key: exception.member,
    maskingLevel: exception.maskingLevel,
    expirationTimestamp,
    supportActions: new Set([exception.action]),
    rawExpression: expression,
    databaseResource: conditionExpression.databaseResources
      ? conditionExpression.databaseResources[0]
      : undefined,
  };

  if (exception.member.startsWith(groupBindingPrefix)) {
    access.type = "group";
    access.group = groupStore.getGroupByIdentifier(exception.member);
  } else {
    access.type = "user";
    access.user = userStore.getUserByIdentifier(exception.member);
  }

  if (!access.group && !access.user) {
    return;
  }

  return access;
};

const getExceptionIdentifier = (
  exception: MaskingExceptionPolicy_MaskingException
): string => {
  const expression = exception.condition?.expression ?? "";
  const res: string[] = [
    `level:"${maskingLevelToJSON(exception.maskingLevel)}"`,
    expression,
  ];
  return res.join(" && ");
};

const getMemberBinding = (access: AccessUser): string => {
  if (access.type === "user") {
    return getUserEmailInBinding(access.user!.email);
  }
  const email = extractGroupEmail(access.group!.name);
  return getGroupEmailInBinding(email);
};

const updateAccessUserList = async (policy: Policy | undefined) => {
  if (!policy || !policy.maskingExceptionPolicy) {
    return [];
  }

  // Exec data merge, we will merge data with same expiration time and level.
  // For example, the exception list and merge exec should be:
  // - 1. user1, action:export, level:FULL, expires at 2023-09-03
  // - 2. user1, action:export, level:FULL, expires at 2023-09-04
  // - 3. user1, action:export, level:PARTIAL, expires at 2023-09-04
  // - 4. user1, action:query, level:PARTIAL, expires at 2023-09-04
  // - 5. user1, action:query, level:FULL, expires at 2023-09-03
  // After the merge we should get:
  // - 1 & 5 is merged: user1, action:export+action, level:FULL, expires at 2023-09-03
  // - 2 cannot merge: user1, action:export, level:FULL, expires at 2023-09-04
  // - 3 & 4 is merged: user1, action:export+action, level:PARTIAL, expires at 2023-09-04
  const memberMap = new Map<string, AccessUser>();
  for (const exception of policy.maskingExceptionPolicy.maskingExceptions) {
    if (!props.filterException(exception)) {
      continue;
    }
    const identifier = getExceptionIdentifier(exception);
    const item = await getAccessUsers(exception);
    if (!item) {
      continue;
    }
    const id = `${getMemberBinding(item)}:${identifier}`;
    item.key = id;
    const target = memberMap.get(id) ?? item;
    if (memberMap.has(id)) {
      for (const action of item.supportActions) {
        target.supportActions.add(action);
      }
    }
    memberMap.set(id, target);
  }

  state.accessList = orderBy(
    [...memberMap.values()],
    [
      (access) => (access.type === "user" ? 1 : 0),
      (access) => {
        if (access.group) {
          return access.group.name;
        } else if (access.user) {
          return access.user.name;
        }
        return "";
      },
    ],
    ["desc", "desc"]
  );
};

watchEffect(async () => {
  updateAccessUserList(policy.value);
});

const accessTableColumns = computed(
  (): DataTableColumn<AccessUser & { hide?: boolean }>[] => {
    return [
      {
        type: "expand",
        expandable: (_: AccessUser) => true,
        hide: props.showDatabaseColumn,
        renderExpand: (item: AccessUser) => {
          return getDatabaseAccessResource(item);
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
        resizable: true,
        hide: !props.showDatabaseColumn,
        render: (item: AccessUser) => {
          return getDatabaseAccessResource(item);
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
              onUpdateChecked={() =>
                onAccessControlUpdate(
                  row,
                  (item) =>
                    toggleAction(
                      item,
                      MaskingExceptionPolicy_MaskingException_Action.EXPORT
                    ),
                  (item) =>
                    toggleAction(
                      item,
                      MaskingExceptionPolicy_MaskingException_Action.EXPORT
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
              onUpdate:checked={() =>
                onAccessControlUpdate(
                  row,
                  (item) =>
                    toggleAction(
                      item,
                      MaskingExceptionPolicy_MaskingException_Action.QUERY
                    ),
                  (item) =>
                    toggleAction(
                      item,
                      MaskingExceptionPolicy_MaskingException_Action.QUERY
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
              onUpdate:level={(e) => {
                if (e) {
                  onAccessControlUpdate(row, (item) => (item.maskingLevel = e));
                }
              }}
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
              actions={["confirm"]}
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
          return h(
            MiniActionButton,
            {
              onClick: () => {
                revokeAccessAlert(row);
              },
            },
            {
              default: () => h(TrashIcon, { class: "w-4 h-4" }),
            }
          );
        },
      },
    ].filter((column) => !column.hide) as DataTableColumn<AccessUser>[];
  }
);

const toggleAction = (
  item: AccessUser,
  action: MaskingExceptionPolicy_MaskingException_Action
) => {
  if (item.supportActions.has(action)) {
    item.supportActions.delete(action);
  } else {
    item.supportActions.add(action);
  }
};

const revokeAccessAlert = (
  index: number,
  revert: (item: AccessUser) => void = () => {}
) => {
  const item = state.accessList[index];
  $dialog.warning({
    title: t("common.warning"),
    content: () => {
      return (
        <div class="space-y-3">
          <div class="textlabel !text-base">
            {t("project.masking-access.revoke-access-title", {
              member: getMemberBinding(item),
            })}
          </div>
          {getDatabaseAccessResource(item)}
        </div>
      );
    },
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    closeOnEsc: false,
    maskClosable: false,
    onClose: () => revert(item),
    onPositiveClick: async () => {
      await onRemove(index);
    },
    onNegativeClick: () => revert(item),
  });
};

const onRemove = async (index: number) => {
  state.accessList.splice(index, 1);
  await onSubmit();
};

const onAccessControlUpdate = async (
  index: number,
  callback: (item: AccessUser) => void,
  revert: (item: AccessUser) => void = () => {}
) => {
  const item = state.accessList[index];
  if (!item) {
    return;
  }
  callback(item);
  if (item.supportActions.size === 0) {
    revokeAccessAlert(index, revert);
  } else {
    await onSubmit();
  }
};

const onSubmit = async () => {
  state.processing = true;

  try {
    await updateExceptionPolicy();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.processing = false;
  }
};

const updateExceptionPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const unChangedExceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter((exception) => !props.filterException(exception));

  for (const accessUser of state.accessList) {
    const expressions = accessUser.rawExpression.split(" && ");
    const index = expressions.findIndex((exp) =>
      exp.startsWith("request.time")
    );
    if (index >= 0) {
      if (!accessUser.expirationTimestamp) {
        expressions.splice(index, 1);
      } else {
        expressions[index] = `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`;
      }
    } else if (accessUser.expirationTimestamp) {
      expressions.push(
        `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`
      );
    }
    for (const action of accessUser.supportActions) {
      unChangedExceptions.push({
        maskingLevel: accessUser.maskingLevel,
        action,
        member: getMemberBinding(accessUser),
        condition: Expr.fromPartial({
          expression: expressions.join(" && "),
        }),
      });
    }
  }

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: unChangedExceptions,
  };
  await policyStore.upsertPolicy({
    parentPath: props.project,
    policy,
  });
};
</script>
