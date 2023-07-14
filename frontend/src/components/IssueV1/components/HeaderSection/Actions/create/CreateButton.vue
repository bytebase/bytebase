<template>
  <NButton type="primary" size="large" @click="doCreateIssue">
    {{ $t("common.create") }}
  </NButton>
</template>

<script setup lang="ts">
import {
  getLocalSheetByName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useSheetV1Store } from "@/store";
import { Plan_ChangeDatabaseConfig } from "@/types/proto/v1/rollout_service";
import { Sheet } from "@/types/proto/v1/sheet_service";

const { issue } = useIssueContext();

const doCreateIssue = async () => {
  await createSheets();
};

// Create sheets for spec configs and update their resource names.
const createSheets = async () => {
  const steps = issue.value.planEntity?.steps ?? [];
  const flattenSpecList = steps.flatMap((step) => {
    return step.specs;
  });

  const configWithSheetList: Plan_ChangeDatabaseConfig[] = [];
  const pendingCreateSheetMap = new Map<string, Sheet>();

  for (let i = 0; i < flattenSpecList.length; i++) {
    const spec = flattenSpecList[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    configWithSheetList.push(config);
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const sheet = getLocalSheetByName(config.sheet);
    sheet.database = config.target;
    pendingCreateSheetMap.set(sheet.name, sheet);
  }
  const pendingCreateSheetList = Array.from(pendingCreateSheetMap.values());
  const sheetNameMap = new Map<string, string>();
  for (let i = 0; i < pendingCreateSheetList.length; i++) {
    const sheet = pendingCreateSheetList[i];
    sheet.title = issue.value.title;
    const createdSheet = await useSheetV1Store().createSheet(
      issue.value.project,
      sheet
    );
    sheetNameMap.set(sheet.name, createdSheet.name);
  }
  configWithSheetList.forEach((config) => {
    config.sheet = sheetNameMap.get(config.sheet) ?? "";
  });
};
</script>
