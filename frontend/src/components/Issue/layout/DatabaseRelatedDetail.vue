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
          <template #tips>
            <IssueRollbackFromTips />
          </template>
        </IssueHighlightPanel>
      </div>

      <!-- Stage Flow Bar -->
      <template v-if="showPipelineFlowBar">
        <template v-if="isGhostMode">
          <PipelineGhostFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else-if="isTenantMode">
          <PipelineTenantFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else-if="isPITRMode">
          <PipelinePITRFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else>
          <PipelineSimpleFlow class="border-t border-b" />
        </template>

        <IssueStagePanel
          v-if="!create"
          class="px-4 py-4 md:flex md:flex-col border-b"
        />
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
              <section v-if="showIssueTaskSDLPanel" class="mb-4">
                <IssueTaskSDLPanel />
              </section>
              <section v-if="showIssueTaskStatementPanel" class="border-b mb-4">
                <IssueTaskStatementPanel :sql-hint="sqlHint()" />
              </section>

              <IssueDescriptionPanel />

              <section
                v-if="!create"
                aria-labelledby="activity-title"
                class="mt-4"
              >
                <IssueActivityPanel />
              </section>
            </div>
          </div>
        </div>
      </main>
    </div>
  </component>
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { computed, onMounted, PropType, ref, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { defaultTemplate, templateForType } from "@/plugins";
import { useInstanceV1Store, useProjectV1Store, useTaskStore } from "@/store";
import type {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  MigrationType,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  pipelineType,
  PipelineType,
  activeStage as activeStageOfPipeline,
  activeTaskInStage,
  activeTask as activeTaskOfPipeline,
} from "@/utils";
import IssueActivityPanel from "../IssueActivityPanel.vue";
import IssueBanner from "../IssueBanner.vue";
import IssueDescriptionPanel from "../IssueDescriptionPanel.vue";
import IssueHighlightPanel from "../IssueHighlightPanel.vue";
import IssueOutputPanel from "../IssueOutputPanel.vue";
import IssueRollbackFromTips from "../IssueRollbackFromTips.vue";
import IssueSidebar from "../IssueSidebar.vue";
import IssueStagePanel from "../IssueStagePanel.vue";
import IssueTaskSDLPanel from "../IssueTaskSDLPanel.vue";
import IssueTaskStatementPanel from "../IssueTaskStatementPanel.vue";
import PipelineGhostFlow from "../PipelineGhostFlow.vue";
import PipelinePITRFlow from "../PipelinePITRFlow.vue";
import PipelineSimpleFlow from "../PipelineSimpleFlow.vue";
import PipelineTenantFlow from "../PipelineTenantFlow.vue";
import TaskCheckBar from "../TaskCheckBar.vue";
import {
  provideIssueLogic,
  TenantModeProvider,
  GhostModeProvider,
  StandardModeProvider,
  TaskTypeWithStatement,
  IssueLogic,
  useBaseIssueLogic,
} from "../logic";

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

const { t } = useI18n();

const route = useRoute();

const taskStore = useTaskStore();
const projectV1Store = useProjectV1Store();

const create = computed(() => props.create);
const issue = computed(() => props.issue);

const dialog = useDialog();

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
  selectedStatement,
  allowApplyTaskStateToOthers,
  applyTaskStateToOthers,
} = useBaseIssueLogic({ issue, create });

const issueLogic = ref<IssueLogic>();

// Determine which type of IssueLogicProvider should be used
const logicProviderType = computed(() => {
  if (isGhostMode.value) return GhostModeProvider;
  if (isTenantMode.value) return TenantModeProvider;
  return StandardModeProvider;
});

watchEffect(() => {
  if (props.create) {
    projectV1Store.getOrFetchProjectByUID(
      String((props.issue as IssueCreate).projectId)
    );
  }
});

const issueTemplate = computed(
  () => templateForType(props.issue.type) || defaultTemplate()
);

const runTaskChecks = (taskList: Task[]) => {
  const requests = taskList.map((task) => {
    return taskStore.runChecks({
      issueId: (props.issue as Issue).id,
      pipelineId: (props.issue as Issue).pipeline!.id,
      taskId: task.id,
    });
  });
  Promise.allSettled(requests).then(() => {
    emit("status-changed", true);
  });
};

