<template>
  <component :is="logicProviderType" ref="issueLogic">
    <div
      id="issue-detail-top"
      class="flex-1 overflow-auto focus:outline-none"
      tabindex="0"
    >
      <IssueBanner v-if="!create" />

      <!-- Highlight Panel -->
      <div class="bg-white px-4 pb-4">
        <IssueHighlightPanel>
          <IssueStatusTransitionButtonGroup />
        </IssueHighlightPanel>
      </div>

      <!-- Remind banner for bb.feature.sql-review -->
      <FeatureAttention
        v-if="!hasSQLReviewPolicyFeature"
        custom-class="m-5 mt-0"
        feature="bb.feature.sql-review"
        :description="$t('subscription.features.bb-feature-sql-review.desc')"
      />

      <!-- Stage Flow Bar -->
      <template v-if="showPipelineFlowBar">
        <template v-if="isTenantMode">
          <PipelineTenantFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else-if="isGhostMode">
          <PipelineGhostFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else-if="isPITRMode">
          <PipelinePITRFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else>
          <PipelineSimpleFlow class="border-t border-b" />
        </template>

        <div v-if="!create" class="px-4 py-4 md:flex md:flex-col border-b">
          <IssueStagePanel />
        </div>
      </template>

      <!-- Output Panel -->
      <!-- Only render the top border if PipelineFlowBar is not displayed, otherwise it would overlap with the bottom border of that -->
      <div
        v-if="showIssueOutputPanel"
        class="px-2 py-4 md:flex md:flex-col"
        :class="showPipelineFlowBar ? '' : 'lg:border-t'"
      >
        <IssueOutputPanel />
      </div>

      <!-- Main Content -->
      <main
        class="flex-1 relative overflow-y-auto focus:outline-none"
        :class="
          showPipelineFlowBar && !showIssueOutputPanel
            ? ''
            : 'lg:border-t lg:border-block-border'
        "
        tabindex="-1"
      >
        <div class="flex max-w-3xl mx-auto px-6 lg:max-w-full">
          <div
            class="flex flex-col flex-1 lg:flex-row-reverse lg:col-span-2 overflow-x-hidden"
          >
            <div
              class="py-6 lg:pl-4 lg:w-72 xl:w-96 lg:border-l lg:border-block-border overflow-hidden"
            >
              <IssueSidebar :database="selectedDatabase" :instance="instance" />
            </div>
            <div class="lg:hidden border-t border-block-border" />
            <div class="w-full lg:w-auto lg:flex-1 py-4 pr-4 overflow-x-hidden">
              <div v-if="!create" class="mb-4">
                <TaskCheckBar
                  :task="(selectedTask as Task)"
                  @run-checks="runTaskChecks"
                />
              </div>
              <section v-if="showIssueTaskStatementPanel" class="border-b mb-4">
                <IssueTaskStatementPanel :sql-hint="sqlHint()" />
              </section>

              <IssueDescriptionPanel />

              <section
                v-if="!create"
                aria-labelledby="activity-title"
                class="mt-4"
              >
                <IssueActivityPanel @run-checks="runTaskChecks" />
              </section>
            </div>
          </div>
        </div>
      </main>
    </div>
  </component>
</template>

<script lang="ts" setup>
import { computed, onMounted, PropType, ref, watch, watchEffect } from "vue";
import { useRoute } from "vue-router";
import {
  pipelineType,
  PipelineType,
  activeStage as activeStageOfPipeline,
  activeTaskInStage,
  activeTask as activeTaskOfPipeline,
} from "@/utils";
import IssueBanner from "./IssueBanner.vue";
import IssueHighlightPanel from "./IssueHighlightPanel.vue";
import IssueStagePanel from "./IssueStagePanel.vue";
import IssueStatusTransitionButtonGroup from "./IssueStatusTransitionButtonGroup.vue";
import IssueOutputPanel from "./IssueOutputPanel.vue";
import IssueSidebar from "./IssueSidebar.vue";
import IssueTaskStatementPanel from "./IssueTaskStatementPanel.vue";
import IssueDescriptionPanel from "./IssueDescriptionPanel.vue";
import IssueActivityPanel from "./IssueActivityPanel.vue";
import PipelineSimpleFlow from "./PipelineSimpleFlow.vue";
import PipelineTenantFlow from "./PipelineTenantFlow.vue";
import PipelineGhostFlow from "./PipelineGhostFlow.vue";
import PipelinePITRFlow from "./PipelinePITRFlow.vue";
import TaskCheckBar from "./TaskCheckBar.vue";
import type {
  Issue,
  IssueCreate,
  Instance,
  Task,
  TaskDatabaseSchemaUpdatePayload,
  TaskCreate,
  MigrationType,
} from "@/types";
import { defaultTemplate, templateForType } from "@/plugins";
import {
  featureToRef,
  useInstanceStore,
  useProjectStore,
  useTaskStore,
} from "@/store";
import {
  provideIssueLogic,
  TenantModeProvider,
  GhostModeProvider,
  StandardModeProvider,
  TaskTypeWithStatement,
  IssueLogic,
  useBaseIssueLogic,
} from "./logic";

const props = defineProps({
  create: {
    type: Boolean,
    required: true,
  },
  issue: {
    type: Object as PropType<Issue | IssueCreate>,
    required: true,
  },
});

const emit = defineEmits<{
  (e: "status-changed", eager: boolean): void;
}>();

