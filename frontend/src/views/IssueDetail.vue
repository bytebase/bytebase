<template>
  <div
    id="issue-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <IssueBanner v-if="!state.create" :issue="issue" />

    <!-- Highlight Panel -->
    <div class="bg-white px-4 pb-4">
      <IssueHighlightPanel
        :issue="issue"
        :create="state.create"
        :allow-edit="allowEditNameAndDescription"
        @update-name="updateName"
      >
        <IssueStatusTransitionButtonGroup
          :create="state.create"
          :allow-rollback="allowRollback"
          :issue="issue"
          :issue-template="issueTemplate"
          @create="doCreate"
          @rollback="doRollback"
          @change-issue-status="changeIssueStatus"
          @change-task-status="changeTaskStatus"
        />
      </IssueHighlightPanel>
    </div>

    <!-- Stage Flow Bar -->
    <template v-if="showPipelineFlowBar">
      <PipelineSimpleFlow
        :create="state.create"
        :pipeline="issue.pipeline"
        :selected-stage="selectedStage"
        @select-stage-id="selectStageId"
      />
      <div v-if="!state.create" class="px-4 py-4 md:flex md:flex-col border-b">
        <IssueStagePanel :stage="selectedStage" />
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
        :issue="issue"
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
            class="py-6 lg:pl-4 lg:w-96 xl:w-112 lg:border-l lg:border-block-border"
          >
            <IssueSidebar
              :issue="issue"
              :task="selectedTask"
              :database="database"
              :instance="instance"
              :create="state.create"
              :selected-stage="selectedStage"
              :input-field-list="issueTemplate.inputFieldList"
              :allow-edit="allowEditSidebar"
              @update-assignee-id="updateAssigneeId"
              @update-earliest-allowed-time="updateEarliestAllowedTime"
              @add-subscriber-id="addSubscriberId"
              @remove-subscriber-id="removeSubscriberId"
              @update-custom-field="updateCustomField"
              @select-stage-id="selectStageId"
            />
          </div>
          <div class="lg:hidden border-t border-block-border" />
          <div class="w-full py-4 pr-4">
            <section v-if="showIssueTaskStatementPanel" class="border-b mb-4">
              <div v-if="!state.create" class="mb-4">
                <TaskCheckBar
                  :task="selectedTask"
                  @run-checks="runTaskChecks"
                />
              </div>
              <!-- The way this is written is awkward and is to workaround an issue in IssueTaskStatementPanel.
                   The statement panel is in non-edit mode when not creating the issue, and we use v-highlight
                   to apply syntax highlighting when the panel is in non-edit mode. However, the v-highlight
                   doesn't seem to work well with the reactivity. So for non-edit mode when !state.create, we
                   list every IssueTaskStatementPanel for each stage and use v-if to show the active one. -->
              <template v-if="state.create">
                <IssueTaskStatementPanel
                  :sql-hint="sqlHint(false)"
                  :statement="selectedStatement"
                  :create="state.create"
                  :allow-edit="true"
                  :rollback="false"
                  :show-apply-statement="showIssueTaskStatementApply"
                  @update-statement="updateStatement"
                  @apply-statement-to-other-stages="applyStatementToOtherStages"
                />
              </template>
              <template
                v-for="(stage, index) in issue.pipeline.stageList"
                v-else
                :key="index"
              >
                <template v-if="selectedStage.id == stage.id">
                  <IssueTaskStatementPanel
                    :sql-hint="sqlHint(false)"
                    :statement="statement(stage)"
                    :create="state.create"
                    :allow-edit="allowEditStatement"
                    :rollback="false"
                    :show-apply-statement="showIssueTaskStatementApply"
                    @update-statement="updateStatement"
                  />
                </template>
              </template>
            </section>
            <section
              v-if="showIssueTaskRollbackStatementPanel"
              class="border-b mb-4"
            >
              <template v-if="state.create">
                <IssueTaskStatementPanel
                  :sql-hint="sqlHint(true)"
                  :statement="selectedRollbackStatement"
                  :create="state.create"
                  :allow-edit="false"
                  :rollback="true"
                  :show-apply-statement="showIssueTaskStatementApply"
                  @update-statement="updateRollbackStatement"
                  @apply-statement-to-other-stages="
                    applyRollbackStatementToOtherStages
                  "
                />
              </template>
              <template
                v-for="(stage, index) in issue.pipeline.stageList"
                v-else
                :key="index"
              >
                <template v-if="selectedStage.id == stage.id">
                  <IssueTaskStatementPanel
                    :sql-hint="sqlHint(true)"
                    :statement="rollbackStatement(stage)"
                    :create="state.create"
                    :allow-edit="false"
                    :rollback="true"
                    :show-apply-statement="showIssueTaskStatementApply"
                    @update-statement="updateRollbackStatement"
                  />
                </template>
              </template>
            </section>
            <IssueDescriptionPanel
              :issue="issue"
              :create="state.create"
              :allow-edit="allowEditNameAndDescription"
              @update-description="updateDescription"
            />
            <section
              v-if="!state.create"
              aria-labelledby="activity-title"
              class="mt-4"
            >
              <IssueActivityPanel
                :issue="issue"
                :issue-template="issueTemplate"
                @add-subscriber-id="addSubscriberId"
              />
            </section>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
