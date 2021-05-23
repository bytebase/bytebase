<template>
  <div
    id="issue-detail-top"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
  >
    <div
      v-if="showCancelBanner"
      class="
        h-10
        w-full
        text-2xl
        font-bold
        bg-gray-400
        text-white
        flex
        justify-center
        items-center
      "
    >
      Canceled
    </div>
    <div
      v-if="showSuccessBanner"
      class="
        h-10
        w-full
        text-2xl
        font-bold
        bg-success
        text-white
        flex
        justify-center
        items-center
      "
    >
      Done
    </div>
    <!-- Highlight Panel -->
    <div class="bg-white px-4 pb-4">
      <IssueHighlightPanel
        :issue="issue"
        :create="state.create"
        :allowEdit="allowEditNameAndDescription"
        @update-name="updateName"
      >
        <IssueStatusTransitionButtonGroup
          :create="state.create"
          :issue="issue"
          :issueTemplate="issueTemplate"
          @create="doCreate"
        />
      </IssueHighlightPanel>
    </div>

    <!-- Stage Flow Bar -->
    <template v-if="showPipelineFlowBar">
      <template
        v-if="
          currentPipelineType == 'MULTI_SINGLE_TASK_STAGE' ||
          currentPipelineType == 'SINGLE_STAGE'
        "
      >
        <PipelineSimpleFlow
          :pipeline="issue.pipeline"
          :selectedStage="selectedStage"
          @select-stage-id="selectStageId"
        />
      </template>
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
        :outputFieldList="issueTemplate.outputFieldList"
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
            class="
              py-6
              lg:pl-4
              lg:w-96
              xl:w-112
              lg:border-l lg:border-block-border
            "
          >
            <IssueSidebar
              :issue="issue"
              :create="state.create"
              :selectedStage="selectedStage"
              :inputFieldList="issueTemplate.inputFieldList"
              :allowEdit="allowEditSidebar"
              @update-assignee-id="updateAssigneeId"
              @update-subscriber-list="updateSubscriberIdList"
              @update-custom-field="updateCustomField"
              @select-stage-id="selectStageId"
            />
          </div>
          <div class="lg:hidden border-t border-block-border" />
          <div class="w-full py-6 pr-4">
            <section v-if="showIssueSqlPanel" class="border-b mb-4">
              <IssueSqlPanel
                :issue="issue"
                :create="state.create"
                :rollback="false"
                :allowEdit="allowEditSql"
                @update-sql="updateSql"
              />
            </section>
            <section v-if="showIssueRollbackSqlPanel" class="border-b mb-4">
              <IssueSqlPanel
                :issue="issue"
                :create="state.create"
                :rollback="true"
                :allowEdit="allowEditSql"
                @update-sql="updateRollbackSql"
              />
            </section>
            <IssueDescriptionPanel
              :issue="issue"
              :create="state.create"
              :allowEdit="allowEditNameAndDescription"
              @update-description="updateDescription"
            />
            <section
              v-if="!state.create"
              aria-labelledby="activity-title"
              class="mt-4"
            >
              <IssueActivityPanel
                :issue="issue"
                :issueTemplate="issueTemplate"
                @update-subscriber-list="updateSubscriberIdList"
              />
            </section>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, watch, watchEffect, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import {
  idFromSlug,
  issueSlug,
  isDemo,
  pipelineType,
  PipelineType,
  indexFromSlug,
  activeStage,
  stageSlug,
} from "../utils";
import IssueHighlightPanel from "../views/IssueHighlightPanel.vue";
import IssueOutputPanel from "../views/IssueOutputPanel.vue";
import IssueSqlPanel from "../views/IssueSqlPanel.vue";
import IssueDescriptionPanel from "./IssueDescriptionPanel.vue";
import IssueActivityPanel from "../views/IssueActivityPanel.vue";
import IssueSidebar from "../views/IssueSidebar.vue";
import IssueStatusTransitionButtonGroup from "../components/IssueStatusTransitionButtonGroup.vue";
import PipelineSimpleFlow from "./PipelineSimpleFlow.vue";
import {
  UNKNOWN_ID,
  Issue,
  IssueCreate,
  IssueType,
  IssuePatch,
  PrincipalId,
  Database,
  Environment,
  Stage,
  StageId,
} from "../types";
import {
  defaulTemplate,
  templateForType,
  InputField,
  OutputField,
  IssueTemplate,
} from "../plugins";

