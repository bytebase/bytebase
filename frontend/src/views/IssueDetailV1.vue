<template>
  <div class="w-full h-full relative">
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
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useTitle } from "@vueuse/core";

import { UNKNOWN_ID } from "@/types";
import {
  IssueDetailPage,
  provideIssueContext,
  useBaseIssueContext,
  useInitializeIssue,
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

const { isCreating, issue, isInitializing } = useInitializeIssue(issueSlug);
const ready = computed(() => {
  return !isInitializing.value && !!issue.value;
});

provideIssueContext(
  {
    isCreating,
    issue,
    ready,
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

// const tryCreate = async () => {
//   const projectResource = "projects/project-d2a1c91c";
//   const databaseResource = "instances/instance-36d77ff4/databases/employee";
//   const sheetResource = "projects/project-d2a1c91c/sheets/1424";
//   try {
//     const config = Plan_ChangeDatabaseConfig.fromJSON({
//       target: databaseResource,
//       sheet: sheetResource,
//       type: Plan_ChangeDatabaseConfig_Type.DATA,
//     });
//     const spec = Plan_Spec.fromJSON({
//       changeDatabaseConfig: config,
//     });
//     const step = Plan_Step.fromJSON({
//       specs: [spec],
//     });
//     const plan = Plan.fromJSON({
//       steps: [step],
//     });
//     // const plan = await rolloutServiceClient.createPlan({
//     //   parent: projectResource,
//     //   plan: {
//     //     steps: [step],
//     //   },
//     // });
//     const rollout = await rolloutServiceClient.previewRollout({
//       project: projectResource,
//       plan,
//     });
//     console.log("plan", plan);
//     console.log("rollout", rollout);
//     // const issue = await issueServiceClient.createIssue({
//     //   parent: projectResource,
//     //   review: {
//     //     assignee: "users/lj@bytebase.com",
//     //     plan: plan.name,
//     //   },
//     // });
//     // console.log(issue);
//     // const plan = Plan.fromJSON({
//     //   steps: [step],
//     // });
//     // await issueServiceClient.createIssue({
//     //   parent,
//     //   review: {
//     //     assignee: "users/lj@bytebase.com",
//     //     'plan'
//     //   },
//     // });
//   } catch (err) {
//     console.log("error", err);
//   }
// };
</script>

<style lang="postcss">
.issue-debug {
  @apply bg-red-200/50 font-mono text-xs;
}
</style>
