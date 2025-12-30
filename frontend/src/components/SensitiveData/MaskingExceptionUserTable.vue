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
import { InfoIcon, TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NDatePicker, NTooltip, useDialog } from "naive-ui";
import type { VNodeChild } from "vue";
import { computed, h, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import {
  DatabaseV1Name,
  InstanceV1Name,
  MiniActionButton,
} from "@/components/v2";
import { UserLink } from "@/components/v2/Model/cells";
import {
  composePolicyBindings,
  extractUserId,
  pushNotification,
  useDatabaseV1Store,
  useGroupStore,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import { groupBindingPrefix, isValidDatabaseName } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { autoDatabaseRoute, hasProjectPermissionV2 } from "@/utils";
import {
  batchConvertFromCELString,
  type ConditionExpression,
} from "@/utils/issue/cel";
import { type AccessUser } from "./types";

interface LocalState {
  loading: boolean;
  processing: boolean;
  rawAccessList: AccessUser[];
}

const props = defineProps<{
  size: "small" | "medium";
  disabled: boolean;
  project: Project;
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
    policyType: PolicyType.MASKING_EXEMPTION,
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
    <div class="flex flex-col gap-y-1">
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

const getAccessUsers = (
  exception: MaskingExemptionPolicy_Exemption,
  condition: ConditionExpression
): AccessUser[] => {
  let expirationTimestamp: number | undefined;
  const expression = exception.condition?.expression ?? "";
  const description = exception.condition?.description ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    expirationTimestamp = new Date(matches[1]).getTime();
  }

  const result: AccessUser[] = [];
  for (const member of exception.members) {
    const access: AccessUser = {
      type: member.startsWith(groupBindingPrefix) ? "group" : "user",
      member,
      key: `${member}:${expression}.${description}`,
      expirationTimestamp,
      rawExpression: expression,
      description,
      databaseResource: condition.databaseResources
        ? condition.databaseResources[0]
        : undefined,
    };

    result.push(access);
  }

  return result;
};

const updateAccessUserList = async () => {
  if (!ready.value) {
    return;
  }

  if (!policy.value || policy.value.policy?.case !== "maskingExemptionPolicy") {
    state.rawAccessList = [];
    state.loading = false;
    return;
  }

  const memberMap = new Map<string, AccessUser>();
  const { exemptions } = policy.value.policy.value;
  const expressionList = exemptions.map((e) =>
    e.condition?.expression ? e.condition?.expression : "true"
  );
  const conditionList = await batchConvertFromCELString(expressionList);

  await composePolicyBindings(exemptions, true);
  for (let i = 0; i < exemptions.length; i++) {
    const exception = exemptions[i];
    const condition = conditionList[i];

    const items = getAccessUsers(exception, condition);
    for (const item of items) {
      memberMap.set(item.key, item);
    }
  }

  state.rawAccessList = orderBy(
    [...memberMap.values()],
    [(access) => (access.type === "user" ? 1 : 0), (access) => access.member],
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
          if (item.member.startsWith(groupBindingPrefix)) {
            const group = groupStore.getGroupByIdentifier(item.member);
            if (group) {
              return <GroupNameCell group={group} />;
            }
          } else {
            const email = extractUserId(item.member);
            return <UserLink title={email} email={email} />;
          }
          return <span>{item.member}</span>;
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

const revokeAccessAlert = (item: AccessUser) => {
  $dialog.warning({
    title: t("common.warning"),
    content: () => {
      return (
        <div class="flex flex-col gap-y-3">
          <div class="textlabel text-base!">
            {t("project.masking-exemption.revoke-exemption-title", {
              member: item.member,
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
    onPositiveClick: async () => {
      await onRemove(item);
    },
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
  callback: (item: AccessUser) => void
) => {
  callback(item);
  await onSubmit();
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
    policyType: PolicyType.MASKING_EXEMPTION,
  });
  if (!policy) {
    return;
  }

  const expressionsMap = new Map<
    string,
    {
      description: string;
      members: string[];
    }
  >();

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
    const finalExpression = expressions.join(" && ");
    if (!expressionsMap.has(finalExpression)) {
      expressionsMap.set(finalExpression, {
        description: accessUser.description,
        members: [],
      });
    }
    expressionsMap.get(finalExpression)!.members.push(accessUser.member);
  }

  const exceptions = [];
  for (const [expression, { description, members }] of expressionsMap) {
    exceptions.push({
      members,
      condition: create(ExprSchema, {
        description: description,
        expression: expression,
      }),
    });
  }

  policy.policy = {
    case: "maskingExemptionPolicy",
    value: create(MaskingExemptionPolicySchema, {
      exemptions: exceptions,
    }),
  };
  await policyStore.upsertPolicy({
    parentPath: props.project.name,
    policy,
  });
};
</script>
