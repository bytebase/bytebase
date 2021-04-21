<template>
  <div
    id="issue-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <div
      v-if="showCancelBanner"
      class="h-10 w-full text-2xl font-bold bg-gray-400 text-white flex justify-center items-center"
    >
      Canceled
    </div>
    <div
      v-if="showSuccessBanner"
      class="h-10 w-full text-2xl font-bold bg-success text-white flex justify-center items-center"
    >
      Done
    </div>
    <!-- Highlight Panel -->
    <div class="bg-white px-4 pb-4">
      <IssueHighlightPanel
        :issue="state.issue"
        :new="state.new"
        :allowEdit="allowEditNameAndDescription"
        @update-name="updateName"
      >
        <template v-if="state.new">
          <button
            type="button"
            class="btn-primary px-4 py-2"
            @click.prevent="doCreate"
            :disabled="!allowCreate"
          >
            Create
          </button>
        </template>
        <!-- Action Button List -->
        <div
          v-else-if="applicableStepStatusTransitionList.length > 0"
          class="flex space-x-2"
        >
          <template
            v-for="(
              transition, index
            ) in applicableStepStatusTransitionList.reverse()"
            :key="index"
          >
            <button
              type="button"
              :class="transition.buttonClass"
              @click.prevent="tryStartStepStatusTransition(transition)"
            >
              {{ transition.buttonName }}
            </button>
          </template>
        </div>
      </IssueHighlightPanel>
    </div>

    <!-- Task Flow Bar -->
    <template v-if="showPipelineFlowBar">
      <template v-if="currentPipelineType == 'MULTI_SINGLE_STEP_TASK'">
        <PipelineSimpleFlow :pipeline="state.issue.pipeline" />
      </template>
    </template>

    <!-- Output Panel -->
    <!-- Only render the top border if IssueTaskFlow is not displayed, otherwise it would overlap with the bottom border of the IssueTaskFlow -->
    <div
      v-if="showIssueOutputPanel"
      class="px-2 py-4 md:flex md:flex-col"
      :class="showPipelineFlowBar ? '' : 'lg:border-t'"
    >
      <IssueOutputPanel
        :issue="state.issue"
        :fieldList="outputFieldList"
        :allowEdit="allowEditOutput"
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
              :issue="state.issue"
              :new="state.new"
              :fieldList="inputFieldList"
              :allowEdit="allowEditSidebar"
              @update-assignee-id="updateAssigneeId"
              @update-subscriber-list="updateSubscriberIdList"
              @update-custom-field="updateCustomField"
            />
          </div>
          <div class="lg:hidden border-t border-block-border" />
          <div class="w-full py-6 pr-4">
            <section v-if="showIssueSqlPanel" class="border-b mb-4">
              <IssueSqlPanel
                :issue="state.issue"
                :new="state.new"
                :rollback="false"
                :allowEdit="allowEditSql"
                @update-sql="updateSql"
              />
            </section>
            <section v-if="showIssueRollbackSqlPanel" class="border-b mb-4">
              <IssueSqlPanel
                :issue="state.issue"
                :new="state.new"
                :rollback="true"
                :allowEdit="allowEditSql"
                @update-sql="updateRollbackSql"
              />
            </section>
            <IssueDescriptionPanel
              :issue="state.issue"
              :new="state.new"
              :allowEdit="allowEditNameAndDescription"
              @update-description="updateDescription"
            />
            <section
              v-if="!state.new"
              aria-labelledby="activity-title"
              class="mt-4"
            >
              <IssueActivityPanel
                :issue="state.issue"
                :issueTemplate="issueTemplate"
                @update-subscriber-list="updateSubscriberIdList"
              />
            </section>
          </div>
        </div>
      </div>
    </main>
  </div>
  <BBModal
    v-if="updateStatusModalState.show"
    :title="updateStatusModalState.title"
    @close="updateStatusModalState.show = false"
  >
    <IssueStatusTransitionForm
      :okText="updateStatusModalState.okText"
      :issue="state.issue"
      :transition="updateStatusModalState.payload.transition"
      :outputFieldList="outputFieldList"
      @submit="
        (comment) => {
          updateStatusModalState.show = false;
          doIssueStatusTransition(updateStatusModalState.payload, comment);
        }
      "
      @cancel="
        () => {
          updateStatusModalState.show = false;
        }
      "
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, onMounted, watch, watchEffect, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEmpty from "lodash-es/isEmpty";
import isEqual from "lodash-es/isEqual";
import {
  idFromSlug,
  issueSlug,
  isDemo,
  pendingResolve,
  StepStatusTransition,
  applicableStepTransition,
  activeStep,
  pipelineType,
  PipelineType,
} from "../utils";
import IssueHighlightPanel from "../views/IssueHighlightPanel.vue";
import IssueTaskFlow from "./IssueTaskFlow.vue";
import IssueOutputPanel from "../views/IssueOutputPanel.vue";
import IssueSqlPanel from "../views/IssueSqlPanel.vue";
import IssueDescriptionPanel from "./IssueDescriptionPanel.vue";
import IssueActivityPanel from "../views/IssueActivityPanel.vue";
import IssueSidebar from "../views/IssueSidebar.vue";
import IssueStatusTransitionForm from "../components/IssueStatusTransitionForm.vue";
import PipelineSimpleFlow from "./PipelineSimpleFlow.vue";
import {
  Issue,
  IssueNew,
  IssueType,
  IssuePatch,
  IssueStatusTransition,
  IssueStatusTransitionType,
  TaskStatusPatch,
  PrincipalId,
  ISSUE_STATUS_TRANSITION_LIST,
  Database,
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  IssueStatusPatch,
  TaskStatus,
  TaskId,
  UNKNOWN_ID,
  ProjectId,
  Environment,
  StepStatusPatch,
  StepId,
  Pipeline,
  EMPTY_ID,
} from "../types";
import {
  defaulTemplate,
  templateForType,
  IssueField,
  IssueTemplate,
  IssueContext,
} from "../plugins";