import {
  computed,
  onMounted,
  onUnmounted,
  watch,
  watchEffect,
  reactive,
  ref,
  defineProps,
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import {
  idFromSlug,
  issueSlug,
  pipelineType,
  PipelineType,
  indexFromSlug,
  activeStage,
  stageSlug,
  activeTask,
} from "../utils";
import {
  IssueBanner,
  IssueHighlightPanel,
  IssueStagePanel,
  IssueStatusTransitionButtonGroup,
  IssueOutputPanel,
  IssueSidebar,
  IssueTaskStatementPanel,
  IssueDescriptionPanel,
  IssueActivityPanel,
  PipelineSimpleFlow,
  TaskCheckBar,
} from "../components/Issue";
import {
  UNKNOWN_ID,
  Issue,
  IssueCreate,
  IssueType,
  IssuePatch,
  PrincipalId,
  Database,
  Instance,
  Environment,
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
  TaskDatabaseCreatePayload,
  TaskGeneralPayload,
  NORMAL_POLL_INTERVAL,
  POLL_JITTER,
  POST_CHANGE_POLL_INTERVAL,
  Project,
  MigrationType,
  TaskPatch,
  Policy,
} from "../types";
import {
  defaulTemplate,
  templateForType,
  InputField,
  OutputField,
  IssueTemplate,
} from "../plugins";
import { isEmpty } from "lodash-es";

interface LocalState {
  // Needs to maintain this state and set it to false manually after creating the issue.
  // router.push won't trigger the reload because new and existing issue shares
  // the same component.
  create: boolean;
  newIssue?: IssueCreate;
  // Timer tracking the issue poller, we need this to cancel the outstanding one when needed.
  pollIssueTimer?: ReturnType<typeof setTimeout>;
}

const props = defineProps<{ issueSlug: string }>();

const store = useStore();
const router = useRouter();

const currentUser = computed(() => store.getters["auth/currentUser"]());

let newIssueTemplate = ref<IssueTemplate>(defaulTemplate());

const refreshTemplate = () => {
  const issueType = router.currentRoute.value.query.template as IssueType;
  if (issueType) {
    const template = templateForType(issueType);
    if (template) {
      newIssueTemplate.value = template;
    } else {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "WARN",
        title: `Unknown template '${issueType}'.`,
        description: "Fallback to the default template",
      });
    }
  }

  if (!newIssueTemplate.value) {
    newIssueTemplate.value = defaulTemplate();
  }
};

// Vue doesn't natively react to query parameter change
// so we need to manually watch here.
watch(
  () => router.currentRoute.value.query.template,
  () => {
    refreshTemplate();
  }
);

watchEffect(refreshTemplate);

