<template>
  <div class="flex flex-row items-center gap-2">
    <slot name="result" :advices="filteredAdvices" :is-running="isRunning">
      <SQLCheckSummary
        v-if="filteredAdvices !== undefined && !isRunning"
        :database="database"
        :advices="filteredAdvices"
        @click="showDetailPanel = true"
      />
    </slot>

    <NPopover :disabled="policyErrors.length === 0" to="body">
      <template #trigger>
        <NButton
          style="--n-padding: 0 14px 0 10px"
          :disabled="policyErrors.length > 0"
          :style="buttonStyle"
          tag="div"
          v-bind="buttonProps"
          @click="handleButtonClick"
        >
          <template #icon>
            <BBSpin v-if="isRunning" :size="20" />
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
                :to="{ name: WORKSPACE_ROUTE_SQL_REVIEW }"
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
      v-if="filteredAdvices && showDetailPanel"
      :database="database"
      :advices="filteredAdvices"
      :affected-rows="checkResult.affectedRows"
      :risk-level="checkResult.riskLevel"
      :confirm="confirmDialog"
      :override-title="$t('issue.sql-check.sql-review-violations')"
      :show-code-location="showCodeLocation"
      :allow-force-continue="allowForceContinue"
      :ignore-issue-creation-restriction="ignoreIssueCreationRestriction"
      @close="onPanelClose"
    >
      <template #row-title-extra="{ row }">
        <slot name="row-title-extra" :row="row" :confirm="confirmDialog" />
      </template>
    </SQLCheckPanel>
  </div>
</template>

<script lang="ts" setup>
import type { ButtonProps } from "naive-ui";
import { NButton, NPopover } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, onUnmounted, ref, watch } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { releaseServiceClient } from "@/grpcweb";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import { useReviewPolicyForDatabase } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  CheckReleaseResponse,
  Release_File_ChangeType,
  ReleaseFileType,
} from "@/types/proto/v1/release_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import type { Defer, VueStyle } from "@/utils";
import { defer, hasWorkspacePermissionV2 } from "@/utils";
import { providePlanCheckRunContext } from "../PlanCheckRun/context";
import ErrorList from "../misc/ErrorList.vue";
import SQLCheckPanel from "./SQLCheckPanel.vue";
import SQLCheckSummary from "./SQLCheckSummary.vue";
import { useSQLCheckContext } from "./context";

const props = withDefaults(
  defineProps<{
    getStatement: () => Promise<{
      errors: string[];
      statement: string;
    }>;
    database: ComposedDatabase;
    databaseMetadata?: DatabaseMetadata;
    buttonProps?: ButtonProps;
    buttonStyle?: VueStyle;
    changeType?: Release_File_ChangeType;
    showCodeLocation?: boolean;
    ignoreIssueCreationRestriction?: boolean;
    adviceFilter?: (advices: Advice, index: number) => boolean;
  }>(),
  {
    databaseMetadata: undefined,
    buttonProps: undefined,
    buttonStyle: undefined,
    changeType: undefined,
    showCodeLocation: undefined,
    ignoreIssueCreationRestriction: false,
    adviceFilter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:advices", advices: Advice[] | undefined): void;
}>();

const { t } = useI18n();
// SKIP_CHECK_THRESHOLD is the MaxSheetCheckSize in the backend.
const SKIP_CHECK_THRESHOLD = 2 * 1024 * 1024;
const isRunning = ref(false);
const showDetailPanel = ref(false);
const allowForceContinue = ref(true);
const context = useSQLCheckContext();
const confirmDialog = ref<Defer<boolean>>();
const checkResult = ref<CheckReleaseResponse>(
  CheckReleaseResponse.fromPartial({})
);

const filteredAdvices = computed(() => {
  const { adviceFilter } = props;
  const advices = checkResult.value.results.flatMap((r) => r.advices);
  if (!adviceFilter) {
    return advices;
  }
  return advices?.filter(adviceFilter);
});

const reviewPolicy = useReviewPolicyForDatabase(
  computed(() => {
    return props.database;
  })
);

const hasManageSQLReviewPolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const noReviewPolicyTips = computed(() => {
  if (
    !reviewPolicy.value ||
    !reviewPolicy.value.enforce ||
    reviewPolicy.value.ruleList.length === 0
  ) {
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

providePlanCheckRunContext({});

const runCheckInternal = async (statement: string) => {
  const { database, changeType } = props;
  const result = await releaseServiceClient.checkRelease({
    parent: database.project,
    release: {
      files: [
        {
          // Use a random uuid to avoid duplication.
          version: uuidv4(),
          type: ReleaseFileType.VERSIONED,
          statement: statement,
          // Default to DDL change type.
          changeType: changeType || Release_File_ChangeType.DDL,
        },
      ],
    },
    targets: [database.name],
  });
  return result;
};

const handleButtonClick = async () => {
  await runChecks();

  if (hasError.value) {
    const d = defer<boolean>();
    d.promise.finally(onPanelClose);
    showDetailPanel.value = true;
    return d.promise;
  }
};

const runChecks = async () => {
  if (policyErrors.value.length > 0) {
    return;
  }

  const handleErrors = (errors: string[]) => {
    // Mock the pre-check errors to advices.
    checkResult.value = CheckReleaseResponse.fromPartial({
      results: [
        {
          advices: errors.map((err) =>
            Advice.fromPartial({
              title: "Pre check",
              status: Advice_Status.WARNING,
              content: err,
            })
          ),
        },
      ],
    });
    isRunning.value = false;
  };

  isRunning.value = true;
  const { statement, errors } = await props.getStatement();
  allowForceContinue.value = errors.length === 0;
  if (new Blob([statement]).size > SKIP_CHECK_THRESHOLD) {
    return handleErrors([t("issue.sql-check.statement-is-too-large")]);
  }
  if (errors.length > 0) {
    return handleErrors(errors);
  }
  try {
    checkResult.value = await runCheckInternal(statement);
  } finally {
    isRunning.value = false;
  }
};

const onPanelClose = () => {
  showDetailPanel.value = false;
  confirmDialog.value = undefined;
};

const hasError = computed(() => {
  return filteredAdvices.value?.some(
    (advice) =>
      advice.status === Advice_Status.ERROR ||
      advice.status === Advice_Status.WARNING
  );
});

onMounted(() => {
  if (!context) return;
  context.runSQLCheck.value = async () => {
    if (policyErrors.value.length > 0) {
      // If SQL Check is disabled, we will do nothing to stop the user.
      return true;
    }

    await runChecks();
    if (hasError.value) {
      const d = defer<boolean>();
      confirmDialog.value = d;
      d.promise.finally(onPanelClose);

      showDetailPanel.value = true;

      return d.promise;
    }

    return true;
  };
});

onUnmounted(() => {
  if (!context) return;
  context.runSQLCheck.value = undefined;
});

watch(filteredAdvices, (advices) => {
  emit("update:advices", advices);
});
</script>
