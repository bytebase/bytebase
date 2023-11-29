<template>
  <div class="flex flex-row items-center gap-2">
    <slot name="result" :advices="advices" :is-running="isRunning" />

    <NPopover :disabled="policyErrors.length === 0" to="body">
      <template #trigger>
        <NButton
          style="--n-padding: 0 14px 0 10px"
          :disabled="policyErrors.length > 0"
          :style="buttonStyle"
          tag="div"
          v-bind="buttonProps"
          @click="runChecks"
        >
          <template #icon>
            <BBSpin v-if="isRunning" class="w-4 h-4" />
            <heroicons-outline:play v-else class="w-4 h-4" />
          </template>
          <template #default>
            <template v-if="isRunning">
              {{ $t("task.checking") }}
            </template>
            <template v-else>
              {{ $t("task.run-checks") }}
            </template>
          </template>
        </NButton>
      </template>
      <template #default>
        <template v-if="noReviewPolicyTips">
          <i18n-t :keypath="noReviewPolicyTips" tag="div">
            <template #environment>
              <span>{{ database.effectiveEnvironmentEntity.title }}</span>
            </template>
            <template #link>
              <router-link
                v-if="hasManageSQLReviewPolicyPermission"
                :to="`/setting/sql-review`"
                class="ml-1 normal-link underline"
              >
                {{ $t("common.go-to-configure") }}
              </router-link>
            </template>
          </i18n-t>
        </template>
        <ErrorList v-else :errors="policyErrors" />
      </template>
    </NPopover>

    <SQLCheckPanel
      v-if="database && advices && confirmDialog"
      :database="database"
      :advices="advices"
      :confirm="confirmDialog"
      :override-title="$t('issue.sql-check.sql-review-violations')"
      @close="confirmDialog = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { ButtonProps, NButton, NPopover } from "naive-ui";
import { computed, onUnmounted, ref } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { sqlServiceClient } from "@/grpcweb";
import { usePolicyByParentAndType } from "@/store";
import { ComposedDatabase } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import { Defer, VueStyle, defer, useWorkspacePermissionV1 } from "@/utils";
import ErrorList from "../misc/ErrorList.vue";
import SQLCheckPanel from "./SQLCheckPanel.vue";
import { useSQLCheckContext } from "./context";

const props = withDefaults(
  defineProps<{
    getStatement: () => Promise<{ errors: string[]; statement: string }>;
    database: ComposedDatabase;
    buttonProps?: ButtonProps;
    buttonStyle?: VueStyle;
  }>(),
  {
    buttonProps: undefined,
    buttonStyle: undefined,
  }
);

const { t } = useI18n();
// SKIP_CHECK_THRESHOLD is the MaxSheetCheckSize in the backend.
const SKIP_CHECK_THRESHOLD = 512 * 1024;
const isRunning = ref(false);
const advices = ref<Advice[]>();
const checkRunCounter = ref(0);
const context = useSQLCheckContext();
const confirmDialog = ref<Defer<boolean>>();

const reviewPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.database.effectiveEnvironment,
    policyType: PolicyType.SQL_REVIEW,
  }))
);

const hasManageSQLReviewPolicyPermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-sql-review-policy"
);

const noReviewPolicyTips = computed(() => {
  if (!reviewPolicy.value?.sqlReviewPolicy) {
    if (hasManageSQLReviewPolicyPermission.value) {
      return "issue.sql-check.no-configured-sql-review-policy.admin";
    } else {
      return "issue.sql-check.no-configured-sql-review-policy.developer";
    }
  }
  return "";
});

const policyErrors = computed(() => {
  if (noReviewPolicyTips.value) return [noReviewPolicyTips.value];
  return [];
});

const runCheckInternal = async (statement: string) => {
  const { database } = props;
  const result = await sqlServiceClient.check({
    statement,
    database: database.name,
  });
  checkRunCounter.value++;
  return result;
};

const runChecks = async () => {
  if (policyErrors.value.length > 0) {
    return;
  }

  const handleErrors = (errors: string[]) => {
    // Mock the pre-check errors to advices
    advices.value = errors.map((err) =>
      Advice.fromPartial({
        title: "Pre check",
        status: Advice_Status.WARNING,
        content: err,
      })
    );
    isRunning.value = false;
  };

  isRunning.value = true;
  if (!advices.value) {
    advices.value = [];
  }
  const { statement, errors } = await props.getStatement();
  if (statement.length === 0) {
    return handleErrors([t("common.nothing-changed")]);
  }
  if (statement.length > SKIP_CHECK_THRESHOLD) {
    return handleErrors([t("issue.sql-check.statement-is-too-large")]);
  }
  if (errors.length > 0) {
    return handleErrors(errors);
  }
  try {
    const result = await runCheckInternal(statement);
    advices.value = result.advices;
  } finally {
    isRunning.value = false;
  }
};

onMounted(() => {
  if (!context) return;
  context.runSQLCheck.value = async () => {
    if (policyErrors.value.length > 0) {
      // If SQL Check is disabled, we will do nothing to stop the user.
      return true;
    }

    if (checkRunCounter.value === 0) {
      await runChecks();
    }

    const hasError = advices.value?.some(
      (advice) =>
        advice.status === Advice_Status.ERROR ||
        advice.status === Advice_Status.WARNING
    );
    if (hasError) {
      const d = defer<boolean>();
      confirmDialog.value = d;
      d.promise.finally(() => {
        confirmDialog.value = undefined;
      });

      return d.promise;
    }

    return true;
  };
});

onUnmounted(() => {
  if (!context) return;
  context.runSQLCheck.value = undefined;
});
</script>