const buildNewIssue = (): IssueCreate => {
  var newIssue: IssueCreate;

  // Create rollback issue
  if (router.currentRoute.value.query.rollbackIssue) {
    const rollbackIssue: Issue = store.getters["issue/issueById"](
      parseInt(router.currentRoute.value.query.rollbackIssue as string)
    );

    let validState = false;
    let title = "";

    // We expect user to create rollback from the original issue, which should have already fetched
    // issue remotely. Otherwise, this will return UNKNOWN_ID
    if (rollbackIssue.id == UNKNOWN_ID) {
      title = "INVALID STATE, please create rollback from the original issue.";
    } else {
      if (rollbackIssue.type != "bb.issue.database.schema.update") {
        title = "INVALID STATE, only support to rollback update schema issue.";
      } else if (
        rollbackIssue.status != "DONE" &&
        rollbackIssue.status != "CANCELED"
      ) {
        title =
          "INVALID STATE, only support to rollback issue in closed state.";
      } else {
        for (const stage of rollbackIssue.pipeline.stageList) {
          for (const task of stage.taskList) {
            if (
              task.status == "DONE" &&
              task.type == "bb.task.database.schema.update" &&
              !isEmpty(
                (task.payload as TaskDatabaseSchemaUpdatePayload)
                  .rollbackStatement
              )
            ) {
              validState = true;
              break;
            }
          }
          if (validState) {
            break;
          }
        }

        if (!validState) {
          title =
            "INVALID STATE, no applicable update schema task to rollback.";
        }
      }
    }

    if (validState) {
      let environmentList: Environment[] = [];
      const approvalPolicyList: Policy[] = [];
      const databaseList: Database[] = [];
      const statementList: string[] = [];
      const rollbackStatementList: string[] = [];
      for (let i = rollbackIssue.pipeline.stageList.length - 1; i >= 0; i--) {
        const stage = rollbackIssue.pipeline.stageList[i];
        for (let j = stage.taskList.length - 1; j >= 0; j--) {
          const task = stage.taskList[j];
          if (
            task.status == "DONE" &&
            task.type == "bb.task.database.schema.update" &&
            !isEmpty(
              (task.payload as TaskDatabaseSchemaUpdatePayload)
                .rollbackStatement
            )
          ) {
            environmentList.push(stage.environment);
            approvalPolicyList.push(
              store.getters["policy/policyByEnvironmentIdAndType"](
                stage.environment.id,
                "bb.policy.pipeline-approval"
              )
            );
            databaseList.push(task.database!);
            statementList.push(
              (task.payload as TaskDatabaseSchemaUpdatePayload)
                .rollbackStatement
            );
            rollbackStatementList.push(
              (task.payload as TaskDatabaseSchemaUpdatePayload).statement
            );
          }
        }
      }

      if (environmentList.length == 0) {
        newIssue = {
          ...defaulTemplate().buildIssue({
            environmentList: [],
            approvalPolicyList: [],
            databaseList: [],
            currentUser: currentUser.value,
          }),
          name: "INVALID STATE, no applicable update schema task to rollback.",
          projectId: UNKNOWN_ID,
        };
      } else {
        newIssue = {
          ...newIssueTemplate.value.buildIssue({
            environmentList,
            approvalPolicyList,
            databaseList,
            statementList,
            rollbackStatementList,
            currentUser: currentUser.value,
          }),
          projectId: rollbackIssue.project.id,
          name: `[Rollback] issue/${rollbackIssue.id} - ${rollbackIssue.name}`,
          description: rollbackIssue.description
            ? `====Original issue description BEGIN====\n\n${rollbackIssue.description}\n\n====Original issue description END====\n\n`
            : "",
          assigneeId: rollbackIssue.assignee.id,
        };
      }
    } else {
      newIssue = {
        ...defaulTemplate().buildIssue({
          environmentList: [],
          approvalPolicyList: [],
          databaseList: [],
          currentUser: currentUser.value,
        }),
        name: title,
        projectId: UNKNOWN_ID,
      };
    }
    newIssue.rollbackIssueId = rollbackIssue.id;
  }
  // Create issue from normal query parameter
  else {
    const databaseList: Database[] = [];
    if (router.currentRoute.value.query.databaseList) {
      for (const databaseId of (
        router.currentRoute.value.query.databaseList as string
      ).split(","))
        databaseList.push(store.getters["database/databaseById"](databaseId));
    }

    const environmentList: Environment[] = [];
    const approvalPolicyList: Policy[] = [];
    if (router.currentRoute.value.query.environment) {
      environmentList.push(
        store.getters["environment/environmentById"](
          router.currentRoute.value.query.environment
        )
      );
    } else if (databaseList.length > 0) {
      for (const database of databaseList) {
        environmentList.push(database.instance.environment);
      }
    } else {
      environmentList.push(...store.getters["environment/environmentList"]());
    }

    for (const environment of environmentList) {
      approvalPolicyList.push(
        store.getters["policy/policyByEnvironmentIdAndType"](
          environment.id,
          "bb.policy.pipeline-approval"
        )
      );
    }

    newIssue = {
      ...newIssueTemplate.value.buildIssue({
        environmentList,
        approvalPolicyList,
        databaseList,
        currentUser: currentUser.value,
      }),
      projectId: router.currentRoute.value.query.project
        ? parseInt(router.currentRoute.value.query.project as string)
        : UNKNOWN_ID,
    };

    // For demo mode, we assign the issue to the current user, so it can also experience the assignee user flow.
    if (store.getters["actuator/isDemo"]()) {
      newIssue.assigneeId = currentUser.value.id;
    }

    if (router.currentRoute.value.query.name) {
      newIssue.name = router.currentRoute.value.query.name as string;
    }
    if (router.currentRoute.value.query.description) {
      newIssue.description = router.currentRoute.value.query
        .description as string;
    }
    if (router.currentRoute.value.query.assignee) {
      newIssue.assigneeId = parseInt(
        router.currentRoute.value.query.assignee as string
      );
    }
  }
  for (const field of newIssueTemplate.value.inputFieldList) {
    const value = router.currentRoute.value.query[field.slug] as string;
    if (value) {
      if (field.type == "Boolean") {
        newIssue.payload[field.id] =
          value != "0" && value.toLowerCase() != "false";
      } else {
        newIssue.payload[field.id] = value;
      }
    }
  }

  return newIssue;
};

