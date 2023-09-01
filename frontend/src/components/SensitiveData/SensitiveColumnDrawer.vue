<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="
        $t('settings.sensitive-data.column-detail.masking-setting-for-column', {
          column: column.maskData.column,
        })
      "
    >
      <div class="divide-block-border divide-y space-y-8 w-[50rem] h-full">
        <div>
          <div class="w-full">
            <p class="mb-2">
              {{ $t("settings.sensitive-data.masking-level.self") }}
            </p>
            <MaskingLevelRadioGroup
              :disabled="hasPermission"
              :level-list="MASKING_LEVELS"
              :selected="column.maskData.maskingLevel"
              @update="column.maskData.maskingLevel = $event"
            />
          </div>
        </div>
        <div class="pt-8 space-y-5">
          <div class="flex justify-between">
            <div>
              <h1>
                {{
                  $t("settings.sensitive-data.column-detail.access-user-list")
                }}
              </h1>
              <span class="textinfolabel">{{
                $t(
                  "settings.sensitive-data.column-detail.access-user-list-desc"
                )
              }}</span>
            </div>
            <NButton type="primary" :disabled="!hasPermission" @click="">
              {{ $t("settings.sensitive-data.grant-access") }}
            </NButton>
          </div>
          <BBTable
            ref="tableRef"
            :column-list="tableHeaderList"
            :data-source="accessUserList"
            :show-header="true"
            :left-bordered="true"
            :right-bordered="true"
            :top-bordered="true"
            :bottom-bordered="true"
            :compact-section="true"
            :row-clickable="false"
            @click-row=""
          >
            <template
              #body="{
                rowData: item,
                row,
              }: {
                rowData: AccessUser,
                row: number,
              }"
            >
              <BBTableCell class="bb-grid-cell">
                <div class="flex items-center space-x-2">
                  <UserAvatar size="SMALL" :user="item.user" />
                  <div class="flex flex-col">
                    <router-link
                      :to="`/u/${extractUserUID(item.user.name)}`"
                      class="normal-link"
                    >
                      {{ item.user.title }}
                    </router-link>
                    <span class="textinfolabel">
                      {{ item.user.email }}
                    </span>
                  </div>
                </div>
              </BBTableCell>
              <BBTableCell class="bb-grid-cell">
                <BBCheckbox
                  :value="
                    item.supportActions.has(
                      MaskingExceptionPolicy_MaskingException_Action.EXPORT
                    )
                  "
                  @toggle="(checked: boolean) => onUpdate(row, (item) => toggleAction(item, MaskingExceptionPolicy_MaskingException_Action.EXPORT, checked))"
                />
              </BBTableCell>
              <BBTableCell class="bb-grid-cell">
                <BBCheckbox
                  :value="
                    item.supportActions.has(
                      MaskingExceptionPolicy_MaskingException_Action.QUERY
                    )
                  "
                  @toggle="(checked: boolean) => onUpdate(row, (item) => toggleAction(item, MaskingExceptionPolicy_MaskingException_Action.QUERY, checked))"
                />
              </BBTableCell>
              <BBTableCell class="bb-grid-cell">
                <MaskingLevelDropdown
                  :disabled="!hasPermission"
                  :selected="item.maskingLevel"
                  :level-list="[MaskingLevel.PARTIAL, MaskingLevel.NONE]"
                  @update="(level: MaskingLevel) => onUpdate(row, (item) => item.maskingLevel = level)"
                />
              </BBTableCell>
              <BBTableCell class="bb-grid-cell">
                <NDatePicker
                  :value="item.expirationTimestamp"
                  style="width: 100%"
                  type="datetime"
                  :is-date-disabled="(date: number) => date < Date.now()"
                  clearable
                  @update:value="(val: number | undefined) => onUpdate(row, (item) => item.expirationTimestamp = val)"
                />
              </BBTableCell>
              <BBTableCell v-if="hasPermission" class="bb-grid-cell">
                <NPopconfirm @positive-click="onRemove(row)">
                  <template #trigger>
                    <button
                      class="w-5 h-5 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                      @click.stop=""
                    >
                      <heroicons-outline:trash />
                    </button>
                  </template>

                  <div class="whitespace-nowrap">
                    {{
                      $t(
                        "settings.sensitive-data.column-detail.remove-user-permission"
                      )
                    }}
                  </div>
                </NPopconfirm>
              </BBTableCell>
            </template>
          </BBTable>
        </div>
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              :disabled="!hasPermission || !state.dirty || state.processing"
              type="primary"
              @click.prevent="onSubmit"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { groupBy } from "lodash-es";
