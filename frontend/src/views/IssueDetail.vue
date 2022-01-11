<template>
  <IssueDetailLayout
    v-if="issue"
    :issue="issue"
    :create="state.create"
    @status-changed="onStatusChanged"
  />
  <div v-else class="w-full h-full flex justify-center items-center">
    <NSpin />
  </div>
</template>

<script lang="ts">
import {
  computed,
  onMounted,
  onUnmounted,
  watch,
  watchEffect,
  reactive,
  ref,
  defineComponent,
} from "vue";
import { useStore } from "vuex";
import { useRoute, useRouter } from "vue-router";
import { idFromSlug } from "../utils";
import { IssueDetailLayout } from "../components/Issue";
import {
  UNKNOWN_ID,
  Issue,
  IssueCreate,
  IssueType,
  Database,
  Environment,
  TaskDatabaseSchemaUpdatePayload,
  NORMAL_POLL_INTERVAL,
  POLL_JITTER,
  POST_CHANGE_POLL_INTERVAL,
  Project,
  Policy,
  unknown,
} from "../types";
import {
  defaulTemplate as defaultTemplate,
  templateForType,
  IssueTemplate,
} from "../plugins";
import { isEmpty } from "lodash-es";
import { NSpin } from "naive-ui";

interface LocalState {
  // Needs to maintain this state and set it to false manually after creating the issue.
  // router.push won't trigger the reload because new and existing issue shares
  // the same component.
  create: boolean;
  newIssue?: IssueCreate;
  // Timer tracking the issue poller, we need this to cancel the outstanding one when needed.
  pollIssueTimer?: ReturnType<typeof setTimeout>;
}

export default defineComponent({
  name: "IssueDetail",
  components: {
    IssueDetailLayout,
    NSpin,
  },
  props: {
    issueSlug: {
      required: true,
      type: String,
    },
  },

  setup(props) {
    const store = useStore();
    const router = useRouter();
    const route = useRoute();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    let newIssueTemplate = ref<IssueTemplate>(defaultTemplate());

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
        newIssueTemplate.value = defaultTemplate();
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

    const state = reactive<LocalState>({
      create: props.issueSlug.toLowerCase() == "new",
      newIssue: undefined,
    });

    const issue = computed((): Issue | IssueCreate => {
      return state.create
        ? state.newIssue
        : store.getters["issue/issueById"](idFromSlug(props.issueSlug));
    });

    const buildNewTenantSchemaUpdateIssue = async (
      project: Project
    ): Promise<IssueCreate> => {
      const baseTemplate = newIssueTemplate.value.buildIssue({
        environmentList: [],
        approvalPolicyList: [],
        databaseList: [],
        currentUser: currentUser.value,
      });
      const issueCreate: IssueCreate = {
        projectId: project.id,
        name: (route.query.name as string) || baseTemplate.name,
        type: "bb.issue.database.schema.update",
        description: baseTemplate.description,
        assigneeId: currentUser.value.id,
        createContext: {
          migrationType: "MIGRATE",
          updateSchemaDetailList: [
            {
              databaseName: route.query.databaseName,
              statement: "/* YOUR_SQL_HERE */",
              rollbackStatement: "",
            },
          ],
        },
        payload: {},
      };
      const issue: Issue = await store.dispatch(
        "issue/validateIssue",
        issueCreate
      );

      issueCreate.assigneeId = baseTemplate.assigneeId;
      issueCreate.pipeline = {
        name: issue.pipeline.name,
        stageList: issue.pipeline.stageList.map((stage) => ({
          name: stage.name,
          environmentId: stage.environment.id,
          taskList: stage.taskList.map((task) => {
            const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
            return {
              name: task.name,
              status: task.status,
              type: task.type,
              instanceId: task.instance.id,
              databaseId: task.database?.id,
              migrationType: payload.migrationType,
              statement: payload.statement,
              rollbackStatement: payload.rollbackStatement,
              earliestAllowedTs: task.earliestAllowedTs,
            };
          }),
        })),
      };

      return issueCreate;
    };

    const maybeBuildTenantDeployIssue = async (): Promise<
      IssueCreate | false
    > => {
      if (route.query.mode !== "tenant") {
        return false;
      }

      const projectId = route.query.project
        ? parseInt(route.query.project as string)
        : UNKNOWN_ID;
      let project = unknown("PROJECT") as Project;
      if (projectId !== UNKNOWN_ID) {
        project = await store.dispatch("project/fetchProjectById", projectId);
      }
      const issueType = route.query.template as IssueType;
      if (
        project.tenantMode === "TENANT" &&
        issueType === "bb.issue.database.schema.update"
      ) {
        return buildNewTenantSchemaUpdateIssue(project);
      }
      return false;
    };

    const buildNewIssue = async (): Promise<IssueCreate | undefined> => {
      const tenant = await maybeBuildTenantDeployIssue();
      if (tenant) {
        return tenant;
      }

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
          title =
            "INVALID STATE, please create rollback from the original issue.";
        } else {
          if (rollbackIssue.type != "bb.issue.database.schema.update") {
            title =
              "INVALID STATE, only support to rollback update schema issue.";
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
          for (
            let i = rollbackIssue.pipeline.stageList.length - 1;
            i >= 0;
            i--
          ) {
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
              ...defaultTemplate().buildIssue({
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
            ...defaultTemplate().buildIssue({
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
            databaseList.push(
              store.getters["database/databaseById"](databaseId)
            );
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
          environmentList.push(
            ...store.getters["environment/environmentList"]()
          );
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

    onMounted(async () => {
      if (!state.create) {
        pollOnCreateStateChange();
      } else {
        state.newIssue = await buildNewIssue();
      }
    });

    onUnmounted(() => {
      if (state.pollIssueTimer) {
        clearInterval(state.pollIssueTimer);
      }
    });

    watch(
      () => props.issueSlug,
      async (cur) => {
        const oldCreate = state.create;
        state.create = cur.toLowerCase() == "new";
        if (!state.create && oldCreate) {
          pollOnCreateStateChange();
        } else if (state.create && !oldCreate) {
          clearInterval(state.pollIssueTimer as any);
          state.newIssue = await buildNewIssue();
        }
      }
    );

    const onStatusChanged = (eager: boolean) => {
      pollIssue(eager ? POST_CHANGE_POLL_INTERVAL : NORMAL_POLL_INTERVAL);
    };

    return {
      state,
      issue,
      onStatusChanged,
    };
  },
});
</script>
