<template>
  <div class="flex flex-row items-center gap-2">
    <slot
      name="result"
      :is-running="isRunning"
      :advices="filteredAdvices"
      :affected-rows="checkResult?.affectedRows"
      :risk-level="checkResult?.riskLevel"
    >
      <SQLCheckSummary
        v-if="filteredAdvices && !isRunning"
        :database="database"
        :advices="filteredAdvices"
        @click="showDetailPanel = true"
      />
    </slot>

    <NPopover :disabled="statementErrors.length === 0" to="body">
      <template #trigger>
        <NButton
          style="--n-padding: 0 14px 0 10px"
          :disabled="statementErrors.length > 0"
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
        <ErrorList :errors="statementErrors" />
      </template>
    </NPopover>

    <SQLCheckPanel
      v-if="checkResult && filteredAdvices && showDetailPanel"
      :project="database.project"
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
import { create } from "@bufbuild/protobuf";
import { asyncComputed } from "@vueuse/core";
import type { ButtonProps } from "naive-ui";
import { NButton, NPopover } from "naive-ui";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { releaseServiceClientConnect } from "@/connect";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { CheckReleaseResponse } from "@/types/proto-es/v1/release_service_pb";
import {
  CheckReleaseRequestSchema,
  CheckReleaseResponseSchema,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import type { Advice } from "@/types/proto-es/v1/sql_service_pb";
import { Advice_Level, AdviceSchema } from "@/types/proto-es/v1/sql_service_pb";
import type { Defer, VueStyle } from "@/utils";
import { defer } from "@/utils";
import ErrorList from "../misc/ErrorList.vue";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "./common";
import { useSQLCheckContext } from "./context";
import SQLCheckPanel from "./SQLCheckPanel.vue";
import SQLCheckSummary from "./SQLCheckSummary.vue";

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
    enableGhost?: boolean;
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
const isRunning = ref(false);
const showDetailPanel = ref(false);
const allowForceContinue = ref(true);
const context = useSQLCheckContext();
const confirmDialog = ref<Defer<boolean>>();
const checkResult = ref<CheckReleaseResponse | undefined>();

const filteredAdvices = computed(() => {
  const { adviceFilter } = props;
  const advices = checkResult.value?.results.flatMap((r) => r.advices);
  if (!advices) return undefined;
  if (!adviceFilter) {
    return advices;
  }
  return advices?.filter(adviceFilter);
});

const statementErrors = asyncComputed(async () => {
  const { statement, errors } = await props.getStatement();
  if (errors.length > 0) {
    return errors;
  }
  if (statement.length === 0) {
    return [t("issue.sql-check.statement-is-required")];
  }
  if (new Blob([statement]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
    return [t("issue.sql-check.statement-is-too-large")];
  }
  return [];
}, []);

const runCheckInternal = async (statement: string) => {
  const { database, enableGhost } = props;
  const request = create(CheckReleaseRequestSchema, {
    parent: database.project,
    release: {
      type: Release_Type.VERSIONED,
      files: [
        {
          // Use "0" for dummy version.
          version: "0",
          statement: new TextEncoder().encode(statement),
          enableGhost: enableGhost ?? false,
        },
      ],
    },
    targets: [database.name],
  });
  const response = await releaseServiceClientConnect.checkRelease(request);
  return response;
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
    checkResult.value = create(CheckReleaseResponseSchema, {
      results: [
        {
          advices: errors.map((err) =>
            create(AdviceSchema, {
              title: "Pre check",
              status: Advice_Level.WARNING,
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
  if (new Blob([statement]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
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
      advice.status === Advice_Level.ERROR ||
      advice.status === Advice_Level.WARNING
  );
});

onMounted(() => {
  if (!context) return;
  context.runSQLCheck.value = async () => {
    if (statementErrors.value.length > 0) {
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
