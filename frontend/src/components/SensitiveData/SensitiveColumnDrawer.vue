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
        <div class="space-y-6">
          <div class="w-full">
            <h1 class="mb-2 font-semibold">
              {{ $t("settings.sensitive-data.masking-level.self") }}
            </h1>
            <MaskingLevelRadioGroup
              :level="state.maskingLevel"
              :level-list="MASKING_LEVELS"
              :disabled="!hasPermission || state.processing"
              :effective-masking-level="columnMetadata?.effectiveMaskingLevel"
              @update:level="onMaskingLevelUpdate($event)"
            />
          </div>
          <div class="w-full">
            <h1 class="mb-2 font-semibold">
              {{ $t("settings.sensitive-data.algorithms.self") }}
            </h1>
            <div class="flex flex-col space-y-2">
              <span class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.semantic-types.table.full-masking-algorithm"
                  )
                }}
              </span>
              <NSelect
                v-model:value="state.fullMaskingAlgorithmId"
                :options="algorithmList"
                :consistent-menu-width="false"
                :placeholder="$t('settings.sensitive-data.algorithms.default')"
                :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
                clearable
                size="small"
                style="min-width: 7rem; max-width: 20rem; overflow-x: hidden"
                @change="onMaskingAlgorithmChanged"
              />
            </div>
            <div class="flex flex-col mt-4 space-y-2">
              <span class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.semantic-types.table.partial-masking-algorithm"
                  )
                }}
              </span>
              <NSelect
                v-model:value="state.partialMaskingAlgorithmId"
                :options="algorithmList"
                :consistent-menu-width="false"
                :placeholder="$t('settings.sensitive-data.algorithms.default')"
                :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
                clearable
                size="small"
                style="min-width: 7rem; max-width: 20rem; overflow-x: hidden"
                @change="onMaskingAlgorithmChanged"
              />
            </div>
          </div>
        </div>
        <div class="pt-8 space-y-5">
          <div class="flex justify-between">
            <div>
              <h1 class="font-semibold">
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
          <BBGrid
            :column-list="gridColumnList"
            :data-source="accessUserList"
            :row-clickable="false"
            class="border compact"
          >
            <template #item="{ item, row }: AccessUserRow">
              <div class="bb-grid-cell gap-x-2">
                <UserAvatar size="SMALL" :user="item.user" />
                <div class="flex flex-col">
                  <router-link
                    :to="`/users/${item.user.email}`"
                    class="normal-link"
                  >
                    {{ item.user.title }}
                  </router-link>
                  <span class="textinfolabel">
                    {{ item.user.email }}
                  </span>
                </div>
              </div>
              <div class="bb-grid-cell">
                <NCheckbox
                  :checked="
                    item.supportActions.has(
                      MaskingExceptionPolicy_MaskingException_Action.EXPORT
                    )
                  "
                  :disabled="!hasPermission || state.processing"
                  @update-checked="
                    onAccessControlUpdate(row, (item) =>
                      toggleAction(
                        item,
                        MaskingExceptionPolicy_MaskingException_Action.EXPORT,
                        $event
                      )
                    )
                  "
                />
              </div>
              <div class="bb-grid-cell">
                <NCheckbox
                  :checked="
                    item.supportActions.has(
                      MaskingExceptionPolicy_MaskingException_Action.QUERY
                    )
                  "
                  :disabled="!hasPermission || state.processing"
                  @update:checked="
                    onAccessControlUpdate(row, (item) =>
                      toggleAction(
                        item,
                        MaskingExceptionPolicy_MaskingException_Action.QUERY,
                        $event
                      )
                    )
                  "
                />
              </div>
              <div class="bb-grid-cell">
                <MaskingLevelDropdown
                  :disabled="!hasPermission || state.processing"
                  :level="item.maskingLevel"
                  :level-list="[MaskingLevel.PARTIAL, MaskingLevel.NONE]"
                  @update:level="
                    onAccessControlUpdate(
                      row,
                      (item) => (item.maskingLevel = $event)
                    )
                  "
                />
              </div>
              <div class="bb-grid-cell">
                <NDatePicker
                  :value="item.expirationTimestamp"
                  style="width: 100%"
                  type="datetime"
                  :is-date-disabled="(date: number) => date < Date.now()"
                  clearable
                  :disabled="!hasPermission || state.processing"
                  @update:value="(val: number | undefined) => onAccessControlUpdate(row, (item) => item.expirationTimestamp = val)"
                />
              </div>
              <div v-if="hasPermission" class="bb-grid-cell">
                <NPopconfirm @positive-click="onRemove(row)">
                  <template #trigger>
                    <MiniActionButton
                      tag="div"
                      :disabled="!hasPermission || state.processing"
                      @click.stop=""
                    >
                      <TrashIcon class="w-4 h-4" />
                    </MiniActionButton>
                  </template>

                  <div class="whitespace-nowrap">
                    {{
                      $t(
                        "settings.sensitive-data.column-detail.remove-user-permission"
                      )
                    }}
                  </div>
                </NPopconfirm>
              </div>
            </template>
          </BBGrid>
        </div>
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
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
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { TrashIcon } from "lucide-vue-next";
import {
  NSelect,
  NButton,
  NCheckbox,
  NDatePicker,
  NPopconfirm,
  SelectOption,
} from "naive-ui";
import { computed, reactive, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid } from "@/bbkit";
import type { BBGridColumn, BBGridRow } from "@/bbkit/types";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  useSettingV1Store,
  usePolicyV1Store,
  usePolicyByParentAndType,
  useUserStore,
  pushNotification,
  useCurrentUserV1,
  useDBSchemaV1Store,
} from "@/store";
import { getUserId } from "@/store/modules/v1/common";
import { unknownUser } from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import { User } from "@/types/proto/v1/auth_service";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  MaskData,
  PolicyResourceType,
  MaskingExceptionPolicy_MaskingException,
  MaskingExceptionPolicy_MaskingException_Action,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import UserAvatar from "../User/UserAvatar.vue";
