<template>
  <NDataTable
    :size="size"
    :columns="accessTableColumns"
    :data="filteredList"
    :row-key="(row: AccessUser) => row.key"
    :bordered="false"
    :striped="true"
    :loading="!ready || state.loading"
    :max-height="'calc(100vh - 15rem)'"
    virtual-scroll
  />
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import { TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NDatePicker, NEllipsis, useDialog } from "naive-ui";
import { computed, h, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { MiniActionButton } from "@/components/v2";
import { UserLink } from "@/components/v2/Model/cells";
import {
  composePolicyBindings,
  extractUserEmail,
  pushNotification,
  useGroupStore,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import { groupBindingPrefix } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
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
const policyStore = usePolicyV1Store();
const $dialog = useDialog();

const hasGetDatabasePermission = computed(() =>
  hasProjectPermissionV2(props.project, "bb.databases.get")
);

const { policy, ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project.name,
    policyType: PolicyType.MASKING_EXEMPTION,
  }))
);

const filteredList = computed(() =>
  state.rawAccessList.filter(props.filterAccessUser)
);

const expirationTimeRegex = /request\.time\s*<\s*timestamp\("(.+)?"\)/;

// Extract the condition portion of a CEL expression, excluding request.time.
const getConditionExpression = (expression: string): string => {
  if (!expression) return "";
  return expression
    .split(" && ")
    .filter((part) => !part.match(expirationTimeRegex))
    .join(" && ");
};

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

  const conditionExpression = getConditionExpression(expression);
  const databaseResources =
    condition.databaseResources && condition.databaseResources.length > 0
      ? condition.databaseResources
      : undefined;

  const result: AccessUser[] = [];
  for (const member of exception.members) {
    result.push({
      type: member.startsWith(groupBindingPrefix) ? "group" : "user",
      member,
      key: `${member}:${expression}.${description}`,
      expirationTimestamp,
      rawExpression: expression,
      description,
      conditionExpression,
      databaseResources,
    });
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
    e.condition?.expression ? e.condition.expression : "true"
  );
  const conditionList = await batchConvertFromCELString(expressionList);

  await composePolicyBindings(exemptions, true);
  for (let i = 0; i < exemptions.length; i++) {
    const items = getAccessUsers(exemptions[i], conditionList[i]);
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
        key: "member",
        title: t("common.members"),
        width: 180,
        render: (item: AccessUser) => {
          if (item.member.startsWith(groupBindingPrefix)) {
            const group = groupStore.getGroupByIdentifier(item.member);
            if (group) {
              return <GroupNameCell group={group} />;
            }
          } else {
            const email = extractUserEmail(item.member);
            return <UserLink title={email} email={email} />;
          }
          return <span>{item.member}</span>;
        },
      },
      {
        key: "condition",
        title: t("cel.condition.self"),
        render: (item: AccessUser) => {
          if (!item.conditionExpression) {
            return <span class="textinfo">{t("database.all")}</span>;
          }
          // Best-effort: render database resources nicely if parseable.
          if (item.databaseResources && item.databaseResources.length > 0) {
            return (
              <div class="flex flex-col gap-y-1">
                {item.databaseResources.map((res) => (
                  <div class="flex flex-col gap-y-0.5 text-sm textinfo">
                    <div class="flex items-center gap-x-1">
                      <span class="font-medium">{t("common.database")}:</span>
                      {hasGetDatabasePermission.value ? (
                        <span
                          class="normal-link hover:underline cursor-pointer"
                          onClick={() => {
                            const query: Record<string, string> = {};
                            if (res.schema) query.schema = res.schema;
                            if (res.table) query.table = res.table;
                            router.push({ path: res.databaseFullName, query });
                          }}
                        >
                          {res.databaseFullName}
                        </span>
                      ) : (
                        <span>{res.databaseFullName}</span>
                      )}
                    </div>
                    {res.schema && (
                      <div class="flex items-center gap-x-1">
                        <span class="font-medium">{t("common.schema")}:</span>
                        <span>{res.schema}</span>
                      </div>
                    )}
                    {res.table && (
                      <div class="flex items-center gap-x-1">
                        <span class="font-medium">{t("common.table")}:</span>
                        <span>{res.table}</span>
                      </div>
                    )}
                    {res.columns && res.columns.length > 0 && (
                      <div class="flex items-center gap-x-1">
                        <span class="font-medium">
                          {t("database.columns")}:
                        </span>
                        <span>{res.columns.join(", ")}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            );
          }
          // Fallback: show raw CEL expression.
          return (
            <NEllipsis class="text-sm font-mono">
              {item.conditionExpression}
            </NEllipsis>
          );
        },
      },
      {
        key: "expire",
        title: t("common.expiration"),
        width: 220,
        render: (item: AccessUser) => {
          return (
            <NDatePicker
              value={item.expirationTimestamp}
              style={"width: 100%"}
              type={"datetime"}
              actions={["confirm"]}
              isDateDisabled={(date: number) => date < Date.now()}
              clearable={true}
              disabled={props.disabled}
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
        width: 120,
        render: (item: AccessUser) => {
          return item.description;
        },
      },
      {
        key: "operation",
        title: "",
        hide: props.disabled,
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
          {item.conditionExpression && (
            <div class="text-sm textinfo">
              {item.databaseResources && item.databaseResources.length > 0
                ? item.databaseResources
                    .map((r) =>
                      [
                        r.databaseFullName,
                        r.schema,
                        r.table,
                        r.columns?.join(", "),
                      ]
                        .filter(Boolean)
                        .join(" / ")
                    )
                    .join("; ")
                : item.conditionExpression}
            </div>
          )}
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