const state = reactive<LocalState>({
  create: props.issueSlug.toLowerCase() == "new",
  newIssue:
    props.issueSlug.toLowerCase() == "new" ? buildNewIssue() : undefined,
});

// pollIssue invalidates the current timer and schedule a new timer in <<interval>> microseconds
const pollIssue = (interval: number) => {
  if (state.pollIssueTimer) {
    clearInterval(state.pollIssueTimer);
  }

  state.pollIssueTimer = setTimeout(() => {
    store.dispatch("issue/fetchIssueById", idFromSlug(props.issueSlug));
    pollIssue(Math.min(interval * 2, NORMAL_POLL_INTERVAL));
  }, Math.max(1000, Math.min(interval, NORMAL_POLL_INTERVAL) + (Math.random() * 2 - 1) * POLL_JITTER));
};

const pollOnCreateStateChange = () => {
  let interval = NORMAL_POLL_INTERVAL;
  // We will poll faster if meets either of the condition
  // 1. Created the database create issue, expect creation result quickly.
  // 2. Update the database schema, will do connection and syntax check.
  if (
    (issue.value.type == "bb.issue.database.create" ||
      issue.value.type == "bb.issue.database.schema.update") &&
    Date.now() - (issue.value as Issue).updatedTs * 1000 < 5000
  ) {
    interval = POST_CHANGE_POLL_INTERVAL;
  }
  pollIssue(interval);
};

onMounted(() => {
  // Always scroll to top, the scrollBehavior doesn't seem to work.
  // The hypothesis is that because the scroll bar is in the nested
  // route, thus setting the scrollBehavior in the global router
  // won't work.
  document.getElementById("issue-detail-top")!.scrollIntoView();
  if (!state.create) {
    pollOnCreateStateChange();
  }
});

onUnmounted(() => {
  if (state.pollIssueTimer) {
    clearInterval(state.pollIssueTimer);
  }
});

watch(
  () => props.issueSlug,
  (cur) => {
    const oldCreate = state.create;
    state.create = cur.toLowerCase() == "new";
    if (!state.create && oldCreate) {
      pollOnCreateStateChange();
    } else if (state.create && !oldCreate) {
      clearInterval(state.pollIssueTimer!);
      state.newIssue = buildNewIssue();
    }
  }
);

const issue = computed((): Issue | IssueCreate => {
  return state.create
    ? state.newIssue
    : store.getters["issue/issueById"](idFromSlug(props.issueSlug));
});

const issueTemplate = computed(
  () => templateForType(issue.value.type) || defaulTemplate()
);

const project = computed((): Project => {
  if (state.create) {
    return store.getters["project/projectById"](
      (issue.value as IssueCreate).projectId
    );
  }
  return (issue.value as Issue).project;
});

const updateName = (
  newName: string,
  postUpdated: (updatedIssue: Issue) => void
) => {
  if (state.create) {
    state.newIssue!.name = newName;
  } else {
    patchIssue(
      {
        name: newName,
      },
      postUpdated
    );
  }
};

