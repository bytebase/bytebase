<template>
  <NPopover :disabled="statementErrors.length === 0" to="body">
    <template #trigger>
      <NButton
        style="--n-padding: 0 14px 0 10px"
        :disabled="statementErrors.length > 0"
        :style="buttonStyle"
        tag="div"
        size="small"
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
      <ErrorList :errors="statementErrors" />
    </template>
  </NPopover>

  <SQLCheckPanel
    v-if="checkResult && advices && showDetailPanel"
    :project="plan.project"
    :database="database"
    :advices="advices"
    :affected-rows="checkResult.affectedRows"
    :risk-level="checkResult.riskLevel"
    :confirm="confirmDialog"
    :override-title="$t('issue.sql-check.sql-review-violations')"
    :show-code-location="showCodeLocation"
    :allow-force-continue="allowForceContinue"
    @close="onPanelClose"
  >
    <template #row-title-extra="{ row }">
      <slot name="row-title-extra" :row="row" :confirm="confirmDialog" />
    </template>
  </SQLCheckPanel>
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import { NButton, NPopover } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { SQLCheckPanel } from "@/components/SQLCheck";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "@/components/SQLCheck/common";
import ErrorList from "@/components/misc/ErrorList.vue";
import { releaseServiceClient } from "@/grpcweb";
import {
  CheckReleaseResponse,
  ReleaseFileType,
} from "@/types/proto/v1/release_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import type { Defer, VueStyle } from "@/utils";
import { defer } from "@/utils";
import { databaseForSpec, usePlanContext } from "../../logic";
import { useSpecSheet } from "../StatementSection/useSpecSheet";
import { getSpecChangeType } from "./common";
import { usePlanSQLCheckContext } from "./context";

withDefaults(
  defineProps<{
    buttonStyle?: VueStyle;
    showCodeLocation?: boolean;
  }>(),
  {
    buttonStyle: undefined,
    showCodeLocation: undefined,
  }
);

const { t } = useI18n();
const { plan, selectedSpec } = usePlanContext();
const { upsertResult } = usePlanSQLCheckContext();
const { sheetStatement } = useSpecSheet();

const isRunning = ref(false);
const showDetailPanel = ref(false);
const allowForceContinue = ref(true);
const confirmDialog = ref<Defer<boolean>>();
const checkResult = ref<CheckReleaseResponse | undefined>();

const advices = computed(() => {
  return checkResult.value?.results.flatMap((r) => r.advices);
});

const database = computed(() =>
  databaseForSpec(plan.value.projectEntity, selectedSpec.value)
);

const statement = computed(() => sheetStatement.value);

const changeType = computed(() => getSpecChangeType(selectedSpec.value));

const statementErrors = asyncComputed(async () => {
  if (statement.value.length === 0) {
    return [t("issue.sql-check.statement-is-required")];
  }
  if (new Blob([statement.value]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
    return [t("issue.sql-check.statement-is-too-large")];
  }
  return [];
}, []);

const runCheckInternal = async (statement: string) => {
  const result = await releaseServiceClient.checkRelease({
    parent: database.value.project,
    release: {
      files: [
        {
          // Use a random uuid to avoid duplication.
          version: uuidv4(),
          type: ReleaseFileType.VERSIONED,
          statement: new TextEncoder().encode(statement),
          changeType: changeType.value,
        },
      ],
    },
    targets: [database.value.name],
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
  if (statementErrors.value.length > 0) {
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
  if (new Blob([statement.value]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
    return handleErrors([t("issue.sql-check.statement-is-too-large")]);
  }
  try {
    checkResult.value = await runCheckInternal(statement.value);
  } finally {
    isRunning.value = false;
  }
};

const onPanelClose = () => {
  showDetailPanel.value = false;
  confirmDialog.value = undefined;
};

const hasError = computed(() => {
  return advices.value?.some(
    (advice) =>
      advice.status === Advice_Status.ERROR ||
      advice.status === Advice_Status.WARNING
  );
});

watch(checkResult, (result) => {
  for (const r of result?.results || []) {
    upsertResult(r.target, r);
  }
});
</script>
