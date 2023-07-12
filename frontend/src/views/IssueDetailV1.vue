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
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { v4 as uuidv4 } from "uuid";

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
import {
  experimentalFetchIssueByUID,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
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

const route = useRoute();
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

const tryInitializeIssue = async (uid: string) => {
  ready.value = false;
  // if (uid === String(UNKNOWN_ID)) {
  //   issue.value = unknownIssue();
  // }
  // if (uid === String(EMPTY_ID)) {
  //   issue.value = emptyIssue();
  // }

  const project = await useProjectV1Store().getOrFetchProjectByUID(
    route.query.project as string
  );
  // const template = route.query.template as TemplateType;

  issue.value = emptyIssue();
  issue.value.project = project.name;
  issue.value.projectEntity = project;

  const test = async (
    type: Plan_ChangeDatabaseConfig_Type,
    targets: string[]
  ) => {
    const specs = targets.map((target) => {
      const config = Plan_ChangeDatabaseConfig.fromJSON({
        target,
        // sheet: `${project.name}/sheets/101`,
        sheet: `${project.name}/sheets/10086`,
        type,
        rollbackEnabled: true,
      });
      const spec = Plan_Spec.fromJSON({
        changeDatabaseConfig: config,
        id: uuidv4(),
      });
      return spec;
    });
    const step = Plan_Step.fromJSON({
      specs,
    });
    const plan = Plan.fromJSON({
      steps: [step],
    });
    console.log("plan", plan);
    issue.value.plan = plan.name;
    issue.value.planEntity = plan;
    const rollout = await rolloutServiceClient.previewRollout({
      project: project.name,
      plan,
    });
    console.log("rollout", rollout);
    issue.value.rollout = rollout.name;
    issue.value.rolloutEntity = rollout;
  };

  if (route.query.mode === "tenant") {
    await test(Plan_ChangeDatabaseConfig_Type.DATA, [
      `${project.name}/deploymentConfig`,
    ]);
  } else {
    const databaseUIDList = (route.query.databaseList as string).split(",");

    const databaseList = await Promise.all(
      databaseUIDList.map((uid) =>
        useDatabaseV1Store().getOrFetchDatabaseByUID(uid)
      )
    );
    await test(
      Plan_ChangeDatabaseConfig_Type.DATA,
      databaseList.map((db) => db.name)
    );
  }

  ready.value = true;
};

watchEffect(() => {
  const uid = issueUID.value;

  if (uid === String(UNKNOWN_ID) || uid === String(EMPTY_ID)) {
    tryInitializeIssue(uid);
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
    const rollout = await rolloutServiceClient.previewRollout({
      project: projectResource,
      plan,
    });
    console.log("plan", plan);
    console.log("rollout", rollout);
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
