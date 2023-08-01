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
      <ErrorList :errors="issueCreateErrorList" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NTooltip, NButton } from "naive-ui";

import { CreateIssueRequest, Issue } from "@/types/proto/v1/issue_service";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Rollout,
} from "@/types/proto/v1/rollout_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetV1Store } from "@/store";
import {
  getLocalSheetByName,
  isValidStage,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { ErrorList } from "@/components/IssueV1/components/common";
import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { extractSheetUID } from "@/utils";
import { ComposedIssue } from "@/types";
import { useRouter } from "vue-router";

const router = useRouter();
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
  const createdPlan = await createPlan();
  if (!createdPlan) return;

  issue.value.plan = createdPlan.name;
  issue.value.planEntity = createdPlan;

  console.log(
    "CreateIssueRequest",
    JSON.stringify(
      CreateIssueRequest.toJSON(
        CreateIssueRequest.fromJSON({
          parent: issue.value.project,
          issue: issue.value,
        })
      ),
      null,
      "  "
    )
  );

  const createdIssue = await issueServiceClient.createIssue({
    parent: issue.value.project,
    issue: issue.value,
  });

  const createdRollout = await rolloutServiceClient.createRollout({
    parent: issue.value.project,
    plan: createdPlan.name,
  });
  console.log(
    "createdRollout",
    JSON.stringify(Rollout.toJSON(createdRollout), null, "  ")
  );

  createdIssue.rollout = createdRollout.name;

  console.log(
    "created issue",
    JSON.stringify(Issue.toJSON(createdIssue), null, "  ")
  );

  const composedIssue: ComposedIssue = {
    ...issue.value,
    ...createdIssue,
    planEntity: createdPlan,
    rolloutEntity: createdRollout,
  };
  console.log("created composed issue", composedIssue);

  router.push(`/issue-v1/${composedIssue.uid}`);

  return composedIssue;
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
  return createdPlan;
};
</script>
