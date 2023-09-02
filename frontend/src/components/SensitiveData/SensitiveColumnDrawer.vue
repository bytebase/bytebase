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
              :disabled="!hasPermission"
              :level-list="MASKING_LEVELS"
              :selected="state.maskingLevel"
              @update="onLevelUpdate"
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
            <NButton
              type="primary"
              :disabled="!hasPermission"
              @click="state.showGrantAccessDrawer = true"
            >
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
              :disabled="
                !hasPermission ||
                (!state.dirty && !maskingPolicyChanged) ||
                state.processing
              "
              type="primary"
              @click.prevent="onSubmit"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>

    <GrantAccessDrawer
      :show="state.showGrantAccessDrawer"
      :column-list="[props.column]"
      @dismiss="state.showGrantAccessDrawer = false"
    />
  </Drawer>

  <BBAlert
    v-if="state.showAlert"
    :style="'WARN'"
    :ok-text="$t('common.ok')"
    :title="
      $t('settings.sensitive-data.column-detail.remove-sensitive-warning')
    "
    @ok="state.showAlert = false"
    @cancel="
      () => {
        state.showAlert = false;
        state.maskingLevel = props.column.maskData.maskingLevel;
      }
    "
  >
  </BBAlert>
</template>

<script lang="ts" setup>
import { NButton, NDatePicker, NPopconfirm } from "naive-ui";
import { computed, reactive, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn } from "@/bbkit/types";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  usePolicyV1Store,
  usePolicyByParentAndType,
  useUserStore,
  pushNotification,
  useCurrentUserV1,
} from "@/store";
import { getUserId } from "@/store/modules/v1/common";
import { unknownUser } from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import { User } from "@/types/proto/v1/auth_service";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  MaskingExceptionPolicy_MaskingException,
  MaskingExceptionPolicy_MaskingException_Action,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV1, extractUserUID } from "@/utils";
import { SensitiveColumn } from "./types";
import {
  getMaskDataIdentifier,
  isCurrentColumnException,
  removeSensitiveColumn,
} from "./utils";

interface AccessUser {
  user: User;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  maskingLevel: MaskingLevel;
  expirationTimestamp?: number;
  rawExpression: string;
}

interface LocalState {
  dirty: boolean;
  processing: boolean;
  maskingLevel: MaskingLevel;
  showGrantAccessDrawer: boolean;
  showAlert: boolean;
}

const props = defineProps<{
  show: boolean;
  column: SensitiveColumn;
}>();

const emit = defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  dirty: false,
  processing: false,
  maskingLevel: props.column.maskData.maskingLevel,
  showGrantAccessDrawer: false,
  showAlert: false,
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

const policy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.column.database.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  }))
);

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const expirationTimeRegex = /request.time < timestamp\("(.+)?"\)/;

const getAccessUsers = (
  exception: MaskingExceptionPolicy_MaskingException
): AccessUser => {
  let expirationTimestamp: number | undefined;
  const expression = exception.condition?.expression ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    expirationTimestamp = new Date(matches[1]).getTime();
  }

  const user = userStore.getUserByIdentifier(exception.member) ?? unknownUser();
  return {
    user,
    maskingLevel: exception.maskingLevel,
    expirationTimestamp,
    supportActions: new Set([exception.action]),
    rawExpression: exception.condition?.expression ?? "",
  };
};

const getExceptionIdentifier = (
  exception: MaskingExceptionPolicy_MaskingException
): string => {
  const res: string[] = [
    `level == "${maskingLevelToJSON(exception.maskingLevel)}"`,
  ];
  const expression = exception.condition?.expression ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    res.push(matches[0]);
  }
  return res.join(" && ");
};

const updateAccessUserList = (policy: Policy | undefined) => {
  if (!policy || !policy.maskingExceptionPolicy) {
    return [];
  }

  const userMap = new Map<string, AccessUser>();
  for (const exception of policy.maskingExceptionPolicy.maskingExceptions) {
    if (!isCurrentColumnException(exception, props.column.maskData)) {
      continue;
    }
    const identifier = getExceptionIdentifier(exception);
    const item = getAccessUsers(exception);
    const id = `${item.user.name}:${identifier}`;
    const target = userMap.get(id) ?? item;
    if (userMap.has(id)) {
      for (const action of item.supportActions) {
        target.supportActions.add(action);
      }
    }
    userMap.set(id, target);
  }

  accessUserList.value = [...userMap.values()].sort(
    (u1, u2) => getUserId(u1.user.name) - getUserId(u2.user.name)
  );
};

watch(
  () => [props.show, policy.value],
  () => {
    if (props.show) {
      state.maskingLevel = props.column.maskData.maskingLevel;
    }
    if (props.show && policy.value) {
      updateAccessUserList(policy.value);
    }
  },
  {
    immediate: true,
    deep: true,
  }
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

const onLevelUpdate = (level: MaskingLevel) => {
  state.maskingLevel = level;
  if (level === MaskingLevel.NONE) {
    state.showAlert = true;
  }
};

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

const maskingPolicyChanged = computed(() => {
  return state.maskingLevel !== props.column.maskData.maskingLevel;
});

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
    if (maskingPolicyChanged.value) {
      if (state.maskingLevel === MaskingLevel.NONE) {
        // remove masking level and exceptions
        await removeSensitiveColumn(props.column);
        state.dirty = false;
      } else {
        await updateMaskingPolicy();
      }
    }
    if (state.dirty) {
      await updateExceptionPolicy();
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("dismiss");
  } finally {
    state.processing = false;
  }
};

const updateMaskingPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.column.database.name,
    policyType: PolicyType.MASKING,
  });
  if (!policy) {
    return;
  }

  const maskData = policy.maskingPolicy?.maskData ?? [];
  const existedIndex = maskData.findIndex(
    (data) =>
      getMaskDataIdentifier(data) ===
      getMaskDataIdentifier(props.column.maskData)
  );
  if (existedIndex < 0) {
    maskData.push({
      ...props.column.maskData,
      maskingLevel: state.maskingLevel,
    });
  } else {
    maskData[existedIndex] = {
      ...props.column.maskData,
      maskingLevel: state.maskingLevel,
    };
  }
  policy.maskingPolicy = {
    ...(policy.maskingPolicy ?? {}),
    maskData,
  };

  await policyStore.updatePolicy(["payload"], policy);
};

const updateExceptionPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.column.database.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter(
    (exception) => !isCurrentColumnException(exception, props.column.maskData)
  );

  for (const accessUser of accessUserList.value) {
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
    }
    for (const action of accessUser.supportActions) {
      exceptions.push({
        maskingLevel: accessUser.maskingLevel,
        action,
        member: `user:${accessUser.user.email}`,
        condition: Expr.fromPartial({
          expression: expressions.join(" && "),
        }),
      });
    }
  }

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: exceptions,
  };
  await policyStore.updatePolicy(["payload"], policy);
};
</script>
