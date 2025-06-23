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
import { create } from "@bufbuild/protobuf";
import {
  ErrorList,
  useSpecsValidation,
} from "@/components/Plan/components/common";
import {
  databaseEngineForSpec,
  getLocalSheetByName,
} from "@/components/Plan/logic";
import { usePlanContext } from "@/components/Plan/logic";
import { planServiceClientConnect } from "@/grpcweb";
import { CreatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { convertOldPlanToNew, convertNewPlanToOld } from "@/utils/v1/plan-conversions";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useSheetV1Store } from "@/store";
import { type Plan_ChangeDatabaseConfig } from "@/types/proto/v1/plan_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
import {
  extractPlanUID,
  extractProjectResourceName,
  extractSheetUID,
  hasProjectPermissionV2,
} from "@/utils";

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const sheetStore = useSheetV1Store();
const loading = ref(false);

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(plan.value.specs);

const planCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (!hasProjectPermissionV2(project.value, "bb.plans.create")) {
    errorList.push(t("common.missing-required-permission"));
  }
  if (!plan.value.title.trim()) {
    errorList.push("Missing plan title");
  }
  if (plan.value.specs.some((spec) => isSpecEmpty(spec))) {
    errorList.push("Missing statement");
  }
  return errorList;
});

const doCreatePlan = async () => {
  loading.value = true;

  try {
    await createSheets();
    const newPlan = convertOldPlanToNew(plan.value);
    const request = create(CreatePlanRequestSchema, {
      parent: project.value.name,
      plan: newPlan,
    });
    const response = await planServiceClientConnect.createPlan(request);
    const createdPlan = convertNewPlanToOld(response);
    if (!createdPlan) return;

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        params: {
          projectId: extractProjectResourceName(createdPlan.name),
          planId: extractPlanUID(createdPlan.name),
        },
      });
    });

    return createdPlan;
  } catch {
    loading.value = false;
  }
};

// Create sheets for spec configs and update their resource names.
const createSheets = async () => {
  const specs = plan.value.specs || [];
  const configWithSheetList: Plan_ChangeDatabaseConfig[] = [];
  const pendingCreateSheetMap = new Map<string, Sheet>();

  for (let i = 0; i < specs.length; i++) {
    const spec = specs[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    configWithSheetList.push(config);
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      // The sheet is pending create
      const sheet = getLocalSheetByName(config.sheet);
      const engine = await databaseEngineForSpec(spec);
      sheet.engine = engine;
      pendingCreateSheetMap.set(sheet.name, sheet);
    }
  }
  const pendingCreateSheetList = Array.from(pendingCreateSheetMap.values());
  const sheetNameMap = new Map<string, string>();
  for (let i = 0; i < pendingCreateSheetList.length; i++) {
    const sheet = pendingCreateSheetList[i];
    sheet.title = plan.value.title;
    const createdSheet = await sheetStore.createSheet(
      project.value.name,
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
</script>
