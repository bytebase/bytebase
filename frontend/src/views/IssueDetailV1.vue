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
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import Emittery from "emittery";
import { useTitle } from "@vueuse/core";

import { UNKNOWN_ID } from "@/types";
import {
  IssueDetailPage,
  IssueEvents,
  provideIssueContext,
  useBaseIssueContext,
  useInitializeIssue,
} from "@/components/IssueV1";
import { uidFromSlug } from "@/utils";
import { experimentalFetchIssueByUID } from "@/store";
import { useProgressivePoll } from "@/composables/useProgressivePoll";

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
const events: IssueEvents = new Emittery();

const pollIssue = () => {
  if (!isCreating.value && ready.value) {
    const uid = uidFromSlug(issueSlug.value);

    experimentalFetchIssueByUID(uid).then(
      (updatedIssue) => (issue.value = updatedIssue)
    );
  }
};

const poller = useProgressivePoll(pollIssue, {
  interval: {
    min: 500,
    max: 10000,
    growth: 2,
    jitter: 500,
  },
});

watch(
  [isCreating, ready],
  () => {
    if (!isCreating.value && ready.value) {
      poller.start();
    } else {
      poller.stop();
    }
  },
  {
    immediate: true,
  }
);

events.on("status-changed", ({ eager }) => {
  if (eager) {
    pollIssue();
    poller.restart();
  }
});

provideIssueContext(
  {
    isCreating,
    issue,
    ...useBaseIssueContext({
      isCreating,
      ready,
      issue,
      events,
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
  @apply hidden bg-red-200/50 font-mono text-xs;
}
</style>
