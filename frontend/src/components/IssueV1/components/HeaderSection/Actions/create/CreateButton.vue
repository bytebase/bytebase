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
import { Sheet } from "@/types/proto/v1/sheet_service";

const { issue } = useIssueContext();

const doCreateIssue = async () => {
  await createSheets();
};

const createSheets = async () => {
  const steps = issue.value.planEntity?.steps ?? [];
  const flattenSpecList = steps.flatMap((step) => {
    return step.specs;
  });
  const pendingCreateSheetMap = new Map<string, Sheet>();
  for (let i = 0; i < flattenSpecList.length; i++) {
    const spec = flattenSpecList[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const sheet = getLocalSheetByName(config.sheet);
    sheet.database = config.target;
    pendingCreateSheetMap.set(sheet.name, sheet);
  }
  const pendingCreateSheetList = Array.from(pendingCreateSheetMap.values());
  console.log("pendingCreateSheetList", pendingCreateSheetList);
};
</script>
