<template>
  <NTooltip :disabled="issueCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="large"
        tag="div"
        :disabled="issueCreateErrorList.length > 0"
        @click="doCreateIssue"
      >
        {{ $t("common.create") }}
      </NButton>
    </template>

    <template #default>
      <ul
        class="flex flex-col gap-y-2 whitespace-nowrap"
        :class="[issueCreateErrorList.length > 1 && 'list-disc pl-4']"
      >
        <li v-for="(error, i) in issueCreateErrorList" :key="i">
          {{ error }}
        </li>
      </ul>
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NTooltip, NButton } from "naive-ui";

import { Issue } from "@/types/proto/v1/issue_service";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
} from "@/types/proto/v1/rollout_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetV1Store } from "@/store";
import {
  getLocalSheetByName,
  isValidStage,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { extractSheetUID } from "@/utils";

const { issue } = useIssueContext();

const issueCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (issue.value.rolloutEntity.stages.some((stage) => !isValidStage(stage))) {
    errorList.push("Missing SQL statement in some stages.");
  }
  if (!issue.value.assignee) {
    errorList.push("Assignee is required.");
  }
  return errorList;
});

const doCreateIssue = async () => {
  await createSheets();
  const plan = await createPlan();
  if (!plan) return;

  const createdIssue = await issueServiceClient.createIssue({
    parent: issue.value.project,
    issue: issue.value,
  });

  console.log("created issue", Issue.toJSON(createdIssue));
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
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      // The sheet is pending create
      const sheet = getLocalSheetByName(config.sheet);
      sheet.database = config.target;
      pendingCreateSheetMap.set(sheet.name, sheet);
    }
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
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      config.sheet = sheetNameMap.get(config.sheet) ?? "";
    }
  });
};

const createPlan = async () => {
  const plan = issue.value.planEntity;
  if (!plan) return;
  const createdPlan = await rolloutServiceClient.createPlan({
    parent: issue.value.project,
    plan,
  });
  console.log("created plan", Plan.toJSON(createdPlan));
  issue.value.plan = createPlan.name;
  return createdPlan;
};
</script>