type UpdateStatusModalStatePayload = {
  transition: IssueStatusTransition;
  didTransit: () => {};
};

interface UpdateStatusModalState {
  show: boolean;
  style: string;
  okText: string;
  title: string;
  payload?: UpdateStatusModalStatePayload;
}

interface LocalState {
  // Needs to maintain this state and set it to false manually after creating the issue.
  // router.push won't trigger the reload because new and existing issue shares
  // the same component.
  new: boolean;
  issue: Issue | IssueNew;
}

export default {
  name: "IssueDetail",
  props: {
    issueSlug: {
      required: true,
      type: String,
    },
  },
  components: {
    IssueHighlightPanel,
    IssueTaskFlow,
    IssueOutputPanel,
    IssueSqlPanel,
    IssueDescriptionPanel,
    IssueActivityPanel,
    IssueSidebar,
    IssueStatusTransitionForm,
    PipelineSimpleFlow,
  },

  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const updateStatusModalState = reactive<UpdateStatusModalState>({
      show: false,
      style: "INFO",
      okText: "OK",
      title: "",
    });

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("issue-detail-top")!.scrollIntoView();
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const issueContext = computed(
      (): IssueContext => {
        return {
          store,
          currentUser: currentUser.value,
          new: state.new,
          issue: state.issue,
        };
      }
    );

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
            style: "CRITICAL",
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
      (curTemplate, prevTemplate) => {
        refreshTemplate();
      }
    );

    watchEffect(refreshTemplate);

    const isNew = props.issueSlug.toLowerCase() == "new";

    let newIssue: IssueNew;
    if (isNew) {
      const databaseList: Database[] = [];
      if (router.currentRoute.value.query.databaseList) {
        for (const databaseId of (router.currentRoute.value.query
          .databaseList as string).split(","))
          databaseList.push(store.getters["database/databaseById"](databaseId));
      }

      const environmentList: Environment[] = [];
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
      }

      newIssue = {
        ...newIssueTemplate.value.buildIssue({
          environmentList,
          databaseList,
          currentUser: currentUser.value,
        }),
        projectId: router.currentRoute.value.query.project
          ? (router.currentRoute.value.query.project as ProjectId)
          : UNKNOWN_ID,
        creatorId: currentUser.value.id,
      };

      // For demo mode, we assign the issue to the current user, so it can also experience the assignee user flow.
      if (isDemo()) {
        newIssue.assigneeId = currentUser.value.id;
      }

      if (router.currentRoute.value.query.name) {
        newIssue.name = router.currentRoute.value.query.name as string;
      }
      if (router.currentRoute.value.query.description) {
        newIssue.description = router.currentRoute.value.query
          .description as string;
      }
      if (router.currentRoute.value.query.sql) {
        newIssue.sql = router.currentRoute.value.query.sql as string;
      }
      if (router.currentRoute.value.query.rollbacksql) {
        newIssue.rollbackSql = router.currentRoute.value.query
          .rollbacksql as string;
      }
      if (router.currentRoute.value.query.assignee) {
        newIssue.assigneeId = router.currentRoute.value.query
          .assignee as PrincipalId;
      }

      for (const field of newIssueTemplate.value.fieldList.filter(
        (item) => item.category == "INPUT"
      )) {
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
    }

    const state = reactive<LocalState>({
      new: isNew,
      issue: isNew
        ? newIssue!
        : cloneDeep(
            store.getters["issue/issueById"](idFromSlug(props.issueSlug))
          ),
    });

    const refreshIssue = () => {
      state.issue = state.new
        ? newIssue
        : cloneDeep(
            store.getters["issue/issueById"](idFromSlug(props.issueSlug))
          );
    };

    watchEffect(refreshIssue);

    const issueTemplate = computed(
      () => templateForType(state.issue.type) || defaulTemplate()
    );

    const outputFieldList = computed(
      () =>
        issueTemplate.value.fieldList.filter(
          (item) => item.category == "OUTPUT"
        ) || []
    );
    const inputFieldList = computed(
      () =>
        issueTemplate.value.fieldList.filter(
          (item) => item.category == "INPUT"
        ) || []
    );

    const updateName = (
      newName: string,
      postUpdated: (updatedIssue: Issue) => void
    ) => {
      if (state.new) {
        (state.issue as IssueNew).name = newName;
      } else {
        patchIssue(
          {
            name: newName,
          },
          postUpdated
        );
      }
    };

    const updateSql = (
      newSql: string,
      postUpdated: (updatedIssue: Issue) => void
    ) => {
      if (state.new) {
        (state.issue as IssueNew).sql = newSql;
      } else {
        patchIssue(
          {
            sql: newSql,
          },
          postUpdated
        );
      }
    };

    const updateRollbackSql = (
      newSql: string,
      postUpdated: (updatedIssue: Issue) => void
    ) => {
      if (state.new) {
        (state.issue as IssueNew).rollbackSql = newSql;
      } else {
        patchIssue(
          {
            rollbackSql: newSql,
          },
          postUpdated
        );
      }
    };

    const updateDescription = (
      newDescription: string,
      postUpdated: (updatedIssue: Issue) => void
    ) => {
      if (state.new) {
        (state.issue as IssueNew).description = newDescription;
      } else {
        patchIssue(
          {
            description: newDescription,
          },
          postUpdated
        );
      }
    };

    const allowTransition = (transition: IssueStatusTransition): boolean => {
      const issue: Issue = state.issue as Issue;
      if (transition.type == "RESOLVE") {
        if (pendingResolve(issue)) {
          // Returns false if any of the required output fields is not provided.
          for (let i = 0; i < outputFieldList.value.length; i++) {
            const field = outputFieldList.value[i];
            if (field.required && !field.resolved(issueContext.value)) {
              return false;
            }
          }
          return true;
        }
        return false;
      }
      return true;
    };

    const tryStartStepStatusTransition = (transition: StepStatusTransition) => {
      const step = activeStep((state.issue as Issue).pipeline);
      doStepStatusTransition(transition, step.id);
    };

    const doStepStatusTransition = (
      transition: StepStatusTransition,
      stepId: StepId,
      comment?: string
    ) => {
      const stepStatusPatch: StepStatusPatch = {
        updaterId: currentUser.value.id,
        status: transition.to,
        comment: comment ? comment.trim() : undefined,
      };

      store.dispatch("step/updateStatus", {
        issueId: (state.issue as Issue).id,
        pipelineId: (state.issue as Issue).pipeline.id,
        stepId,
        stepStatusPatch,
      });
    };

    const changeStepStatus = (
      taskId: TaskId,
      taskStatus: TaskStatus,
      comment?: string
    ) => {
      const taskStatusPatch: TaskStatusPatch = {
        updaterId: currentUser.value.id,
        status: taskStatus,
        comment: comment ? comment.trim() : undefined,
      };

      store.dispatch("task/updateTaskStatus", {
        issueId: (state.issue as Issue).id,
        taskId,
        taskStatusPatch,
      });
    };

    const tryStartIssueStatusTransition = (
      type: IssueStatusTransitionType,
      didTransit: () => {}
    ) => {
      const transition = ISSUE_STATUS_TRANSITION_LIST.get(type)!;
      updateStatusModalState.okText = transition.actionName;
      switch (transition.to) {
        case "OPEN":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = "Reopen issue?";
          break;
        case "DONE":
          updateStatusModalState.style = "SUCCESS";
          updateStatusModalState.title = "Resolve issue?";
          break;
        case "CANCELED":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = "Abort issue?";
          break;
      }
      updateStatusModalState.payload = {
        transition,
        didTransit,
      };
      updateStatusModalState.show = true;
    };

    const doIssueStatusTransition = (
      payload: UpdateStatusModalStatePayload,
      comment?: string
    ) => {
      const issueStatusPatch: IssueStatusPatch = {
        updaterId: currentUser.value.id,
        status: payload.transition.to,
        comment: comment ? comment.trim() : undefined,
      };

      store
        .dispatch("issue/updateIssueStatus", {
          issueId: (state.issue as Issue).id,
          issueStatusPatch,
        })
        .then((updatedIssue) => {
          if (
            payload.transition.to == "DONE" &&
            issueTemplate.value.type == "bytebase.database.schema.update"
          ) {
            store.dispatch("uistate/saveIntroStateByKey", {
              key: "table.create",
              newState: true,
            });
          }
          payload.didTransit();
        });
    };

    const changeTaskStatus = (
      taskId: TaskId,
      taskStatus: TaskStatus,
      comment?: string
    ) => {
      const taskStatusPatch: TaskStatusPatch = {
        updaterId: currentUser.value.id,
        status: taskStatus,
        comment: comment ? comment.trim() : undefined,
      };

      store.dispatch("task/updateTaskStatus", {
        issueId: (state.issue as Issue).id,
        taskId,
        taskStatusPatch,
      });
    };

    const updateAssigneeId = (newAssigneeId: PrincipalId) => {
      if (state.new) {
        (state.issue as IssueNew).assigneeId = newAssigneeId;
      } else {
        patchIssue({
          assigneeId: newAssigneeId,
        });
      }
    };

    const updateSubscriberIdList = (newSubscriberIdList: PrincipalId[]) => {
      patchIssue({
        subscriberIdList: newSubscriberIdList,
      });
    };

    const updateCustomField = (field: IssueField, value: any) => {
      console.log("updateCustomField", field.name, value);
      if (!isEqual(state.issue.payload[field.id], value)) {
        state.issue.payload[field.id] = value;
        if (!state.new) {
          patchIssue({
            payload: state.issue.payload,
          });
        }
      }
    };

    const doCreate = () => {
      store
        .dispatch("issue/createIssue", state.issue)
        .then((createdIssue) => {
          state.new = false;
          router.push(
            `/issue/${issueSlug(createdIssue.name, createdIssue.id)}`
          );
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const patchIssue = (
      issuePatch: Omit<IssuePatch, "updaterId">,
      postUpdated?: (updatedIssue: Issue) => void
    ) => {
      store
        .dispatch("issue/patchIssue", {
          issueId: (state.issue as Issue).id,
          issuePatch: {
            ...issuePatch,
            updaterId: currentUser.value.id,
          },
        })
        .then((updatedIssue) => {
          if (postUpdated) {
            postUpdated(updatedIssue);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const currentPipelineType = computed(
      (): PipelineType => {
        return pipelineType((state.issue as Issue).pipeline);
      }
    );

    console.log(currentPipelineType.value);

    const allowCreate = computed(() => {
      const newIssue = state.issue as IssueNew;
      if (isEmpty(newIssue.name)) {
        return false;
      }

      if (!newIssue.assigneeId) {
        return false;
      }

      if (newIssueTemplate.value.fieldList) {
        for (const field of newIssueTemplate.value.fieldList.filter(
          (item) => item.category == "INPUT"
        )) {
          if (
            field.type != "Boolean" && // Switch is boolean value which always is present
            field.required &&
            !field.resolved(issueContext.value)
          ) {
            return false;
          }
        }
      }
      return true;
    });

    const allowEditSidebar = computed(() => {
      // For now, we only allow assignee to update the field when the issue
      // is 'OPEN'. This reduces flexibility as creator must ask assignee to
      // change any fields if there is typo. On the other hand, this avoids
      // the trouble that the creator changes field value when the creator
      // is performing the issue based on the old value.
      // For now, we choose to be on the safe side at the cost of flexibility.
      return (
        state.new ||
        ((state.issue as Issue).status == "OPEN" &&
          currentUser.value.id == (state.issue as Issue).assignee?.id)
      );
    });

    const allowEditOutput = computed(() => {
      return (
        state.new ||
        ((state.issue as Issue).status == "OPEN" &&
          currentUser.value.id == (state.issue as Issue).assignee?.id)
      );
    });

    const allowEditNameAndDescription = computed(() => {
      return (
        state.new ||
        ((state.issue as Issue).status == "OPEN" &&
          (currentUser.value.id == (state.issue as Issue).assignee?.id ||
            currentUser.value.id == (state.issue as Issue).creator.id))
      );
    });

    const allowEditSql = computed(() => {
      return state.new;
    });

    const showCancelBanner = computed(() => {
      return !state.new && (state.issue as Issue).status == "CANCELED";
    });

    const showSuccessBanner = computed(() => {
      return !state.new && (state.issue as Issue).status == "DONE";
    });

    const showPipelineFlowBar = computed(() => {
      return !state.new && currentPipelineType.value != "NO_PIPELINE";
    });

    const showIssueOutputPanel = computed(() => {
      return !state.new && outputFieldList.value.length > 0;
    });

    const showIssueSqlPanel = computed(() => {
      return (
        state.issue.type == "bytebase.general" ||
        state.issue.type == "bytebase.database.schema.update"
      );
    });

    const showIssueRollbackSqlPanel = computed(() => {
      return state.issue.type == "bytebase.database.schema.update";
    });

    const applicableStepStatusTransitionList = computed(
      (): StepStatusTransition[] => {
        let list: StepStatusTransition[] = [];

        if (currentUser.value.id === (state.issue as Issue).assignee?.id) {
          list = applicableStepTransition((state.issue as Issue).pipeline);
        }

        return list;
      }
    );

    const applicableStatusTransitionList = computed(
      (): IssueStatusTransition[] => {
        const list: IssueStatusTransitionType[] = [];
        if (currentUser.value.id === (state.issue as Issue).assignee?.id) {
          list.push(
            ...ASSIGNEE_APPLICABLE_ACTION_LIST.get(
              (state.issue as Issue).status
            )!
          );
        }
        if (currentUser.value.id === (state.issue as Issue).creator.id) {
          CREATOR_APPLICABLE_ACTION_LIST.get(
            (state.issue as Issue).status
          )!.forEach((item) => {
            if (list.indexOf(item) == -1) {
              list.push(item);
            }
          });
        }

        return list
          .filter((item) => {
            if (pendingResolve(state.issue as Issue)) {
              if (item == "NEXT") {
                return false;
              }
            } else {
              if (item == "RESOLVE") {
                return false;
              }
            }
            return true;
          })
          .map(
            (type: IssueStatusTransitionType) =>
              ISSUE_STATUS_TRANSITION_LIST.get(type)!
          );
      }
    );

    return {
      updateStatusModalState,
      state,
      updateName,
      updateDescription,
      updateSql,
      updateRollbackSql,
      allowTransition,
      tryStartStepStatusTransition,
      doStepStatusTransition,
      changeStepStatus,
      tryStartIssueStatusTransition,
      doIssueStatusTransition,
      changeTaskStatus,
      updateAssigneeId,
      updateSubscriberIdList,
      updateCustomField,
      doCreate,
      currentPipelineType,
      allowCreate,
      currentUser,
      issueTemplate,
      outputFieldList,
      inputFieldList,
      allowEditSidebar,
      allowEditOutput,
      allowEditNameAndDescription,
      allowEditSql,
      showCancelBanner,
      showSuccessBanner,
      showPipelineFlowBar,
      showIssueOutputPanel,
      showIssueSqlPanel,
      showIssueRollbackSqlPanel,
      applicableStepStatusTransitionList,
      applicableStatusTransitionList,
    };
  },
};
</script>