const updateStatement = (
  newStatement: string,
  postUpdated?: (updatedTask: Task) => void
) => {
  if (state.create) {
    const stage = selectedStage.value as StageCreate;
    stage.taskList[0].statement = newStatement;
  } else {
    patchTask(
      (selectedTask.value as Task).id,
      {
        statement: newStatement,
      },
      postUpdated
    );
  }
};

const applyStatementToOtherStages = (newStatement: string) => {
  for (const stage of (issue.value as IssueCreate).pipeline.stageList) {
    for (const task of stage.taskList) {
      if (
        task.type == "bb.task.general" ||
        task.type == "bb.task.database.create" ||
        task.type == "bb.task.database.schema.update"
      ) {
        task.statement = newStatement;
      }
    }
  }
};

const updateRollbackStatement = (newStatement: string) => {
  const stage = selectedStage.value as StageCreate;
  stage.taskList[0].rollbackStatement = newStatement;
};

const applyRollbackStatementToOtherStages = (newStatement: string) => {
  for (const stage of (issue.value as IssueCreate).pipeline.stageList) {
    for (const task of stage.taskList) {
      if (task.type == "bb.task.database.schema.update") {
        task.rollbackStatement = newStatement;
      }
    }
  }
};

const updateDescription = (
  newDescription: string,
  postUpdated: (updatedIssue: Issue) => void
) => {
  if (state.create) {
    state.newIssue!.description = newDescription;
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
  if (state.create) {
    state.newIssue!.assigneeId = newAssigneeId;
  } else {
    patchIssue({
      assigneeId: newAssigneeId,
    });
  }
};

const updateEarliestAllowedTime = (newEarliestAllowedTsMs: number) => {
  if (state.create) {
    selectedTask.value.earliestAllowedTs = newEarliestAllowedTsMs;
  } else {
    const taskPatch: TaskPatch = {
      earliestAllowedTs: newEarliestAllowedTsMs,
    };
    patchTask((selectedTask.value as Task).id, taskPatch);
  }
};

const addSubscriberId = (subscriberId: PrincipalId) => {
  store.dispatch("issueSubscriber/createSubscriber", {
    issueId: (issue.value as Issue).id,
    subscriberId,
  });
};

const removeSubscriberId = (subscriberId: PrincipalId) => {
  store.dispatch("issueSubscriber/deleteSubscriber", {
    issueId: (issue.value as Issue).id,
    subscriberId,
  });
};

const updateCustomField = (field: InputField | OutputField, value: any) => {
  if (!isEqual(issue.value.payload[field.id], value)) {
    if (state.create) {
      state.newIssue!.payload[field.id] = value;
    } else {
      const newPayload = cloneDeep(issue.value.payload);
      newPayload[field.id] = value;
      patchIssue({
        payload: newPayload,
      });
    }
  }
};

const doCreate = () => {
  store.dispatch("issue/createIssue", state.newIssue).then((createdIssue) => {
    // Use replace to omit the new issue url in the navigation history.
    router.replace(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
  });
};

const doRollback = () => {
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.update",
      rollbackIssue: (issue.value as Issue).id,
    },
  });
};

const changeIssueStatus = (newStatus: IssueStatus, comment: string) => {
  const issueStatusPatch: IssueStatusPatch = {
    status: newStatus,
    comment: comment,
  };

  store
    .dispatch("issue/updateIssueStatus", {
      issueId: (issue.value as Issue).id,
      issueStatusPatch,
    })
    .then(() => {
      pollIssue(POST_CHANGE_POLL_INTERVAL);
    });
};

const changeTaskStatus = (
  task: Task,
  newStatus: TaskStatus,
  comment: string
) => {
  // Switch to the stage view containing this task
  selectStageId(task.stage.id);

  const taskStatusPatch: TaskStatusPatch = {
    status: newStatus,
    comment: comment,
  };

  store
    .dispatch("task/updateStatus", {
      issueId: (issue.value as Issue).id,
      pipelineId: (issue.value as Issue).pipeline.id,
      taskId: task.id,
      taskStatusPatch,
    })
    .then(() => {
      pollIssue(POST_CHANGE_POLL_INTERVAL);
    });
};

const runTaskChecks = (task: Task) => {
  store
    .dispatch("task/runChecks", {
      issueId: (issue.value as Issue).id,
      pipelineId: (issue.value as Issue).pipeline.id,
      taskId: task.id,
    })
    .then(() => {
      pollIssue(POST_CHANGE_POLL_INTERVAL);
    });
};