import GrantAccessDrawer from "./GrantAccessDrawer.vue";
import MaskingLevelDropdown from "./components/MaskingLevelDropdown.vue";
import MaskingLevelRadioGroup from "./components/MaskingLevelRadioGroup.vue";
import { SensitiveColumn } from "./types";
import { getMaskDataIdentifier, isCurrentColumnException } from "./utils";

interface AccessUser {
  user: User;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  maskingLevel: MaskingLevel;
  expirationTimestamp?: number;
  rawExpression: string;
}

type AccessUserRow = BBGridRow<AccessUser>;

interface LocalState {
  dirty: boolean;
  processing: boolean;
  maskingLevel: MaskingLevel;
  showGrantAccessDrawer: boolean;
  fullMaskingAlgorithmId: string;
  partialMaskingAlgorithmId: string;
}

const props = defineProps<{
  show: boolean;
  column: SensitiveColumn;
}>();

defineEmits(["dismiss"]);

const state = reactive<LocalState>({
  dirty: false,
  processing: false,
  maskingLevel: props.column.maskData.maskingLevel,
  fullMaskingAlgorithmId: props.column.maskData.fullMaskingAlgorithmId,
  partialMaskingAlgorithmId: props.column.maskData.partialMaskingAlgorithmId,
  showGrantAccessDrawer: false,
});

const MASKING_LEVELS = [
  MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
  MaskingLevel.FULL,
  MaskingLevel.PARTIAL,
  MaskingLevel.NONE,
];

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();
const accessUserList = ref<AccessUser[]>([]);
const policyStore = usePolicyV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const settingStore = useSettingV1Store();

const policy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.column.database.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  }))
);

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
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
    `level:"${maskingLevelToJSON(exception.maskingLevel)}"`,
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
  const userMap = new Map<string, AccessUser>();
  for (const exception of policy.maskingExceptionPolicy.maskingExceptions) {
    if (!isCurrentColumnException(exception, props.column)) {
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
      state.fullMaskingAlgorithmId =
        props.column.maskData.fullMaskingAlgorithmId;
      state.partialMaskingAlgorithmId =
        props.column.maskData.partialMaskingAlgorithmId;
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

const gridColumnList = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("common.user"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.action.export"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.action.query"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.masking-level.self"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.expiration"),
      width: "minmax(min-content, auto)",
    },
  ];
  if (hasPermission.value) {
    // operation.
    columns.push({
      title: "",
      width: "minmax(min-content, auto)",
    });
  }
  return columns;
});

const onRemove = async (index: number) => {
  accessUserList.value.splice(index, 1);
  state.dirty = true;
  await onSubmit();
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

const onAccessControlUpdate = async (
  index: number,
  callback: (item: AccessUser) => void
) => {
  const item = accessUserList.value[index];
  if (!item) {
    return;
  }
  callback(item);
  state.dirty = true;
  await onSubmit();
};

const onMaskingLevelUpdate = async (level: MaskingLevel) => {
  state.maskingLevel = level;
  await onColumnMaskingUpdate();

  dbSchemaStore.getOrFetchTableMetadata({
    database: props.column.database.name,
    schema: props.column.maskData.schema,
    table: props.column.maskData.table,
    skipCache: true,
    silent: false,
  });
};

const onMaskingAlgorithmChanged = async () => {
  await onColumnMaskingUpdate();
};

const onColumnMaskingUpdate = async () => {
  state.processing = true;

  try {
    await upsertMaskingPolicy();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.processing = false;
  }
};

const onSubmit = async () => {
  state.processing = true;

  try {
    if (state.dirty) {
      await updateExceptionPolicy();
      state.dirty = false;
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.processing = false;
  }
};

const upsertMaskingPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.column.database.name,
    policyType: PolicyType.MASKING,
  });

  const maskData = policy?.maskingPolicy?.maskData ?? [];
  const existedIndex = maskData.findIndex(
    (data) =>
      getMaskDataIdentifier(data) ===
      getMaskDataIdentifier(props.column.maskData)
  );
  const newData: MaskData = {
    ...props.column.maskData,
    maskingLevel: state.maskingLevel,
    fullMaskingAlgorithmId: state.fullMaskingAlgorithmId,
    partialMaskingAlgorithmId: state.partialMaskingAlgorithmId,
  };
  if (existedIndex < 0) {
    maskData.push(newData);
  } else {
    maskData[existedIndex] = newData;
  }

  const upsert: Partial<Policy> = {
    type: PolicyType.MASKING,
    resourceType: PolicyResourceType.DATABASE,
    resourceUid: props.column.database.uid,
    maskingPolicy: {
      maskData,
    },
  };

  await policyStore.upsertPolicy({
    parentPath: props.column.database.name,
    policy: upsert,
    updateMask: ["payload"],
  });
};

const updateExceptionPolicy = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: props.column.database.project,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter((exception) => !isCurrentColumnException(exception, props.column));

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

const columnMetadata = computedAsync(async () => {
  const { column } = props;
  if (column.maskData.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
    return undefined;
  }
  const table = await dbSchemaStore.getOrFetchTableMetadata({
    database: column.database.name,
    schema: column.maskData.schema,
    table: column.maskData.table,
  });
  return table?.columns.find((c) => c.name === column.maskData.column);
}, undefined);

const algorithmList = computed((): SelectOption[] => {
  const list = (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));

  list.unshift({
    label: t("settings.sensitive-data.algorithms.default"),
    value: "",
  });

  return list;
});
</script>
