<template>
  <NDataTable
    :size="size"
    :columns="accessTableColumns"
    :data="filteredList"
    :row-key="(row: AccessUser) => row.key"
    :bordered="true"
    :striped="true"
    :loading="!ready || state.loading"
    :max-height="'calc(100vh - 15rem)'"
    virtual-scroll
  />
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import { TrashIcon, InfoIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import {
  NCheckbox,
  NDatePicker,
  NDataTable,
  useDialog,
  NTooltip,
} from "naive-ui";
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
  isValidDatabaseName,
  type ComposedProject,
} from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto-es/v1/org_policy_service_pb";
import type { MaskingExceptionPolicy_MaskingException } from "@/types/proto-es/v1/org_policy_service_pb";
import { MaskingExceptionPolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { autoDatabaseRoute, hasProjectPermissionV2 } from "@/utils";
import {
  type ConditionExpression,
  batchConvertFromCELString,
} from "@/utils/issue/cel";
import UserAvatar from "../User/UserAvatar.vue";
import { type AccessUser } from "./types";

interface LocalState {
  loading: boolean;
  processing: boolean;
  rawAccessList: AccessUser[];
}

const props = defineProps<{
  size: "small" | "medium";
  disabled: boolean;
  project: ComposedProject;
  showDatabaseColumn: boolean;
  filterAccessUser: (accessUser: AccessUser) => boolean;
}>();

const state = reactive<LocalState>({
  loading: true,
  processing: false,
  rawAccessList: [],
});
const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useGroupStore();
const databaseStore = useDatabaseV1Store();
const policyStore = usePolicyV1Store();
const $dialog = useDialog();

const hasPermission = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.policies.update");
});

const { policy, ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  }))
);

const filteredList = computed(() =>
  state.rawAccessList.filter(props.filterAccessUser)
);

const isValidDatabaseResource = (access: AccessUser): boolean => {
  if (!access.databaseResource) {
    return false;
  }
  const database = databaseStore.getDatabaseByName(
    access.databaseResource.databaseFullName
  );
  return isValidDatabaseName(database.name);
};

const getDatabaseAccessResource = (access: AccessUser): VNodeChild => {
  if (!access.databaseResource) {
    return <div class="textinfo">{t("database.all")}</div>;
  }
  const database = databaseStore.getDatabaseByName(
    access.databaseResource.databaseFullName
  );
  const validDatabase = isValidDatabaseResource(access);

  return (
    <div class="space-y-1">
      {validDatabase && (
        <div class="flex flex-col xl:flex-row xl:items-center gap-x-1 text-sm textinfo">
          <span class="font-medium">{`${t("common.instance")}:`}</span>
          <InstanceV1Name instance={database.instanceResource} />
        </div>
      )}
      <div class="flex flex-col xl:flex-row xl:items-center gap-x-1 text-sm textinfo">
        <span class="font-medium">{`${t("common.database")}:`}</span>
        {validDatabase ? (
          <div
            class="normal-link hover:underline cursor-pointer"
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
        ) : (
          <div class="flex items-center gap-x-1">
            <span class="line-through">
              {access.databaseResource.databaseFullName}
            </span>
            {!isValidDatabaseResource(access) && (
              <NTooltip>
                {{
                  trigger: () => <InfoIcon class="w-4 text-red-600" />,
                  default: () => t("database.not-found"),
                }}
              </NTooltip>
            )}
          </div>
        )}
      </div>
      {access.databaseResource.schema && (
        <div class="flex flex-col xl:flex-row xl:items-center gap-x-1 text-sm textinfo">
          <span class="font-medium">{`${t("common.schema")}:`}</span>
          <span>{access.databaseResource.schema}</span>
        </div>
      )}
      {access.databaseResource.table && (
        <div class="flex flex-col xl:flex-row xl:items-center gap-x-1 text-sm textinfo">
          <span class="font-medium">{`${t("common.table")}:`}</span>
          <span>{access.databaseResource.table}</span>
        </div>
      )}
      {access.databaseResource.columns &&
        access.databaseResource.columns.length > 0 && (
          <div class="flex flex-col xl:flex-row xl:items-center gap-x-1 text-sm textinfo">
            <span class="font-medium">{`${t("database.columns")}:`}</span>
            <span>{access.databaseResource.columns.join(", ")}</span>
          </div>
        )}
    </div>
  );
};

const expirationTimeRegex = /request.time < timestamp\("(.+)?"\)/;

