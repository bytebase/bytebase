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
            ref="databaseResourceFormRef"
            :database-resources="databaseResources"
            :project-name="projectName"
            :required-feature="PlanFeature.FEATURE_DATA_MASKING"
            :include-cloumn="true"
            :include-classification-level="true"
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
          :include-service-account="true"
          :include-workload-identity="true"
        />
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end items-center">
        <div class="flex items-center gap-x-2">
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
import { NButton, NDatePicker, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import DatabaseResourceForm from "@/components/RoleGrantPanel/DatabaseResourceForm/index.vue";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import { buildCELExpr } from "@/plugins/cel";
import { pushNotification, usePolicyV1Store, useSettingV1Store } from "@/store";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  MaskingExemptionPolicy_Exemption,
  Policy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils";
import { rewriteResourceDatabase } from "./exemptionDataUtils";
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

const settingStore = useSettingV1Store();
onMounted(async () => {
  await settingStore.getOrFetchSettingByName(
    Setting_SettingName.DATA_CLASSIFICATION,
    true
  );
});

interface LocalState {
  memberList: string[];
  expirationTimestamp?: number;
  processing: boolean;
  description: string;
}

const state = reactive<LocalState>({
  memberList: [],
  processing: false,
  description: "",
});
const databaseResourceFormRef =
  ref<InstanceType<typeof DatabaseResourceForm>>();

const databaseResources = computed(() => {
  return props.columnList.map(convertSensitiveColumnToDatabaseResource);
});

const policyStore = usePolicyV1Store();
const { t } = useI18n();

const resetState = () => {
  state.expirationTimestamp = undefined;
  state.memberList = [];
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
  return !databaseResourceFormRef.value?.isValid;
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
  const exemptions: MaskingExemptionPolicy_Exemption[] = [];

  const extraExpressions: string[] = [];
  if (state.expirationTimestamp) {
    extraExpressions.push(
      `request.time < timestamp("${new Date(
        state.expirationTimestamp
      ).toISOString()}")`
    );
  }

  // When using CEL expression mode, build the expression string directly
  // to preserve non-resource conditions like resource.classification_level.
  const expr = databaseResourceFormRef.value?.getExpr?.();
  if (expr) {
    const parsedExpr = await buildCELExpr(expr);
    if (parsedExpr) {
      let [celString] = await batchConvertParsedExprToCELString([parsedExpr]);
      // The expression editor uses `resource.database` as a convenience factor
      // (e.g., resource.database == "instances/X/databases/Y"), but the backend
      // masking exemption CEL evaluator only supports resource.instance_id and
      // resource.database_name. Rewrite to the backend-compatible form.
      celString = rewriteResourceDatabase(celString);
      const parts = [celString, ...extraExpressions].filter((e) => e);
      exemptions.push(
        create(MaskingExemptionPolicy_ExemptionSchema, {
          members: state.memberList,
          condition: create(ExprSchema, {
            description: state.description,
            expression: parts.length > 0 ? parts.join(" && ") : "",
          }),
        })
      );
    }
  } else {
    // ALL or SELECT mode: use database resource conversion
    const databaseResources =
      await databaseResourceFormRef.value?.getDatabaseResources();

    const resourceExpressions = (
      databaseResources?.map(getExpressionsForDatabaseResource) ?? [[""]]
    ).map((parts) => parts.filter((e) => e).join(" && "));

    // Combine multiple resources with || into a single condition.
    let resourceCondition = "";
    const nonEmpty = resourceExpressions.filter((e) => e);
    if (nonEmpty.length === 1) {
      resourceCondition = nonEmpty[0];
    } else if (nonEmpty.length > 1) {
      resourceCondition = nonEmpty.map((e) => `(${e})`).join(" || ");
    }

    const parts = [resourceCondition, ...extraExpressions].filter((e) => e);
    exemptions.push(
      create(MaskingExemptionPolicy_ExemptionSchema, {
        members: state.memberList,
        condition: create(ExprSchema, {
          description: state.description,
          expression: parts.length > 0 ? parts.join(" && ") : "",
        }),
      })
    );
  }

  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath,
    policyType: PolicyType.MASKING_EXEMPTION,
  });
  const existed =
    policy?.policy?.case === "maskingExemptionPolicy"
      ? policy.policy.value.exemptions
      : [];
  return {
    name: policy?.name,
    type: PolicyType.MASKING_EXEMPTION,
    resourceType: PolicyResourceType.PROJECT,
    policy: {
      case: "maskingExemptionPolicy",
      value: create(MaskingExemptionPolicySchema, {
        exemptions: [...existed, ...exemptions],
      }),
    },
  };
};
</script>
