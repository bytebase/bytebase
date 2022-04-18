<template>
  <div
    id="issue-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <IssueBanner v-if="!create" :issue="issue" />

    <!-- Highlight Panel -->
    <div class="bg-white px-4 pb-4">
      <IssueHighlightPanel
        :issue="issue"
        :create="create"
        :allow-edit="allowEditNameAndDescription"
        @update-name="updateName"
      >
        <IssueStatusTransitionButtonGroup
          :create="create"
          :issue="issue"
          :issue-template="issueTemplate"
          @create="doCreate"
          @change-issue-status="changeIssueStatus"
          @change-task-status="changeTaskStatus"
        />
      </IssueHighlightPanel>
    </div>

    <!-- Remind banner for bb.feature.backward-compatibility -->
    <FeatureAttention
      v-if="
        !hasBackwardCompatibilityFeature && supportBackwardCompatibilityFeature
      "
      custom-class="m-5 mt-0"
      feature="bb.feature.backward-compatibility"
      :description="
        $t('subscription.features.bb-feature-backward-compatibility.desc')
      "
    />

    <!-- Stage Flow Bar -->
    <template v-if="showPipelineFlowBar">
      <template v-if="isTenantDeployMode">
        <PipelineTenantFlow
          v-if="project"
          :create="create"
          :project="project"
          :pipeline="issue.pipeline"
          :selected-stage="selectedStage"
          :selected-task="selectedTask"
          class="border-t border-b"
          @select-stage-id="selectStageId"
          @select-task="selectTask"
        />
      </template>
      <template v-else>
        <PipelineSimpleFlow
          :create="create"
          :pipeline="issue.pipeline"
          :selected-stage="selectedStage"
          @select-stage-id="selectStageId"
        />
      </template>

      <div v-if="!create" class="px-4 py-4 md:flex md:flex-col border-b">
        <IssueStagePanel
          :stage="selectedStage"
          :selected-task="selectedTask"
          :is-tenant-mode="isTenantDeployMode"
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
              :is-tenant-deploy-mode="isTenantDeployMode"
              @update-assignee-id="updateAssigneeId"
              @update-earliest-allowed-time="updateEarliestAllowedTime"
              @add-subscriber-id="addSubscriberId"
              @remove-subscriber-id="removeSubscriberId"
              @update-custom-field="updateCustomField"
              @select-stage-id="selectStageId"
              @select-task-id="selectTaskId"
            />
          </div>
          <div class="lg:hidden border-t border-block-border" />
          <div class="w-full py-4 pr-4">
            <section v-if="showIssueTaskStatementPanel" class="border-b mb-4">
              <div v-if="!create" class="mb-4">
                <TaskCheckBar
                  :task="selectedTask"
                  @run-checks="runTaskChecks"
                />
              </div>
              <template v-if="isTenantDeployMode">
                <!--
                  For tenant deploy mode, we provide only one statement panel.
                  It's editable only when creating an issue.
                  It is not allowed to "Apply to other stages".
                -->
                <IssueTaskStatementPanel
                  :sql-hint="sqlHint()"
                  :statement="selectedStatement"
                  :create="create"
                  :allow-edit="create"
                  :show-apply-statement="false"
                  @update-statement="updateStatement"
                />
              </template>
              <template v-else-if="isGhostMode">
                <!--
                  For gh-ost mode, only the first task (bb.task.database.schema.update.ghost.sync)
                  has a SQL statement.
                  So we display the first task's SQL statement what ever the selectedTaskIs
                -->
                <IssueTaskStatementPanel
                  :sql-hint="sqlHint()"
                  :statement="selectedStatement"
                  :create="create"
                  :allow-edit="allowEditStatement"
                  :show-apply-statement="showIssueTaskStatementApply"
                  @update-statement="updateStatement"
                  @apply-statement-to-other-stages="applyStatementToOtherStages"
                />
              </template>
              <template v-else>
                <!-- The way this is written is awkward and is to workaround an issue in IssueTaskStatementPanel.
                   The statement panel is in non-edit mode when not creating the issue, and we use v-highlight
                   to apply syntax highlighting when the panel is in non-edit mode. However, the v-highlight
                   doesn't seem to work well with the reactivity. So for non-edit mode when !props.create, we
                list every IssueTaskStatementPanel for each stage and use v-if to show the active one.-->
                <template v-if="create">
                  <IssueTaskStatementPanel
                    :sql-hint="sqlHint()"
                    :statement="selectedStatement"
                    :create="create"
                    :allow-edit="true"
                    :show-apply-statement="showIssueTaskStatementApply"
                    @update-statement="updateStatement"
                    @apply-statement-to-other-stages="
                      applyStatementToOtherStages
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
                      :sql-hint="sqlHint()"
                      :statement="statement(stage)"
                      :create="create"
                      :allow-edit="allowEditStatement"
                      :show-apply-statement="showIssueTaskStatementApply"
                      @update-statement="updateStatement"
                    />
                  </template>
                </template>
              </template>
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

