<template>
  <NTooltip :disabled="issueCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="issueCreateErrorList.length > 0 || loading"
        :loading="loading"
        @click="doCreateIssue"
      >
        {{ loading ? $t("common.creating") : $t("issue.create-issue") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="issueCreateErrorList" />
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
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext } from "@/components/Plan/logic";
import { useSQLCheckContext } from "@/components/SQLCheck";
import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentUserV1 } from "@/store";
import { emptyIssue, type ComposedIssue } from "@/types";
import { Issue, IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  issueSlug,
} from "@/utils";

const { t } = useI18n();
const router = useRouter();
const { plan } = usePlanContext();
const { runSQLCheck } = useSQLCheckContext();
const me = useCurrentUserV1();
const loading = ref(false);

const issueCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (
    !hasProjectPermissionV2(
      plan.value.projectEntity,
      me.value,
      "bb.plans.create"
    )
  ) {
    errorList.push(t("common.missing-permission"));
  }
  if (!plan.value.title.trim()) {
    errorList.push("Missing issue title");
  }

  return errorList;
});

const doCreateIssue = async () => {
  loading.value = true;
  const check = runSQLCheck.value;
  if (check && !(await check())) {
    loading.value = false;
    return;
  }

  try {
    const createdIssue = await issueServiceClient.createIssue({
      parent: plan.value.project,
      issue: {
        ...Issue.fromPartial(buildIssue()),
        rollout: "",
        plan: plan.value.name,
      },
    });
    const composedIssue: ComposedIssue = {
      ...emptyIssue(),
      ...createdIssue,
      planEntity: plan.value,
    };
    const createdRollout = await rolloutServiceClient.createRollout({
      parent: plan.value.project,
      rollout: {
        plan: plan.value.name,
      },
    });

    composedIssue.rollout = createdRollout.name;
    composedIssue.rolloutEntity = createdRollout;

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.value.project),
          issueSlug: issueSlug(composedIssue.title, composedIssue.uid),
        },
      });
    });

    return composedIssue;
  } catch {
    loading.value = false;
  }
};

const buildIssue = () => {
  const issue = emptyIssue();
  const me = useCurrentUserV1();
  issue.creator = `users/${me.value.email}`;
  issue.creatorEntity = me.value;
  issue.project = plan.value.projectEntity.name;
  issue.projectEntity = plan.value.projectEntity;
  issue.title = plan.value.title;
  issue.status = IssueStatus.OPEN;
  issue.type = Issue_Type.DATABASE_CHANGE;
  return issue;
};
</script>