const currentPipelineType = computed((): PipelineType => {
  return pipelineType(props.issue.pipeline!);
});

const selectedMigrateType = computed((): MigrationType => {
  const taskType = selectedTask.value.type;
  if (taskType === "bb.task.database.schema.baseline") {
    // The new version of BASELINE tasks
    return "BASELINE";
  }
  if (!props.create && taskType === "bb.task.database.schema.update") {
    // Legacy BASELINE task support
    // Their `type`s are "bb.task.database.schema.update"
    // And their `payload.migrationType`s are "BASELINE"
    const { payload } = selectedTask.value as Task;
    if ((payload as any).migrationType === "BASELINE") {
      return "BASELINE";
    }
  }
  if (taskType === "bb.task.database.data.update") {
    // A "Change data" task
    return "DATA";
  }
  // Fall back to MIGRATE otherwise
  return "MIGRATE";
});

const showPipelineFlowBar = computed(() => {
  return currentPipelineType.value !== "NO_PIPELINE";
});

const showIssueOutputPanel = computed(() => {
  return !props.create && issueTemplate.value.outputFieldList.length > 0;
});

const showIssueTaskSDLPanel = computed(() => {
  if (create.value) return false;
  const task = selectedTask.value as Task;
  return task.type === "bb.task.database.schema.update-sdl";
});

const showIssueTaskStatementPanel = computed(() => {
  if (showIssueTaskSDLPanel.value) return false;

  const task = selectedTask.value;
  if (task.type === "bb.task.database.schema.baseline" && !create.value) {
    return false;
  }
  return TaskTypeWithStatement.includes(task.type);
});

const instance = computed(() => {
  if (props.create) {
    // If database is available, then we derive the instance from database because we always fetch database's instance.
    if (selectedDatabase.value) {
      return selectedDatabase.value.instanceEntity;
    }
    return useInstanceV1Store().getInstanceByUID(
      String((selectedTask.value as TaskCreate).instanceId)
    );
  }

  return useInstanceV1Store().getInstanceByUID(
    String((selectedTask.value as Task).instance.id)
  );
});

const sqlHint = (): string | undefined => {
  if (selectedMigrateType.value === "BASELINE") {
    return t("issue.sql-hint.dont-apply-to-database-in-baseline-migration");
  }
  if (instance.value.engine === Engine.SNOWFLAKE) {
    return t("issue.sql-hint.snowflake");
  }
  if (isTenantMode.value) {
    return t("issue.sql-hint.change-will-apply-to-all-tasks-in-tenant-mode");
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

watch(
  [
    create,
    issue,
    () => route.query.sqlList as string,
    () => route.query.sql as string,
    issueLogic,
  ],
  ([create, issue, sqlList, sql, provider]) => {
    if (create && issue && provider) {
      if (sql) {
        // If 'sql' in URL query, update the issueCreate's statement
        // Only works for the first time.
        // E.g. redirected from SQL editor when user wants to execute DML.
        provider.updateStatement(sql);
      } else {
        // Initial the tasks's statement/sheetId in issueCreate from current route.
        provider.initialTaskListStatementFromRoute();
      }
    }
  }
);

// When activeTask is changed, we automatically select it.
// This enables users to know the pipeline status has changed and we may move forward.
const autoSelectWhenStatusChanged = (immediate: boolean) => {
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
    { immediate }
  );
};

const onStatusChanged = (eager: boolean) => emit("status-changed", eager);

if (route.query.stage || route.query.task) {
  // If we have selected stage/task in URL, don't switch to activeTask immediately.
  autoSelectWhenStatusChanged(false);
} else {
  // Otherwise, automatically switch to the active task immediately.
  autoSelectWhenStatusChanged(true);
}

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
    selectedStatement,
    allowApplyTaskStateToOthers,
    applyTaskStateToOthers,
    dialog,
  },
  true
  // This is the root logic, could be overwritten by other (standard, gh-ost, tenant...) logic providers.
);
</script>
