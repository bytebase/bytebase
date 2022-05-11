<template>
  <div
    id="issue-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <component :is="logicProviderType" ref="issueLogic">
      <IssueBanner v-if="!create" :issue="(issue as Issue)" />

      <!-- Highlight Panel -->
      <div class="bg-white px-4 pb-4">
        <IssueHighlightPanel
          :issue="(issue as Issue)"
          :create="create"
          :allow-edit="allowEditNameAndDescription"
          @update-name="updateName"
        >
          <IssueStatusTransitionButtonGroup
            @change-issue-status="changeIssueStatus"
            @change-task-status="changeTaskStatus"
          />
        </IssueHighlightPanel>
      </div>

      <!-- Remind banner for bb.feature.backward-compatibility -->
      <FeatureAttention
        v-if="
          !hasBackwardCompatibilityFeature &&
          supportBackwardCompatibilityFeature
        "
        custom-class="m-5 mt-0"
        feature="bb.feature.backward-compatibility"
        :description="
          $t('subscription.features.bb-feature-backward-compatibility.desc')
        "
      />

      <!-- Stage Flow Bar -->
      <template v-if="showPipelineFlowBar">
        <template v-if="isTenantMode">
          <PipelineTenantFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else-if="isGhostMode">
          <PipelineGhostFlow v-if="project" class="border-t border-b" />
        </template>
        <template v-else>
          <PipelineSimpleFlow class="border-t border-b" />
        </template>

        <div v-if="!create" class="px-4 py-4 md:flex md:flex-col border-b">
          <IssueStagePanel
            :stage="(selectedStage as Stage)"
            :selected-task="(selectedTask as Task)"
            :is-tenant-mode="isTenantMode"
            :is-ghost-mode="isGhostMode"
          />
        </div>
      </template>

      <!-- Output Panel -->
      <!-- Only render the top border if PipelineFlowBar is not displayed, otherwise it would overlap with the bottom border of that -->
      <div
        v-if="showIssueOutputPanel"
        class="px-2 py-4 md:flex md:flex-col"
        :class="showPipelineFlowBar ? '' : 'lg:border-t'"
      >
        <IssueOutputPanel
          :issue="(issue as Issue)"
          :output-field-list="issueTemplate.outputFieldList"
          :allow-edit="allowEditOutput"
          @update-custom-field="updateCustomField"
        />
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
          <div class="flex flex-col flex-1 lg:flex-row-reverse lg:col-span-2">
            <div
              class="py-6 lg:pl-4 lg:w-96 xl:w-112 lg:border-l lg:border-block-border overflow-hidden"
            >
              <IssueSidebar
                :issue="issue"
                :database="database"
                :instance="instance"
                :create="create"
                :selected-stage="selectedStage"
                :task="selectedTask"
                :input-field-list="issueTemplate.inputFieldList"
                :allow-edit="allowEditSidebar"
                :is-tenant-deploy-mode="isTenantMode"
                @update-assignee-id="updateAssigneeId"
                @update-earliest-allowed-time="updateEarliestAllowedTime"
                @add-subscriber-id="addSubscriberId"
                @remove-subscriber-id="removeSubscriberId"
                @update-custom-field="updateCustomField"
                @select-stage-id="selectStageOrTask"
                @select-task-id="selectTaskId"
              />
            </div>
            <div class="lg:hidden border-t border-block-border" />
            <div class="w-full py-4 pr-4">
              <section v-if="showIssueTaskStatementPanel" class="border-b mb-4">
                <div v-if="!create" class="mb-4">
                  <TaskCheckBar
                    :task="(selectedTask as Task)"
                    @run-checks="runTaskChecks"
                  />
                </div>
                <IssueTaskStatementPanel :sql-hint="sqlHint()" />
              </section>

              <IssueDescriptionPanel
                :issue="issue"
                :create="create"
                :allow-edit="allowEditNameAndDescription"
                @update-description="updateDescription"
              />
              <section
                v-if="!create"
                aria-labelledby="activity-title"
                class="mt-4"
              >
                <IssueActivityPanel
                  :issue="(issue as Issue)"
                  :issue-template="issueTemplate"
                  @add-subscriber-id="addSubscriberId"
                />
              </section>
            </div>
          </div>
        </div>
      </main>
    </component>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import {
  computed,
  nextTick,
  onMounted,
  PropType,
  ref,
  watch,
  watchEffect,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import {
  idFromSlug,
  pipelineType,
  PipelineType,
  indexFromSlug,
  activeStage,
  stageSlug,
  taskSlug,
  isDev,
  activeTaskInStage,
  activeTask,
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
import TaskCheckBar from "./TaskCheckBar.vue";
import type {
  Issue,
  IssueCreate,
  IssuePatch,
  PrincipalId,
  Database,
  Instance,
  Stage,
  StageId,
  IssueStatus,
  TaskId,
  TaskStatusPatch,
  TaskStatus,
  IssueStatusPatch,
  Task,
  TaskDatabaseSchemaUpdatePayload,
  StageCreate,
  TaskCreate,
  Project,
  MigrationType,
  TaskPatch,
} from "@/types";
import {
  defaultTemplate,
  templateForType,
  InputField,
  OutputField,
} from "@/plugins";
import {
  featureToRef,
  useCurrentUser,
  useDatabaseStore,
  useInstanceStore,
  useIssueStore,
  useIssueSubscriberStore,
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

const router = useRouter();
const route = useRoute();

const currentUser = useCurrentUser();
const issueStore = useIssueStore();
const issueSubscriberStore = useIssueSubscriberStore();
const taskStore = useTaskStore();
const projectStore = useProjectStore();
const databaseStore = useDatabaseStore();

const issueLogic = ref<IssueLogic>();

watchEffect(function prepare() {
  if (props.create) {
    projectStore.fetchProjectById((props.issue as IssueCreate).projectId);
  }
});

const issueTemplate = computed(
  () => templateForType(props.issue.type) || defaultTemplate()
);

const project = computed((): Project => {
  if (props.create) {
    return projectStore.getProjectById((props.issue as IssueCreate).projectId);
  }
  return (props.issue as Issue).project;
});

const updateName = (
  newName: string,
  postUpdated: (updatedIssue: Issue) => void
) => {
  if (props.create) {
    props.issue.name = newName;
  } else {
    patchIssue(
      {
        name: newName,
      },
      postUpdated
    );
  }
};

const updateDescription = (
  newDescription: string,
  postUpdated: (updatedIssue: Issue) => void
) => {
  if (props.create) {
    props.issue.description = newDescription;
  } else {
    patchIssue(
      {
        description: newDescription,
      },
      postUpdated
    );
  }
};

const updateAssigneeId = (newAssigneeId: PrincipalId) => {
  if (props.create) {
    (props.issue as IssueCreate).assigneeId = newAssigneeId;
  } else {
    patchIssue({
      assigneeId: newAssigneeId,
    });
  }
};

const updateEarliestAllowedTime = (newEarliestAllowedTsMs: number) => {
  if (props.create) {
    if (isGhostMode.value) {
      // In gh-ost mode, when creating an issue, all sub-tasks in a stage
      // share the same earliestAllowedTs.
      // So updates on any one of them will be applied to others.
      // (They can be updated independently after creation)
      const taskList = selectedStage.value.taskList as TaskCreate[];
      taskList.forEach((task) => {
        task.earliestAllowedTs = newEarliestAllowedTsMs;
      });
    } else {
      selectedTask.value.earliestAllowedTs = newEarliestAllowedTsMs;
    }
  } else {
    const taskPatch: TaskPatch = {
      earliestAllowedTs: newEarliestAllowedTsMs,
    };
    patchTask((selectedTask.value as Task).id, taskPatch);
  }
};

const addSubscriberId = (subscriberId: PrincipalId) => {
  issueSubscriberStore.createSubscriber({
    issueId: (props.issue as Issue).id,
    subscriberId,
  });
};

const removeSubscriberId = (subscriberId: PrincipalId) => {
  issueSubscriberStore.deleteSubscriber({
    issueId: (props.issue as Issue).id,
    subscriberId,
  });
};

const updateCustomField = (field: InputField | OutputField, value: any) => {
  if (!isEqual(props.issue.payload[field.id], value)) {
    if (props.create) {
      props.issue.payload[field.id] = value;
    } else {
      const newPayload = cloneDeep(props.issue.payload);
      newPayload[field.id] = value;
      patchIssue({
        payload: newPayload,
      });
    }
  }
};

const changeIssueStatus = (newStatus: IssueStatus, comment: string) => {
  const issueStatusPatch: IssueStatusPatch = {
    status: newStatus,
    comment: comment,
  };
  issueStore
    .updateIssueStatus({
      issueId: (props.issue as Issue).id,
      issueStatusPatch,
    })
    .then(() => {
      // pollIssue(POST_CHANGE_POLL_INTERVAL);
    });
};

const changeTaskStatus = (
  task: Task,
  newStatus: TaskStatus,
  comment: string
) => {
  // Switch to the stage view containing this task
  selectStageOrTask(task.stage.id);
  nextTick().then(() => {
    selectTaskId(task.id);
  });

  const taskStatusPatch: TaskStatusPatch = {
    status: newStatus,
    comment: comment,
  };
  taskStore
    .updateStatus({
      issueId: (props.issue as Issue).id,
      pipelineId: (props.issue as Issue).pipeline.id,
      taskId: task.id,
      taskStatusPatch,
    })
    .then(() => {
      // pollIssue(POST_CHANGE_POLL_INTERVAL);
      emit("status-changed", true);
    });
};

const runTaskChecks = (task: Task) => {
  taskStore
    .runChecks({
      issueId: (props.issue as Issue).id,
      pipelineId: (props.issue as Issue).pipeline.id,
      taskId: task.id,
    })
    .then(() => {
      // pollIssue(POST_CHANGE_POLL_INTERVAL);
      emit("status-changed", true);
    });
};

const patchIssue = (
  issuePatch: IssuePatch,
  postUpdated?: (updatedIssue: Issue) => void
) => {
  issueStore
    .patchIssue({
      issueId: (props.issue as Issue).id,
      issuePatch,
    })
    .then((updatedIssue) => {
      // issue/patchIssue already fetches the new issue, so we schedule
      // the next poll in NORMAL_POLL_INTERVAL
      // pollIssue(NORMAL_POLL_INTERVAL);
      emit("status-changed", false);
      if (postUpdated) {
        postUpdated(updatedIssue);
      }
    });
};

const patchTask = (
  taskId: TaskId,
  taskPatch: TaskPatch,
  postUpdated?: (updatedTask: Task) => void
) => {
  taskStore
    .patchTask({
      issueId: (props.issue as Issue).id,
      pipelineId: (props.issue as Issue).pipeline.id,
      taskId,
      taskPatch,
    })
    .then((updatedTask) => {
      // For now, the only task/patchTask is to change statement, which will trigger async task check.
      // Thus we use the short poll interval
      emit("status-changed", true);
      if (postUpdated) {
        postUpdated(updatedTask);
      }
    });
};

const currentPipelineType = computed((): PipelineType => {
  return pipelineType(props.issue.pipeline!);
});

const selectedStage = computed((): Stage | StageCreate => {
  const stageSlug = router.currentRoute.value.query.stage as string;
  const taskSlug = router.currentRoute.value.query.task as string;
  // For stage slug, we support both index based and id based.
  // Index based is used when creating the new task and is the one used when clicking the UI.
  // Id based is used when the context only has access to the stage id (e.g. Task only contains StageId)
  if (stageSlug) {
    const index = indexFromSlug(stageSlug);
    if (index < props.issue.pipeline!.stageList.length) {
      return props.issue.pipeline!.stageList[index];
    }
    const stageId = idFromSlug(stageSlug);
    const stageList = (props.issue as Issue).pipeline.stageList;
    for (const stage of stageList) {
      if (stage.id == stageId) {
        return stage;
      }
    }
  } else if (!props.create && taskSlug) {
    const taskId = idFromSlug(taskSlug);
    const stageList = (props.issue as Issue).pipeline.stageList;
    for (const stage of stageList) {
      for (const task of stage.taskList) {
        if (task.id == taskId) {
          return stage;
        }
      }
    }
  }
  if (props.create) {
    return props.issue.pipeline!.stageList[0];
  }
  return activeStage((props.issue as Issue).pipeline);
});

const selectStageOrTask = (
  stageId: StageId,
  taskSlug: string | undefined = undefined
) => {
  const stageList = props.issue.pipeline!.stageList;
  const index = stageList.findIndex((item, index) => {
    if (props.create) {
      return index === stageId;
    }
    return (item as Stage).id == stageId;
  });
  router.replace({
    name: "workspace.issue.detail",
    query: {
      ...router.currentRoute.value.query,
      stage: stageSlug(stageList[index].name, index),
      task: taskSlug,
    },
  });
};

const selectTaskId = (taskId: TaskId) => {
  const taskList = selectedStage.value.taskList as Task[];
  const task = taskList.find((t) => t.id === taskId);
  if (!task) return;
  const slug = taskSlug(task.name, task.id);
  const stage = selectedStage.value as Stage;
  selectStageOrTask(stage.id, slug);
};

const selectedTask = computed((): Task | TaskCreate => {
  const taskSlug = route.query.task as string;
  const { taskList } = selectedStage.value;
  if (taskSlug) {
    const index = indexFromSlug(taskSlug);
    if (index < taskList.length) {
      return taskList[index];
    }
    const id = idFromSlug(taskSlug);
    for (let i = 0; i < taskList.length; i++) {
      const task = taskList[i] as Task;
      if (task.id === id) {
        return task;
      }
    }
  }
  return taskList[0];
});

const isTenantMode = computed((): boolean => {
  if (project.value.tenantMode !== "TENANT") return false;
  return (
    props.issue.type === "bb.issue.database.schema.update" ||
    props.issue.type === "bb.issue.database.data.update"
  );
});

const isGhostMode = computed((): boolean => {
  if (!isDev()) return false;

  return props.issue.type === "bb.issue.database.schema.update.ghost";
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

const allowEditSidebar = computed(() => {
  // For now, we only allow assignee to update the field when the issue
  // is 'OPEN'. This reduces flexibility as creator must ask assignee to
  // change any fields if there is typo. On the other hand, this avoids
  // the trouble that the creator changes field value when the creator
  // is performing the issue based on the old value.
  // For now, we choose to be on the safe side at the cost of flexibility.
  return (
    props.create ||
    ((props.issue as Issue).status == "OPEN" &&
      (props.issue as Issue).assignee?.id == currentUser.value.id)
  );
});

const allowEditOutput = computed(() => {
  return (
    props.create ||
    ((props.issue as Issue).status == "OPEN" &&
      (props.issue as Issue).assignee?.id == currentUser.value.id)
  );
});

const allowEditNameAndDescription = computed(() => {
  return (
    props.create ||
    ((props.issue as Issue).status == "OPEN" &&
      ((props.issue as Issue).assignee?.id == currentUser.value.id ||
        (props.issue as Issue).creator.id == currentUser.value.id))
  );
});

const showPipelineFlowBar = computed(() => {
  return currentPipelineType.value != "NO_PIPELINE";
});

const showIssueOutputPanel = computed(() => {
  return !props.create && issueTemplate.value.outputFieldList.length > 0;
});

const showIssueTaskStatementPanel = computed(() => {
  const task = selectedTask.value;
  return TaskTypeWithStatement.includes(task.type);
});

const database = computed((): Database | undefined => {
  if (props.create) {
    const databaseId = (selectedTask.value as TaskCreate).databaseId;
    if (databaseId) {
      return databaseStore.getDatabaseById(databaseId);
    }
    return undefined;
  }
  return (selectedTask.value as Task).database;
});

const instance = computed((): Instance => {
  if (props.create) {
    // If database is available, then we derive the instance from database because we always fetch database's instance.
    if (database.value) {
      return database.value.instance;
    }
    return useInstanceStore().getInstanceById(
      (selectedTask.value as TaskCreate).instanceId
    );
  }
  return (selectedTask.value as Task).instance;
});

const sqlHint = (): string | undefined => {
  if (!props.create && selectedMigrateType.value == "BASELINE") {
    return `This is a baseline migration and bytebase won't apply the SQL to the database, it will only record a baseline history`;
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
  document.getElementById("issue-detail-top")!.scrollIntoView();
});

const hasBackwardCompatibilityFeature = featureToRef(
  "bb.feature.backward-compatibility"
);

const supportBackwardCompatibilityFeature = computed((): boolean => {
  const engine = database.value?.instance.engine;
  return engine === "MYSQL" || engine === "TIDB";
});

const taskStatusOfStage = (stage: Stage | StageCreate) => {
  if (props.create) {
    return stage.taskList[0].status;
  }
  const activeTask = activeTaskInStage(stage as Stage);
  return activeTask.status;
};

const isValidStage = (stage: Stage | StageCreate) => {
  if (!props.create) {
    return true;
  }

  for (const task of stage.taskList) {
    if (TaskTypeWithStatement.includes(task.type)) {
      if (isEmpty((task as TaskCreate).statement)) {
        return false;
      }
    }
  }
  return true;
};

const logicProviderType = computed(() => {
  if (isTenantMode.value) return TenantModeProvider;
  if (isGhostMode.value) return GhostModeProvider;
  return StandardModeProvider;
});

const create = computed(() => props.create);
const issue = computed(() => props.issue);

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

provideIssueLogic(
  {
    create,
    issue,
    project: project,
    template: issueTemplate,
    selectedStage,
    selectedTask,
    isTenantMode,
    isGhostMode,
    isValidStage,
    taskStatusOfStage,
    activeStageOfPipeline: activeStage,
    activeTaskOfPipeline: activeTask,
    activeTaskOfStage: activeTaskInStage,
    selectStageOrTask: selectStageOrTask,
    patchTask,
    patchIssue,
  },
  true
  // This is the root logic, could be overwritten by other (standard, gh-ost, tenant...) logic providers.
);
</script>