interface LocalState {
  // Needs to maintain this state and set it to false manually after creating the issue.
  // router.push won't trigger the reload because new and existing issue shares
  // the same component.
  create: boolean;
  newIssue?: IssueCreate;
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
    IssueOutputPanel,
    IssueSqlPanel,
    IssueDescriptionPanel,
    IssueActivityPanel,
    IssueSidebar,
    IssueStatusTransitionButtonGroup,
    PipelineSimpleFlow,
  },

  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    onMounted(() => {
      // Always scroll to top, the scrollBehavior doesn't seem to work.
      // The hypothesis is that because the scroll bar is in the nested
      // route, thus setting the scrollBehavior in the global router
      // won't work.
      document.getElementById("issue-detail-top")!.scrollIntoView();
    });

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

    const buildNewIssue = () => {
      const databaseList: Database[] = [];
      if (router.currentRoute.value.query.databaseList) {
        for (const databaseId of (
          router.currentRoute.value.query.databaseList as string
        ).split(","))
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
      } else {
        environmentList.push(...store.getters["environment/environmentList"]());
      }

      const newIssue = {
        ...newIssueTemplate.value.buildIssue({
          environmentList,
          databaseList,
          currentUser: currentUser.value,
        }),
        projectId: router.currentRoute.value.query.project
          ? parseInt(router.currentRoute.value.query.project as string)
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
        newIssue.assigneeId = parseInt(
          router.currentRoute.value.query.assignee as string
        );
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

    const create = props.issueSlug.toLowerCase() == "new";
    const state = reactive<LocalState>({
      create: create,
      newIssue: create ? buildNewIssue() : undefined,
    });

    watch(
      () => props.issueSlug,
      (cur, prev) => {
        state.create = cur.toLowerCase() == "new";
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

    const updateSql = (
      newSql: string,
      postUpdated: (updatedIssue: Issue) => void
    ) => {
      if (state.create) {
        state.newIssue!.sql = newSql;
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
      if (state.create) {
        state.newIssue!.rollbackSql = newSql;
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

    const updateSubscriberIdList = (newSubscriberIdList: PrincipalId[]) => {
      patchIssue({
        subscriberIdList: newSubscriberIdList,
      });
    };

    const updateCustomField = (field: InputField | OutputField, value: any) => {
      console.debug("updateCustomField", field.name, value);
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
      store
        .dispatch("issue/createIssue", state.newIssue)
        .then((createdIssue) => {
          // Use replace to omit the new issue url in the navigation history.
          router.replace(
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
          issueId: (issue.value as Issue).id,
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

    const currentPipelineType = computed((): PipelineType => {
      return pipelineType((issue.value as Issue).pipeline);
    });

    console.debug(currentPipelineType.value);

    const selectedStage = computed((): Stage => {
      const stageSlug = router.currentRoute.value.query.stage as string;
      if (stageSlug) {
        const index = indexFromSlug(stageSlug);
        return (issue.value as Issue).pipeline.stageList[index];
      }
      return activeStage((issue.value as Issue).pipeline);
    });

    const selectStageId = (stageId: StageId) => {
      const stageList = (issue.value as Issue).pipeline.stageList;
      const index = stageList.findIndex((item) => {
        return item.id == stageId;
      });
      router.replace({
        name: "workspace.issue.detail",
        query: {
          ...router.currentRoute.value.query,
          stage: stageSlug(stageList[index].name, index),
        },
      });
    };

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

    const allowEditSql = computed(() => {
      return state.create;
    });

    const showCancelBanner = computed(() => {
      return !state.create && (issue.value as Issue).status == "CANCELED";
    });

    const showSuccessBanner = computed(() => {
      return !state.create && (issue.value as Issue).status == "DONE";
    });

    const showPipelineFlowBar = computed(() => {
      return !state.create && currentPipelineType.value != "NO_PIPELINE";
    });

    const showIssueOutputPanel = computed(() => {
      return !state.create && issueTemplate.value.outputFieldList.length > 0;
    });

    const showIssueSqlPanel = computed(() => {
      return (
        issue.value.type == "bb.issue.general" ||
        issue.value.type == "bb.issue.db.schema.update"
      );
    });

    const showIssueRollbackSqlPanel = computed(() => {
      return issue.value.type == "bb.issue.db.schema.update";
    });

    return {
      state,
      issue,
      updateName,
      updateDescription,
      updateSql,
      updateRollbackSql,
      updateAssigneeId,
      updateSubscriberIdList,
      updateCustomField,
      doCreate,
      currentPipelineType,
      currentUser,
      issueTemplate,
      selectedStage,
      selectStageId,
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
    };
  },
};
</script>