import { NButton, NDatePicker, NPopconfirm } from "naive-ui";
import { computed, reactive, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn, BBTableSectionDataSource } from "@/bbkit/types";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  resolveCELExpr,
  wrapAsGroup,
  buildCELExpr,
  validateSimpleExpr,
} from "@/plugins/cel";
import {
  usePolicyV1Store,
  usePolicyByParentAndType,
  useUserStore,
  pushNotification,
  useCurrentUserV1,
} from "@/store";
import { getUserId } from "@/store/modules/v1/common";
import { unknownUser } from "@/types";
import {
  Expr as CELExpr,
  ParsedExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import {
  LoginRequest,
  LoginResponse,
  User,
  UserType,
} from "@/types/proto/v1/auth_service";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingExceptionPolicy_MaskingException,
  MaskingExceptionPolicy_MaskingException_Action,
  maskingExceptionPolicy_MaskingException_ActionToJSON,
} from "@/types/proto/v1/org_policy_service";
import { extractInstanceResourceName } from "@/utils";
import { hasWorkspacePermissionV1, extractUserUID } from "@/utils";
import {
  convertCELStringToParsedExpr,
  convertParsedExprToCELString,
} from "@/utils";
import {
  convertFromCELString,
  convertFromExpr,
  stringifyDatabaseResources,
} from "@/utils/issue/cel";
import { SensitiveColumn } from "./types";

interface AccessUser {
  user: User;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  maskingLevel: MaskingLevel;
  expirationTimestamp?: number;
}

interface LocalState {
  dirty: boolean;
  processing: boolean;
}

const props = defineProps<{
  show: boolean;
  column: SensitiveColumn;
}>();

const emit = defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  dirty: false,
  processing: false,
});

const MASKING_LEVELS = [
  MaskingLevel.FULL,
  MaskingLevel.PARTIAL,
  MaskingLevel.NONE,
];

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();
const accessUserList = ref<AccessUser[]>([]);
const policyStore = usePolicyV1Store();

const policy = usePolicyByParentAndType({
  parentPath: props.column.database.name,
  policyType: PolicyType.MASKING_EXCEPTION,
});

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const getAccessUsers = async (
  exception: MaskingExceptionPolicy_MaskingException
): Promise<AccessUser[]> => {
  const parsedExpr = await convertCELStringToParsedExpr(
    exception.condition?.expression ?? ""
  );
  let expirationTimestamp: number | undefined;
  if (parsedExpr.expr) {
    const conditionExpr = convertFromExpr(parsedExpr.expr);
    if (conditionExpr.expiredTime) {
      expirationTimestamp = new Date(conditionExpr.expiredTime).getTime();
    }
  }

  return exception.members.map((member) => {
    const user = userStore.getUserByIdentifier(member) ?? unknownUser();
    return {
      user,
      maskingLevel: exception.maskingLevel,
      expirationTimestamp,
      supportActions: new Set([exception.action]),
    };
  });
};

const isCurrentColumnException = (
  exception: MaskingExceptionPolicy_MaskingException
): boolean => {
  const expression = exception.condition?.expression ?? "";
  const matches = [
    `resource.table_name == "${props.column.maskData.table}"`,
    `resource.column_name == "${props.column.maskData.column}"`,
  ];
  if (props.column.maskData.schema) {
    matches.push(`resource.schema_name == "${props.column.maskData.schema}"`);
  }

  for (const match of matches) {
    if (!expression.includes(match)) {
      return false;
    }
  }
  return true;
};

const getExceptionIdentifier = (
  exception: MaskingExceptionPolicy_MaskingException
): string => {
  const res: string[] = [
    `level == "${maskingLevelToJSON(exception.maskingLevel)}"`,
  ];
  const expression = exception.condition?.expression ?? "";
  const matches = /request.time < timestamp\(".+?"\)/.exec(expression);
  if (matches) {
    res.push(matches[0]);
  }
  return res.join(" && ");
};

watch(
  () => policy.value,
  async (policy) => {
    if (!policy || !policy.maskingExceptionPolicy) {
      return [];
    }

    const userMap = new Map<string, AccessUser>();
    for (const exception of policy.maskingExceptionPolicy.maskingExceptions) {
      if (!isCurrentColumnException(exception)) {
        continue;
      }
      const identifier = getExceptionIdentifier(exception);
      const users = await getAccessUsers(exception);
      for (const item of users) {
        const id = `${item.user.name}:${identifier}`;
        const target = userMap.get(id) ?? item;
        if (userMap.has(id)) {
          for (const action of item.supportActions) {
            target.supportActions.add(action);
          }
        }
        userMap.set(id, target);
      }
    }

    accessUserList.value = [...userMap.values()].sort(
      (u1, u2) => getUserId(u1.user.name) - getUserId(u2.user.name)
    );
  },
  { immediate: true, deep: true }
);

const tableHeaderList = computed(() => {
  const list: BBTableColumn[] = [
    {
      title: t("common.user"),
    },
    {
      title: t("settings.sensitive-data.action.export"),
    },
    {
      title: t("settings.sensitive-data.action.query"),
    },
    {
      title: t("settings.sensitive-data.masking-level.self"),
    },
    {
      title: t("common.expiration"),
    },
  ];
  if (hasPermission.value) {
    // operation.
    list.push({
      title: "",
    });
  }
  return list;
});

const onRemove = (index: number) => {
  accessUserList.value.splice(index, 1);
  state.dirty = true;
};

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

const onUpdate = (index: number, callback: (item: AccessUser) => void) => {
  const item = accessUserList.value[index];
  if (!item) {
    return;
  }
  callback(item);
  state.dirty = true;
};

const onSubmit = async () => {
  state.processing = true;

  try {
    emit("dismiss");
  } finally {
    state.processing = false;
  }
};
</script>
