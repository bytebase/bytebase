<template>
  <NTooltip :disabled="planCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="planCreateErrorList.length > 0 || loading"
        :loading="loading"
        @click="doCreatePlan"
      >
        {{ loading ? $t("common.creating") : $t("common.create") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="planCreateErrorList" />
    </template>
  </NTooltip>

  <!-- prevent clicking the page when creating in progress -->
  <div
    v-if="loading"
    v-zindexable="{ enabled: true }"
    class="fixed inset-0 pointer-events-auto flex flex-col items-center justify-center"
    @click.stop.prevent
  />
</template>

<script setup lang="ts">
import { NTooltip, NButton } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { ErrorList } from "@/components/Plan/components/common";
import {
  databaseEngineForSpec,
  getLocalSheetByName,
  isValidSpec,
} from "@/components/Plan/logic";
import { usePlanContext } from "@/components/Plan/logic";
import { useSQLCheckContext } from "@/components/SQLCheck";
import { planServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useSheetV1Store } from "@/store";
import { dialectOfEngineV1, languageOfEngineV1 } from "@/types";
import { type Plan_ChangeDatabaseConfig } from "@/types/proto/v1/plan_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import {
  extractProjectResourceName,
  extractSheetUID,
  flattenSpecList,
  getSheetStatement,
  hasProjectPermissionV2,
  planV1Slug,
  setSheetStatement,
} from "@/utils";

const MAX_FORMATTABLE_STATEMENT_SIZE = 10000; // 10K characters

const { t } = useI18n();
const router = useRouter();
const { plan, formatOnSave } = usePlanContext();
const { runSQLCheck } = useSQLCheckContext();
const sheetStore = useSheetV1Store();
const loading = ref(false);

const planCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (!hasProjectPermissionV2(plan.value.projectEntity, "bb.plans.create")) {
    errorList.push(t("common.missing-permission"));
  }
  if (!plan.value.title.trim()) {
    errorList.push("Missing plan title");
  }
  if (!flattenSpecList(plan.value).every((spec) => isValidSpec(spec))) {
    errorList.push("Missing SQL statement in some tasks");
  }

  return errorList;
});

const doCreatePlan = async () => {
  loading.value = true;
  const check = runSQLCheck.value;
  if (check && !(await check())) {
    loading.value = false;
    return;
  }

  try {
    await createSheets();
    const createdPlan = await planServiceClient.createPlan({
      parent: plan.value.project,
      plan: plan.value,
    });
    if (!createdPlan) return;

    const composedPlan: ComposedPlan = {
      ...plan.value,
      ...createdPlan,
    };

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(composedPlan.project),
          planSlug: planV1Slug(composedPlan),
        },
      });
    });

    return composedPlan;
  } catch {
    loading.value = false;
  }
};

// Create sheets for spec configs and update their resource names.
const createSheets = async () => {
  const flattenSpecList = plan.value.steps.flatMap((step) => step.specs);
  const configWithSheetList: Plan_ChangeDatabaseConfig[] = [];
  const pendingCreateSheetMap = new Map<string, Sheet>();

  for (let i = 0; i < flattenSpecList.length; i++) {
    const spec = flattenSpecList[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    configWithSheetList.push(config);
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      // The sheet is pending create
      const sheet = getLocalSheetByName(config.sheet);
      const engine = await databaseEngineForSpec(
        plan.value.projectEntity,
        spec
      );
      sheet.engine = engine;
      pendingCreateSheetMap.set(sheet.name, sheet);

      await maybeFormatSQL(sheet, config.target);
    }
  }
  const pendingCreateSheetList = Array.from(pendingCreateSheetMap.values());
  const sheetNameMap = new Map<string, string>();
  for (let i = 0; i < pendingCreateSheetList.length; i++) {
    const sheet = pendingCreateSheetList[i];
    sheet.title = plan.value.title;
    const createdSheet = await sheetStore.createSheet(
      plan.value.project,
      sheet
    );
    sheetNameMap.set(sheet.name, createdSheet.name);
  }
  configWithSheetList.forEach((config) => {
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      config.sheet = sheetNameMap.get(config.sheet) ?? "";
    }
  });
};

const maybeFormatSQL = async (sheet: Sheet, target: string) => {
  if (!formatOnSave.value) {
    return;
  }
  const db = await useDatabaseV1Store().getOrFetchDatabaseByName(target);
  if (!db) {
    return;
  }
  const language = languageOfEngineV1(db.instanceResource.engine);
  if (language !== "sql") {
    return;
  }

  const dialect = dialectOfEngineV1(db.instanceResource.engine);
  const statement = getSheetStatement(sheet);
  if (statement.length > MAX_FORMATTABLE_STATEMENT_SIZE) {
    return;
  }
  const { error, data: formatted } = await formatSQL(statement, dialect);
  if (error) {
    return;
  }

  setSheetStatement(sheet, formatted);
};
</script>