<script lang="ts">
/* eslint-disable vue/no-mutating-props */

import {
  computed,
  defineComponent,
  nextTick,
  onMounted,
  PropType,
  watchEffect,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { cloneDeep, isEqual } from "lodash-es";
import {
  idFromSlug,
  issueSlug,
  pipelineType,
  PipelineType,
  indexFromSlug,
  activeStage,
  stageSlug,
  activeTask,
  taskSlug,
} from "../../utils";
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
import TaskCheckBar from "./TaskCheckBar.vue";
import {
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
  TaskDatabaseDataUpdatePayload,
  StageCreate,
  TaskCreate,
  TaskDatabaseCreatePayload,
  TaskGeneralPayload,
  Project,
  MigrationType,
  TaskPatch,
  UpdateSchemaContext,
  UpdateSchemaDetail,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
} from "../../types";
import {
  defaulTemplate as defaultTemplate,
  templateForType,
  InputField,
  OutputField,
} from "../../plugins";
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

export default defineComponent({
  name: "IssueDetailLayout",
  components: {
    IssueBanner,
    IssueHighlightPanel,
    IssueStagePanel,
    IssueOutputPanel,
    IssueTaskStatementPanel,
    IssueDescriptionPanel,
    IssueActivityPanel,
    IssueSidebar,
    IssueStatusTransitionButtonGroup,
    PipelineSimpleFlow,
    PipelineTenantFlow,
    TaskCheckBar,
  },
  props: {
    create: {
      type: Boolean,
      required: true,
    },
    issue: {
      type: Object as PropType<Issue | IssueCreate>,
      required: true,
    },
  },
  emits: {
    "status-changed": (eager: boolean) => true,
  },
  setup(props, { emit }) {
    const router = useRouter();
    const route = useRoute();

    const currentUser = useCurrentUser();
    const issueStore = useIssueStore();
    const issueSubscriberStore = useIssueSubscriberStore();
    const taskStore = useTaskStore();
    const projectStore = useProjectStore();

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
        return projectStore.getProjectById(
          (props.issue as IssueCreate).projectId
        );
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

    const updateStatement = (
      newStatement: string,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      if (props.create) {
        if (isTenantDeployMode.value) {
          // For tenant deploy mode, we apply the statement to all stages and all tasks
          const issueCreate = props.issue as IssueCreate;
          const context = issueCreate.createContext as UpdateSchemaContext;
          issueCreate.pipeline?.stageList.forEach((stage: StageCreate) => {
            stage.taskList.forEach((task) => {
              task.statement = newStatement;
            });
          });
          // We also apply it to the CreateContext
          context.updateSchemaDetailList.forEach(
            (detail) => (detail.statement = newStatement)
          );
        } else {
          // otherwise apply it to the only one task in stage
          // i.e. selectedStage.taskList[0]
          const stage = selectedStage.value as StageCreate;
          stage.taskList[0].statement = newStatement;
        }
      } else {
        if (isTenantDeployMode.value) {
          // <del>For tenant deploy mode, we patch the issue's create context</del>
          // nope, we are not allowed to update statement in tenant deploy mode anyway
        } else {
          // otherwise, patch the task
          patchTask(
            (selectedTask.value as Task).id,
            {
              statement: newStatement,
            },
            postUpdated
          );
        }
      }
    };

    const applyStatementToOtherStages = (newStatement: string) => {
      for (const stage of (props.issue as IssueCreate).pipeline!.stageList) {
        for (const task of stage.taskList) {
          if (
            task.type == "bb.task.general" ||
            task.type == "bb.task.database.create" ||
            task.type == "bb.task.database.schema.update" ||
            task.type == "bb.task.database.schema.update.ghost.sync" ||
            task.type == "bb.task.database.data.update"
          ) {
            task.statement = newStatement;
          }
        }
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
        selectedTask.value.earliestAllowedTs = newEarliestAllowedTsMs;
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

    const doCreate = () => {
      const issue = cloneDeep(props.issue) as IssueCreate;
      if (!isTenantDeployMode.value) {
        // for standard issue pipeline (1 * 1 or M * 1)
        // copy user edited tasks back to issue.createContext
        const taskList = issue.pipeline!.stageList.map(
          (stage) => stage.taskList[0]
        );
        const detailList: UpdateSchemaDetail[] = taskList.map((task) => {
          return {
            databaseId: task.databaseId!,
            databaseName: task.databaseName!,
            statement: task.statement,
            earliestAllowedTs: task.earliestAllowedTs,
          };
        });
        issue.createContext = {
          migrationType: taskList[0].migrationType!,
          updateSchemaDetailList: detailList,
        };
      } else {
        // for multi-tenancy issue pipeline (M * N)
        // createContext is up-to-date already
        // so nothing to do
      }
      // then empty issue.pipeline and issue.payload
      // because we are no longer passing parameters via issue.pipeline
      // we are using issue.createContext instead
      delete issue.pipeline;
      issue.payload = {};

      issueStore.createIssue(issue).then((createdIssue) => {
        // Use replace to omit the new issue url in the navigation history.
        router.replace(
          `/issue/${issueSlug(createdIssue.name, createdIssue.id)}`
        );
      });
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
      selectStageId(task.stage.id);
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
          // pollIssue(POST_CHANGE_POLL_INTERVAL);
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

    const selectStageId = (
      stageId: StageId,
      task: string | undefined = undefined
    ) => {
      const stageList = props.issue.pipeline!.stageList;
      const index = stageList.findIndex((item, index) => {
        if (props.create) {
          return index == stageId;
        }
        return (item as Stage).id == stageId;
      });
      router.replace({
        name: "workspace.issue.detail",
        query: {
          ...router.currentRoute.value.query,
          stage: stageSlug(stageList[index].name, index),
          task,
        },
      });
    };

    const selectTask = (stageId: StageId, taskSlug: string) => {
      selectStageId(stageId, taskSlug);
    };

    const selectTaskId = (taskId: TaskId) => {
      const taskList = selectedStage.value.taskList as Task[];
      const task = taskList.find((t) => t.id === taskId);
      if (!task) return;
      const slug = taskSlug(task.name, task.id);
      const stage = selectedStage.value as Stage;
      selectTask(stage.id, slug);
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

    const statement = (stage: Stage): string => {
      const task = stage.taskList[0];
      switch (task.type) {
        case "bb.task.general":
          return ((task as Task).payload as TaskGeneralPayload).statement || "";
        case "bb.task.database.create":
          return (
            ((task as Task).payload as TaskDatabaseCreatePayload).statement ||
            ""
          );
        case "bb.task.database.schema.update":
          return (
            ((task as Task).payload as TaskDatabaseSchemaUpdatePayload)
              .statement || ""
          );
        case "bb.task.database.data.update":
          return (
            ((task as Task).payload as TaskDatabaseDataUpdatePayload)
              .statement || ""
          );
        case "bb.task.database.restore":
          return "";
      }
    };

    const isTenantDeployMode = computed((): boolean => {
      return (
        (props.issue.type === "bb.issue.database.schema.update" ||
          props.issue.type === "bb.issue.database.data.update") &&
        project.value.tenantMode === "TENANT"
      );
    });

    const isGhostMode = computed((): boolean => {
      return props.issue.type === "bb.issue.database.schema.update.ghost";
    });

    const selectedStatement = computed((): string => {
      if (isTenantDeployMode.value) {
        if (props.create) {
          const issueCreate = props.issue as IssueCreate;
          const context = issueCreate.createContext as UpdateSchemaContext;
          return context.updateSchemaDetailList[0].statement;
        } else {
          const issue = props.issue as Issue;
          const task = issue.pipeline.stageList[0].taskList[0];
          const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
          return payload.statement;
        }
      } else if (isGhostMode.value) {
        if (props.create) {
          const issueCreate = props.issue as IssueCreate;
          const context = issueCreate.createContext as UpdateSchemaContext;
          return context.updateSchemaDetailList[0].statement;
        } else {
          const stage = selectedStage.value as Stage;
          const task = stage.taskList[0];
          const payload =
            task.payload as TaskDatabaseSchemaUpdateGhostSyncPayload;
          return payload.statement;
        }
      } else {
        if (router.currentRoute.value.query.sql) {
          const sql = router.currentRoute.value.query.sql as string;
          updateStatement(sql);
        }

        const task = (selectedStage.value as StageCreate).taskList[0];
        return task.statement;
      }
    });

    const selectedMigrateType = computed((): MigrationType => {
      if (
        !props.create &&
        selectedTask.value.type == "bb.task.database.schema.update"
      ) {
        return (
          (selectedTask.value as Task)
            .payload as TaskDatabaseSchemaUpdatePayload
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

    const allowEditStatement = computed(() => {
      // if creating an issue, it's editable
      if (props.create) {
        return true;
      }
      const checkTask = (task: Task) => {
        return (
          task.status == "PENDING" ||
          task.status == "PENDING_APPROVAL" ||
          task.status == "FAILED"
        );
      };

      const issue = props.issue as Issue;
      // if not creating, we are allowed to edit sql statement only when:
      // 1. issue.status is OPEN
      // 2. AND currentUser is the creator
      // 3. AND workflowType is UI
      if (issue.status !== "OPEN") return false;
      if (issue.creator.id !== currentUser.value.id) return false;
      if (issue.project.workflowType !== "UI") return false;

      if (isTenantDeployMode.value) {
        // <del>then if in tenant deploy mode, EVERY task must be PENDING or PENDING_APPROVAL or FAILED</del>
        // nope, we are not allowed to update statement in tenant deploy mode anyway
        // const allTasks = issue.pipeline.stageList.flatMap(
        //   (stage) => stage.taskList
        // );
        // return allTasks.every((task) => checkTask(task));
        return false;
      } else {
        // otherwise, check `selectedTask`, expected to be PENDING or PENDING_APPROVAL or FAILED
        return checkTask(selectedTask.value as Task);
      }
    });

    const showCancelBanner = computed(() => {
      return !props.create && (props.issue as Issue).status == "CANCELED";
    });

    const showSuccessBanner = computed(() => {
      return !props.create && (props.issue as Issue).status == "DONE";
    });

    const showPendingApproval = computed(() => {
      if (props.create) {
        return false;
      }

      const task = activeTask((props.issue as Issue).pipeline);
      return task.status == "PENDING_APPROVAL";
    });

    const showPipelineFlowBar = computed(() => {
      return currentPipelineType.value != "NO_PIPELINE";
    });

    const showIssueOutputPanel = computed(() => {
      return !props.create && issueTemplate.value.outputFieldList.length > 0;
    });

    const showIssueTaskStatementPanel = computed(() => {
      if (props.issue.type === "bb.issue.database.schema.update.ghost") {
        return true;
      }

      const task = selectedTask.value;
      return (
        task.type == "bb.task.general" ||
        task.type == "bb.task.database.create" ||
        task.type == "bb.task.database.schema.update" ||
        task.type == "bb.task.database.data.update"
      );
    });

    const showIssueTaskStatementApply = computed(() => {
      if (!props.create) {
        return false;
      }
      if (isTenantDeployMode.value) {
        return false;
      }
      let count = 0;
      for (const stage of (props.issue as IssueCreate).pipeline!.stageList) {
        for (const task of stage.taskList) {
          if (
            task.type == "bb.task.general" ||
            task.type == "bb.task.database.create" ||
            task.type == "bb.task.database.schema.update" ||
            task.type == "bb.task.database.schema.update.ghost.sync" ||
            task.type == "bb.task.database.data.update"
          ) {
            count++;
          }
        }
      }
      return count > 1;
    });

    const database = computed((): Database | undefined => {
      if (props.create) {
        const databaseId = (selectedTask.value as TaskCreate).databaseId;
        if (databaseId) {
          return useDatabaseStore().getDatabaseById(databaseId);
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

    return {
      database,
      instance,
      sqlHint,
      updateName,
      updateDescription,
      updateStatement,
      updateEarliestAllowedTime,
      applyStatementToOtherStages,
      updateAssigneeId,
      addSubscriberId,
      removeSubscriberId,
      updateCustomField,
      doCreate,
      changeIssueStatus,
      changeTaskStatus,
      runTaskChecks,
      currentPipelineType,
      currentUser,
      project,
      isTenantDeployMode,
      isGhostMode,
      issueTemplate,
      selectedStage,
      selectedTask,
      selectStageId,
      selectTask,
      selectTaskId,
      statement,
      selectedStatement,
      selectedMigrateType,
      allowEditSidebar,
      allowEditOutput,
      allowEditNameAndDescription,
      allowEditStatement,
      showCancelBanner,
      showSuccessBanner,
      showPendingApproval,
      showPipelineFlowBar,
      showIssueOutputPanel,
      showIssueTaskStatementPanel,
      showIssueTaskStatementApply,
      hasBackwardCompatibilityFeature,
      supportBackwardCompatibilityFeature,
    };
  },
});
</script>
