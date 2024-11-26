<template>
  <FormLayout :title="title">
    <template #body>
      <div class="space-y-8">
        <div class="w-full">
          <div class="flex items-center gap-x-1 mb-2">
            <span class="font-medium text-main">
              {{ $t("common.resources") }}
            </span>
            <span class="text-red-600">*</span>
          </div>
          <DatabaseResourceForm
            v-model:database-resources="state.databaseResources"
            :project-name="projectName"
            :required-feature="'bb.feature.sensitive-data'"
            :include-cloumn="true"
          />
        </div>

        <div class="w-full">
          <div class="flex items-center gap-x-1 mb-2">
            <span class="font-medium text-main">
              {{ $t("settings.sensitive-data.action.self") }}
            </span>
            <span class="text-red-600">*</span>
          </div>
          <div class="flex space-x-4">
            <NCheckbox
              v-for="action in ACTIONS"
              :key="action"
              :checked="state.supportActions.has(action)"
              @update:checked="toggleAction(action, $event)"
            >
              {{
                $t(
                  `settings.sensitive-data.action.${maskingExceptionPolicy_MaskingException_ActionToJSON(
                    action
                  ).toLowerCase()}`
                )
              }}
            </NCheckbox>
          </div>
        </div>

        <div class="w-full">
          <div class="flex items-center gap-x-1 mb-2">
            <span class="font-medium text-main">
              {{ $t("settings.sensitive-data.masking-level.self") }}
            </span>
            <span class="text-red-600">*</span>
          </div>
          <MaskingLevelRadioGroup
            :level="state.maskingLevel"
            :level-list="MASKING_LEVELS"
            @update:level="state.maskingLevel = $event"
          />
        </div>

        <div class="w-full">
          <p class="mb-2 font-medium text-main">
            {{ $t("common.expiration") }}
          </p>
          <NDatePicker
            v-model:value="state.expirationTimestamp"
            style="width: 100%"
            type="datetime"
            :actions="null"
            :update-value-on-close="true"
            :is-date-disabled="(date: number) => date < Date.now()"
            clearable
          />
          <span v-if="!state.expirationTimestamp" class="textinfolabel">{{
            $t("settings.sensitive-data.never-expires")
          }}</span>
        </div>

        <MembersBindingSelect
          v-model:value="state.memberList"
          :required="true"
          :project-name="projectName"
          :include-all-users="false"
          :include-service-account="false"
        />
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end items-center">
        <div class="flex items-center gap-x-3">
          <NButton @click.prevent="onDismiss">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            :disabled="submitDisabled || state.processing"
            type="primary"
            @click.prevent="onSubmit"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </div>
    </template>
  </FormLayout>
</template>

<script lang="tsx" setup>
import { isUndefined } from "lodash-es";
import { NButton, NCheckbox, NDatePicker } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseResourceForm from "@/components/GrantRequestPanel/DatabaseResourceForm/index.vue";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import { usePolicyV1Store, pushNotification } from "@/store";
import type { DatabaseResource } from "@/types";
import { Expr } from "@/types/proto/google/type/expr";
import { MaskingLevel } from "@/types/proto/v1/common";
import type {
  Policy,
  MaskingExceptionPolicy_MaskingException,
} from "@/types/proto/v1/org_policy_service";
import {
  PolicyType,
  PolicyResourceType,
  MaskingExceptionPolicy_MaskingException_Action,
  maskingExceptionPolicy_MaskingException_ActionToJSON,
} from "@/types/proto/v1/org_policy_service";
import MaskingLevelRadioGroup from "./components/MaskingLevelRadioGroup.vue";
import type { SensitiveColumn } from "./types";
import {
  getExpressionsForDatabaseResource,
  convertSensitiveColumnToDatabaseResource,
} from "./utils";

const props = defineProps<{
  columnList: SensitiveColumn[];
  projectName: string;
  title?: string;
}>();

const emit = defineEmits(["dismiss"]);

interface LocalState {
  memberList: string[];
  expirationTimestamp?: number;
  maskingLevel: MaskingLevel;
  processing: boolean;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  databaseResources?: DatabaseResource[];
}

const ACTIONS = [
  MaskingExceptionPolicy_MaskingException_Action.EXPORT,
  MaskingExceptionPolicy_MaskingException_Action.QUERY,
];
const MASKING_LEVELS = [MaskingLevel.PARTIAL, MaskingLevel.NONE];

const state = reactive<LocalState>({
  memberList: [],
  maskingLevel: MaskingLevel.PARTIAL,
  processing: false,
  supportActions: new Set(ACTIONS),
  databaseResources: props.columnList.map(
    convertSensitiveColumnToDatabaseResource
  ),
});

const policyStore = usePolicyV1Store();
const { t } = useI18n();

const resetState = () => {
  state.expirationTimestamp = undefined;
  state.maskingLevel = MaskingLevel.PARTIAL;
  state.supportActions = new Set(ACTIONS);
  state.memberList = [];
  state.databaseResources = undefined;
  state.processing = false;
};

const onDismiss = () => {
  emit("dismiss");
  resetState();
};

const submitDisabled = computed(() => {
  if (state.memberList.length === 0) {
    return true;
  }
  if (state.supportActions.size === 0) {
    return true;
  }
  if (
    !isUndefined(state.databaseResources) &&
    state.databaseResources?.length === 0
  ) {
    return true;
  }
  return false;
});

const onSubmit = async () => {
  state.processing = true;

  try {
    const pendingUpdate = await getPendingUpdatePolicy(props.projectName);
    await policyStore.upsertPolicy({
      parentPath: props.projectName,
      policy: pendingUpdate,
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    onDismiss();
  } finally {
    state.processing = false;
  }
};

const getPendingUpdatePolicy = async (
  parentPath: string
): Promise<Partial<Policy>> => {
  const maskingExceptions: MaskingExceptionPolicy_MaskingException[] = [];

  const expressions = [];
  if (state.expirationTimestamp) {
    expressions.push(
      `request.time < timestamp("${new Date(
        state.expirationTimestamp
      ).toISOString()}")`
    );
  }

  for (const action of state.supportActions.values()) {
    for (const member of state.memberList) {
      if (!state.databaseResources) {
        maskingExceptions.push({
          member,
          action,
          maskingLevel: state.maskingLevel,
          condition: Expr.fromPartial({
            expression:
              expressions.length > 0 ? expressions.join(" && ") : undefined,
          }),
        });
      } else {
        for (const databaseResource of state.databaseResources) {
          const resourceExpressions =
            getExpressionsForDatabaseResource(databaseResource);
          resourceExpressions.push(...expressions);
          maskingExceptions.push({
            member,
            action,
            maskingLevel: state.maskingLevel,
            condition: Expr.fromPartial({
              expression: resourceExpressions.join(" && "),
            }),
          });
        }
      }
    }
  }

  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  const existed = policy?.maskingExceptionPolicy?.maskingExceptions ?? [];
  return {
    name: policy?.name,
    type: PolicyType.MASKING_EXCEPTION,
    resourceType: PolicyResourceType.PROJECT,
    maskingExceptionPolicy: {
      maskingExceptions: [...existed, ...maskingExceptions],
    },
  };
};

const toggleAction = (
  action: MaskingExceptionPolicy_MaskingException_Action,
  check: boolean
) => {
  if (check) {
    state.supportActions.add(action);
  } else {
    state.supportActions.delete(action);
  }
};
</script>