const patchIssue = (
  issuePatch: IssuePatch,
  postUpdated?: (updatedIssue: Issue) => void
) => {
  store
    .dispatch("issue/patchIssue", {
      issueId: (issue.value as Issue).id,
      issuePatch,
    })
    .then((updatedIssue) => {
      // issue/patchIssue already fetches the new issue, so we schedule
      // the next poll in NORMAL_POLL_INTERVAL
      pollIssue(NORMAL_POLL_INTERVAL);
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
  store
    .dispatch("task/patchTask", {
      issueId: (issue.value as Issue).id,
      pipelineId: (issue.value as Issue).pipeline.id,
      taskId,
      taskPatch,
    })
    .then((updatedTask) => {
      // For now, the only task/patchTask is to change statement, which will trigger async task check.
      // Thus we use the short poll interval
      pollIssue(POST_CHANGE_POLL_INTERVAL);
      if (postUpdated) {
        postUpdated(updatedTask);
      }
    });
};

const currentPipelineType = computed((): PipelineType => {
  return pipelineType(issue.value.pipeline);
});

const selectedStage = computed((): Stage | StageCreate => {
  const stageSlug = router.currentRoute.value.query.stage as string;
  const taskSlug = router.currentRoute.value.query.task as string;
  // For stage slug, we support both index based and id based.
  // Index based is used when creating the new task and is the one used when clicking the UI.
  // Id based is used when the context only has access to the stage id (e.g. Task only contains StageId)
  if (stageSlug) {
    const index = indexFromSlug(stageSlug);
    if (index < issue.value.pipeline.stageList.length) {
      return issue.value.pipeline.stageList[index];
    }
    const stageId = idFromSlug(stageSlug);
    const stageList = (issue.value as Issue).pipeline.stageList;
    for (const stage of stageList) {
      if (stage.id == stageId) {
        return stage;
      }
    }
  } else if (!state.create && taskSlug) {
    const taskId = idFromSlug(taskSlug);
    const stageList = (issue.value as Issue).pipeline.stageList;
    for (const stage of stageList) {
      for (const task of stage.taskList) {
        if (task.id == taskId) {
          return stage;
        }
      }
    }
  }
  if (state.create) {
    return issue.value.pipeline.stageList[0];
  }
  return activeStage((issue.value as Issue).pipeline);
});

const selectStageId = (stageId: StageId) => {
  const stageList = issue.value.pipeline.stageList;
  const index = stageList.findIndex((item, index) => {
    if (state.create) {
      return index == stageId;
    }
    return (item as Stage).id == stageId;
  });
  router.replace({
    name: "workspace.issue.detail",
    query: {
      ...router.currentRoute.value.query,
      task: undefined,
      stage: stageSlug(stageList[index].name, index),
    },
  });
};

const selectedTask = computed((): Task | TaskCreate => {
  return selectedStage.value.taskList[0];
});

const statement = (stage: Stage): string => {
  const task = stage.taskList[0];
  switch (task.type) {
    case "bb.task.general":
      return ((task as Task).payload as TaskGeneralPayload).statement || "";
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).statement || ""
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).statement ||
        ""
      );
    case "bb.task.database.restore":
      return "";
  }
};

const rollbackStatement = (stage: Stage): string => {
  const task = stage.taskList[0];
  return (
    (task.payload as TaskDatabaseSchemaUpdatePayload).rollbackStatement || ""
  );
};

const selectedStatement = computed((): string => {
  if (router.currentRoute.value.query.sql) {
    const sql = router.currentRoute.value.query.sql as string;
    updateStatement(sql);
  }

  const task = (selectedStage.value as StageCreate).taskList[0];
  return task.statement;
});

const selectedRollbackStatement = computed((): string => {
  const task = (selectedStage.value as StageCreate).taskList[0];
  return task.rollbackStatement;
});

const selectedMigrateType = computed((): MigrationType => {
  if (
    !state.create &&
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
    state.create ||
    ((issue.value as Issue).status == "OPEN" &&
      (issue.value as Issue).assignee?.id == currentUser.value.id)
  );
});

const allowEditOutput = computed(() => {
  return (
    state.create ||
    ((issue.value as Issue).status == "OPEN" &&
      (issue.value as Issue).assignee?.id == currentUser.value.id)
  );
});

