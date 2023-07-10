<template>
  <div class="w-full h-full relative">
    <template v-if="false">
      <div>{{ issueUID }}</div>
      <div><NButton @click="tryCreate">try create</NButton></div>
    </template>
    <IssueDetailPage v-if="ready" />
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <BBSpin />
    </div>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watchEffect } from "vue";
import { _RouteLocationBase } from "vue-router";
import { useI18n } from "vue-i18n";

import { useTitle } from "@vueuse/core";
import { idFromSlug } from "@/utils";
import { EMPTY_ID, UNKNOWN_ID, emptyIssue, unknownIssue } from "@/types";
import { rolloutServiceClient } from "@/grpcweb";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/rollout_service";
import { experimentalFetchIssueByUID } from "@/store";
import {
  IssueDetailPage,
  provideIssueContext,
  useBaseIssueContext,
} from "@/components/IssueV1";

interface LocalState {
  showFeatureModal: boolean;
}

const props = defineProps({
  issueSlug: {
    required: true,
    type: String,
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const issueSlug = computed(() => props.issueSlug);

const issueUID = computed(() => {
  if (issueSlug.value === "new") return String(EMPTY_ID);
  const uid = Number(idFromSlug(issueSlug.value));
  if (Number.isNaN(uid) || uid <= 0) return String(UNKNOWN_ID);
  return String(uid);
});

const isCreating = computed(() => issueUID.value === String(EMPTY_ID));
const ready = ref(false);
const issue = ref(unknownIssue());

const tryFetchIssue = async (uid: string) => {
  ready.value = false;
  issue.value = await experimentalFetchIssueByUID(uid);
  ready.value = true;
};

watchEffect(() => {
  const uid = issueUID.value;

  if (uid === String(UNKNOWN_ID)) {
    issue.value = unknownIssue();
    return;
  }
  if (uid === String(EMPTY_ID)) {
    issue.value = emptyIssue();
    return;
  }

  tryFetchIssue(uid);
});

provideIssueContext(
  {
    isCreating,
    issue,
    ...useBaseIssueContext({
      isCreating,
      ready,
      issue,
    }),
  },
  true /* root */
);

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("issue.new-issue");
  } else {
    if (ready.value && issue.value.uid !== String(UNKNOWN_ID)) {
      return issue.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);

const tryCreate = async () => {
  const projectResource = "projects/project-d2a1c91c";
  const databaseResource = "instances/instance-36d77ff4/databases/employee";
  const sheetResource = "projects/project-d2a1c91c/sheets/1424";
  try {
    const config = Plan_ChangeDatabaseConfig.fromJSON({
      target: databaseResource,
      sheet: sheetResource,
      type: Plan_ChangeDatabaseConfig_Type.DATA,
    });
    const spec = Plan_Spec.fromJSON({
      changeDatabaseConfig: config,
    });
    const step = Plan_Step.fromJSON({
      specs: [spec],
    });
    const plan = Plan.fromJSON({
      steps: [step],
    });
    // const plan = await rolloutServiceClient.createPlan({
    //   parent: projectResource,
    //   plan: {
    //     steps: [step],
    //   },
    // });
    await rolloutServiceClient.previewRollout({
      project: projectResource,
      plan,
    });
    console.log("plan", plan);
    // const issue = await issueServiceClient.createIssue({
    //   parent: projectResource,
    //   review: {
    //     assignee: "users/lj@bytebase.com",
    //     plan: plan.name,
    //   },
    // });
    // console.log(issue);
    // const plan = Plan.fromJSON({
    //   steps: [step],
    // });
    // await issueServiceClient.createIssue({
    //   parent,
    //   review: {
    //     assignee: "users/lj@bytebase.com",
    //     'plan'
    //   },
    // });
  } catch (err) {
    console.log("error", err);
  }
};
</script>

<style lang="postcss">
.issue-debug {
  @apply bg-red-200/50 font-mono text-xs;
}
</style>
