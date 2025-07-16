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
    :project="project.name"
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
import { create } from "@bufbuild/protobuf";
import { asyncComputed } from "@vueuse/core";
import { NButton, NPopover } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { SQLCheckPanel } from "@/components/SQLCheck";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "@/components/SQLCheck/common";
import ErrorList from "@/components/misc/ErrorList.vue";
import { releaseServiceClientConnect } from "@/grpcweb";
import type { CheckReleaseResponse } from "@/types/proto-es/v1/release_service_pb";
import {
  CheckReleaseRequestSchema,
  CheckReleaseResponseSchema,
  Release_File_Type,
} from "@/types/proto-es/v1/release_service_pb";
import {
  AdviceSchema,
  Advice_Status as ProtoESAdvice_Status,
} from "@/types/proto-es/v1/sql_service_pb";
import { Advice_Status } from "@/types/proto-es/v1/sql_service_pb";
import type { Defer, VueStyle } from "@/utils";
import { defer } from "@/utils";
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
const { database, project, selectedSpec, upsertResult } =
  usePlanSQLCheckContext();
const { sheetStatement } = useSpecSheet(selectedSpec.value);

const isRunning = ref(false);
const showDetailPanel = ref(false);
const allowForceContinue = ref(true);
const confirmDialog = ref<Defer<boolean>>();
const checkResult = ref<CheckReleaseResponse | undefined>();

const advices = computed(() => {
  return checkResult.value?.results.flatMap((r) => r.advices);
});

const changeType = computed(() => getSpecChangeType(selectedSpec.value));

const statementErrors = asyncComputed(async () => {
  if (sheetStatement.value.length === 0) {
    return [t("issue.sql-check.statement-is-required")];
  }
  if (new Blob([sheetStatement.value]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
    return [t("issue.sql-check.statement-is-too-large")];
  }
  return [];
}, []);

const runCheckInternal = async (statement: string) => {
  const request = create(CheckReleaseRequestSchema, {
    parent: project.value.name,
    release: {
      files: [
        {
          // Use "0" for dummy version.
          version: "0",
          type: Release_File_Type.VERSIONED,
          statement: new TextEncoder().encode(statement),
          changeType: changeType.value,
        },
      ],
    },
    targets: [database.value.name],
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
              status: ProtoESAdvice_Status.WARNING,
              content: err,
            })
          ),
        },
      ],
    });
    isRunning.value = false;
  };

  isRunning.value = true;
  if (new Blob([sheetStatement.value]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
    return handleErrors([t("issue.sql-check.statement-is-too-large")]);
  }
  try {
    checkResult.value = await runCheckInternal(sheetStatement.value);
    // Upsert the result to the map.
    for (const r of checkResult.value.results || []) {
      // target is the database name.
      upsertResult(r.target, r);
    }
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
</script>