const getAccessUsers = async (
  exception: MaskingExceptionPolicy_MaskingException,
  condition: ConditionExpression
): Promise<AccessUser | undefined> => {
  let expirationTimestamp: number | undefined;
  const expression = exception.condition?.expression ?? "";
  const description = exception.condition?.description ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    expirationTimestamp = new Date(matches[1]).getTime();
  }

  const access: AccessUser = {
    type: "user",
    key: `${exception.member}:${expression}.${description}`,
    expirationTimestamp,
    supportActions: new Set([exception.action]),
    rawExpression: expression,
    description,
    databaseResource: condition.databaseResources
      ? condition.databaseResources[0]
      : undefined,
  };

  if (exception.member.startsWith(groupBindingPrefix)) {
    access.type = "group";
    access.group = groupStore.getGroupByIdentifier(exception.member);
  } else {
    access.type = "user";
    access.user = await userStore.getOrFetchUserByIdentifier(exception.member);
  }

  if (!access.group && !access.user) {
    return;
  }

  return access;
};

const getMemberBinding = (access: AccessUser): string => {
  if (access.type === "user") {
    return getUserEmailInBinding(access.user!.email);
  }
  const email = extractGroupEmail(access.group!.name);
  return getGroupEmailInBinding(email);
};

const updateAccessUserList = async () => {
  if (!ready.value) {
    return;
  }

  if (!policy.value || policy.value.policy?.case !== "maskingExceptionPolicy") {
    state.rawAccessList = [];
    state.loading = false;
    return;
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
  const { maskingExceptions } = policy.value.policy.value;
  const expressionList = maskingExceptions.map((e) =>
    e.condition?.expression ? e.condition?.expression : "true"
  );
  const conditionList = await batchConvertFromCELString(expressionList);

  for (let i = 0; i < maskingExceptions.length; i++) {
    const exception = maskingExceptions[i];
    const condition = conditionList[i];

    const item = await getAccessUsers(exception, condition);
    if (!item) {
      continue;
    }

    const target = memberMap.get(item.key) ?? item;
    if (memberMap.has(item.key)) {
      for (const action of item.supportActions) {
        target.supportActions.add(action);
      }
    }
    memberMap.set(item.key, target);
  }

  state.rawAccessList = orderBy(
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
  state.loading = false;
};

watchEffect(updateAccessUserList);

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
        render: (item: AccessUser) => {
          return (
            <NCheckbox
              checked={item.supportActions.has(
                MaskingExceptionPolicy_MaskingException_Action.EXPORT
              )}
              disabled={!hasPermission.value || props.disabled}
              onUpdateChecked={() =>
                onAccessControlUpdate(
                  item,
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
        render: (item: AccessUser) => {
          return (
            <NCheckbox
              checked={item.supportActions.has(
                MaskingExceptionPolicy_MaskingException_Action.QUERY
              )}
              disabled={!hasPermission.value || props.disabled}
              onUpdate:checked={() =>
                onAccessControlUpdate(
                  item,
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
        key: "expire",
        title: t("common.expiration"),
        render: (item: AccessUser) => {
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
                  item,
                  (item) => (item.expirationTimestamp = val)
                )
              }
            />
          );
        },
      },
      {
        key: "reason",
        title: t("common.reason"),
        render: (item: AccessUser) => {
          return item.description;
        },
      },
      {
        key: "operation",
        title: "",
        hide: !hasPermission.value,
        width: "4rem",
        render: (item: AccessUser) => {
          return h(
            MiniActionButton,
            {
              onClick: () => {
                revokeAccessAlert(item);
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
  item: AccessUser,
  revert: (item: AccessUser) => void = () => {}
) => {
  $dialog.warning({
    title: t("common.warning"),
    content: () => {
      return (
        <div class="space-y-3">
          <div class="textlabel !text-base">
            {t("project.masking-exemption.revoke-exemption-title", {
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
      await onRemove(item);
    },
    onNegativeClick: () => revert(item),
  });
};

const onRemove = async (item: AccessUser) => {
  const index = state.rawAccessList.findIndex((a) => a.key === item.key);
  if (index < 0) {
    return;
  }
  state.rawAccessList.splice(index, 1);
  await onSubmit();
};

const onAccessControlUpdate = async (
  item: AccessUser,
  callback: (item: AccessUser) => void,
  revert: (item: AccessUser) => void = () => {}
) => {
  callback(item);
  if (item.supportActions.size === 0) {
    revokeAccessAlert(item, revert);
  } else {
    await onSubmit();
  }
};

const onSubmit = async () => {
  if (state.processing) {
    return;
  }
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
    parentPath: props.project.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = [];
  for (const accessUser of state.rawAccessList) {
    const expressions = accessUser.rawExpression
      .split(" && ")
      .filter((expression) => expression);
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
    const member = getMemberBinding(accessUser);
    for (const action of accessUser.supportActions) {
      exceptions.push({
        action,
        member,
        condition: create(ExprSchema, {
          description: accessUser.description,
          expression: expressions.join(" && "),
        }),
      });
    }
  }

  policy.policy = {
    case: "maskingExceptionPolicy",
    value: create(MaskingExceptionPolicySchema, {
      maskingExceptions: exceptions,
    }),
  };
  await policyStore.upsertPolicy({
    parentPath: props.project.name,
    policy,
  });
};
</script>