const route = useRoute();

const taskStore = useTaskStore();
const projectStore = useProjectStore();

const create = computed(() => props.create);
const issue = computed(() => props.issue);

const {
  project,
  isTenantMode,
  isGhostMode,
  isPITRMode,
  createIssue,
  selectedStage,
  selectedTask,
  selectedDatabase,
  selectStageOrTask,
  selectTask,
  taskStatusOfStage,
  isValidStage,
  allowApplyIssueStatusTransition,
  allowApplyTaskStatusTransition,
} = useBaseIssueLogic({ issue, create });

const issueLogic = ref<IssueLogic>();

// Determine which type of IssueLogicProvider should be used
const logicProviderType = computed(() => {
  if (isTenantMode.value) return TenantModeProvider;
  if (isGhostMode.value) return GhostModeProvider;
  return StandardModeProvider;
});

watchEffect(() => {
  if (props.create) {
    projectStore.fetchProjectById((props.issue as IssueCreate).projectId);
  }
});

const issueTemplate = computed(
  () => templateForType(props.issue.type) || defaultTemplate()
);

const runTaskChecks = (task: Task) => {
  taskStore
    .runChecks({
      issueId: (props.issue as Issue).id,
      pipelineId: (props.issue as Issue).pipeline.id,
      taskId: task.id,
    })
    .then(() => {
      emit("status-changed", true);
    });
};

const currentPipelineType = computed((): PipelineType => {
  return pipelineType(props.issue.pipeline!);
});

const selectedMigrateType = computed((): MigrationType => {
  if (
    !props.create &&
    selectedTask.value.type == "bb.task.database.schema.update"
  ) {
    return (
      (selectedTask.value as Task).payload as TaskDatabaseSchemaUpdatePayload
    ).migrationType;
  }
  return "MIGRATE";
});

const showPipelineFlowBar = computed(() => {
  return currentPipelineType.value !== "NO_PIPELINE";
});

const showIssueOutputPanel = computed(() => {
  return !props.create && issueTemplate.value.outputFieldList.length > 0;
});

const showIssueTaskStatementPanel = computed(() => {
  const task = selectedTask.value;
  return TaskTypeWithStatement.includes(task.type);
});

const instance = computed((): Instance => {
  if (props.create) {
    // If database is available, then we derive the instance from database because we always fetch database's instance.
    if (selectedDatabase.value) {
      return selectedDatabase.value.instance;
    }
    return useInstanceStore().getInstanceById(
      (selectedTask.value as TaskCreate).instanceId
    );
  }
  return (selectedTask.value as Task).instance;
});

const sqlHint = (): string | undefined => {
  if (!props.create && selectedMigrateType.value == "BASELINE") {
    return `This is a baseline migration and Bytebase won't apply the SQL to the database, it will only record a baseline history`;
  }
  if (instance.value.engine === "SNOWFLAKE") {
    return `Use <<schema>>.<<table>> to specify a Snowflake table`;
  }
  return undefined;
};

onMounted(() => {
  // Always scroll to top, the scrollBehavior doesn't seem to work.
  // The hypothesis is that because the scroll bar is in the nested
  // route, thus setting the scrollBehavior in the global router
  // won't work.
  // BUT when we have a location.hash #activity(\d+) we won't scroll to the top,
  // since #activity(\d+) is used as an activity anchor
  if (!location.hash.match(/^#activity(\d+)/)) {
    document.getElementById("issue-detail-top")!.scrollIntoView();
  }
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

watch(
  [create, issue, () => route.query.sql as string, issueLogic],
  ([create, issue, sql, provider]) => {
    // If 'sql' in URL query, update the issueCreate's statement
    // Only works for the first time.
    // E.g. redirected from SQL editor when user wants to execute DML.
    if (create && issue && sql && provider) {
      provider.updateStatement(sql);
    }
  }
);

// When activeTask is changed, we automatically select it.
// This enables users to know the pipeline status has changed and we may move forward.
const autoSelectWhenStatusChanged = () => {
  const activeTask = computed((): Task | undefined => {
    if (create.value) return undefined;
    const { pipeline } = issue.value as Issue;
    if (!pipeline) return undefined;
    const task = activeTaskOfPipeline(pipeline);
    return task;
  });

  watch(
    // Watch the task.id instead of the task object itself, Since the object might
    // sometimes drift to another object reference when polling the issue.
    () => activeTask.value?.id,
    () => {
      const task = activeTask.value;
      if (!task) return;
      selectTask(task);
    },
    // Also triggered when the first time the page is loaded.
    { immediate: true }
  );
};

const onStatusChanged = (eager: boolean) => emit("status-changed", eager);

autoSelectWhenStatusChanged();

provideIssueLogic(
  {
    create,
    issue,
    project,
    template: issueTemplate,
    selectedStage,
    selectedTask,
    selectedDatabase,
    isTenantMode,
    isGhostMode,
    isPITRMode,
    isValidStage,
    taskStatusOfStage,
    activeStageOfPipeline,
    activeTaskOfPipeline,
    activeTaskOfStage: activeTaskInStage,
    selectStageOrTask,
    selectTask,
    onStatusChanged,
    createIssue,
    allowApplyIssueStatusTransition,
    allowApplyTaskStatusTransition,
  },
  true
  // This is the root logic, could be overwritten by other (standard, gh-ost, tenant...) logic providers.
);
</script>
