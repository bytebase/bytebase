<template>
  <FormLayout :title="title">
    <template #body>
      <div class="flex flex-col gap-y-8">
        <div class="w-full">
          <div class="flex items-center gap-x-1 mb-2">
            <span class="text-main">
              {{ $t("common.resources") }}
            </span>
            <RequiredStar />
          </div>
          <DatabaseResourceForm
            v-model:database-resources="state.databaseResources"
            :project-name="projectName"
            :required-feature="PlanFeature.FEATURE_DATA_MASKING"
            :include-cloumn="true"
          />
        </div>

        <div class="w-full">
          <p class="mb-2 text-main">
            {{ $t("common.reason") }}
          </p>
          <NInput
            v-model:value="state.description"
            :placeholder="$t('common.description')"
          />
        </div>

        <div class="w-full">
          <p class="mb-2 text-main">
            {{ $t("common.expiration") }}
          </p>
          <NDatePicker
            v-model:value="state.expirationTimestamp"
            style="width: 100%"
            type="datetime"
            :actions="null"
            :update-value-on-close="true"
            :is-date-disabled="(date: number) => date < dayjs().startOf('day').valueOf()"
            clearable
          />
          <span v-if="!state.expirationTimestamp" class="textinfolabel">{{
            $t("settings.sensitive-data.never-expires")
          }}</span>
        </div>

        <MembersBindingSelect
          v-model:value="state.memberList"
          :required="true"
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
import { create } from "@bufbuild/protobuf";
import { isUndefined } from "lodash-es";
import { NButton, NDatePicker, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseResourceForm from "@/components/GrantRequestPanel/DatabaseResourceForm/index.vue";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import { pushNotification, usePolicyV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  MaskingExceptionPolicy_MaskingException,
  Policy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExceptionPolicy_MaskingExceptionSchema,
  MaskingExceptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SensitiveColumn } from "./types";
import {
  convertSensitiveColumnToDatabaseResource,
  getExpressionsForDatabaseResource,
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
  processing: boolean;
  databaseResources?: DatabaseResource[];
  description: string;
}

const state = reactive<LocalState>({
  memberList: [],
  processing: false,
  databaseResources: props.columnList.map(
    convertSensitiveColumnToDatabaseResource
  ),
  description: "",
});

const policyStore = usePolicyV1Store();
const { t } = useI18n();

const resetState = () => {
  state.expirationTimestamp = undefined;
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

  for (const member of state.memberList) {
    const resourceExpressions = state.databaseResources?.map(
      getExpressionsForDatabaseResource
    ) ?? [[""]];
    for (const expressionList of resourceExpressions) {
      const resourceExpression = [...expressionList, ...expressions].filter(
        (e) => e
      );
      maskingExceptions.push(
        create(MaskingExceptionPolicy_MaskingExceptionSchema, {
          member,
          condition: create(ExprSchema, {
            description: state.description,
            expression:
              resourceExpression.length > 0
                ? resourceExpression.join(" && ")
                : "",
          }),
        })
      );
    }
  }

  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  const existed =
    policy?.policy?.case === "maskingExceptionPolicy"
      ? policy.policy.value.maskingExceptions
      : [];
  return {
    name: policy?.name,
    type: PolicyType.MASKING_EXCEPTION,
    resourceType: PolicyResourceType.PROJECT,
    policy: {
      case: "maskingExceptionPolicy",
      value: create(MaskingExceptionPolicySchema, {
        maskingExceptions: [...existed, ...maskingExceptions],
      }),
    },
  };
};
</script>