const allowEditNameAndDescription = computed(() => {
  return (
    state.create ||
    ((issue.value as Issue).status == "OPEN" &&
      ((issue.value as Issue).assignee?.id == currentUser.value.id ||
        (issue.value as Issue).creator.id == currentUser.value.id))
  );
});

const allowEditStatement = computed(() => {
  return (
    state.create ||
    ((issue.value as Issue).status == "OPEN" &&
      (issue.value as Issue).creator.id == currentUser.value.id &&
      // Only allow if it's UI workflow
      (issue.value as Issue).project.workflowType == "UI" &&
      (selectedTask.value.status == "PENDING" ||
        selectedTask.value.status == "PENDING_APPROVAL" ||
        selectedTask.value.status == "FAILED"))
  );
});

// For now, we only support rollback for schema update issue when all below conditions met:
// 1. Issue is in DONE or CANCELED state
// 2. There is at least one completed schema update task and the task contains the rollback statement.
const allowRollback = computed(() => {
  if (!state.create) {
    if (issue.value.type == "bb.issue.database.schema.update") {
      if (
        (issue.value as Issue).status == "DONE" ||
        (issue.value as Issue).status == "CANCELED"
      ) {
        for (const stage of (issue.value as Issue).pipeline.stageList) {
          for (const task of stage.taskList) {
            if (
              task.status == "DONE" &&
              task.type == "bb.task.database.schema.update" &&
              !isEmpty(
                (task.payload as TaskDatabaseSchemaUpdatePayload)
                  .rollbackStatement
              )
            ) {
              return true;
            }
          }
        }
      }
    }
  }
  return false;
});

const showCancelBanner = computed(() => {
  return !state.create && (issue.value as Issue).status == "CANCELED";
});

const showSuccessBanner = computed(() => {
  return !state.create && (issue.value as Issue).status == "DONE";
});

const showPendingApproval = computed(() => {
  if (state.create) {
    return false;
  }

  const task = activeTask((issue.value as Issue).pipeline);
  return task.status == "PENDING_APPROVAL";
});

const showPipelineFlowBar = computed(() => {
  return currentPipelineType.value != "NO_PIPELINE";
});

const showIssueOutputPanel = computed(() => {
  return !state.create && issueTemplate.value.outputFieldList.length > 0;
});

const showIssueTaskStatementPanel = computed(() => {
  const task = selectedTask.value;
  return (
    task.type == "bb.task.general" ||
    task.type == "bb.task.database.create" ||
    task.type == "bb.task.database.schema.update"
  );
});

const showIssueTaskRollbackStatementPanel = computed(() => {
  if (project.value.workflowType == "UI") {
    return issue.value.type == "bb.issue.database.schema.update";
  }
  return false;
});

const showIssueTaskStatementApply = computed(() => {
  if (!state.create) {
    return false;
  }
  let count = 0;
  for (const stage of (issue.value as IssueCreate).pipeline.stageList) {
    for (const task of stage.taskList) {
      if (
        task.type == "bb.task.general" ||
        task.type == "bb.task.database.create" ||
        task.type == "bb.task.database.schema.update"
      ) {
        count++;
      }
    }
  }
  return count > 1;
});

const database = computed((): Database | undefined => {
  if (state.create) {
    const databaseId = selectedStage.value.taskList[0].databaseId;
    if (databaseId) {
      return store.getters["database/databaseById"](databaseId);
    }
    return undefined;
  }
  return selectedStage.value.taskList[0].database;
});

const instance = computed((): Instance => {
  if (state.create) {
    // If database is available, then we derive the instance from database because we always fetch database's instance.
    if (database.value) {
      return database.value.instance;
    }
    return store.getters["instance/instanceById"](
      selectedStage.value.taskList[0].instanceId
    );
  }
  return selectedStage.value.taskList[0].instance;
});

const sqlHint = (isRollBack: boolean): string | undefined => {
  if (!isRollBack && !state.create && selectedMigrateType.value == "BASELINE") {
    return `This is a baseline migration and bytebase won't apply the SQL to the database, it will only record a baseline history`;
  }
  if (!isRollBack && instance.value.engine === "SNOWFLAKE") {
    return `Use <<schema>>.<<table>> to specify a Snowflake table`;
  }
  return undefined;
};
</script>
