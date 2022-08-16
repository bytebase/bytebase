import { Ref } from "vue";
import { TemplateType } from "@/plugins";
import {
  Issue,
  IssueCreate,
  Principal,
  SYSTEM_BOT_ID,
  TaskDatabaseSchemaUpdatePayload,
  UNKNOWN_ID,
} from "@/types";
import {
  useActuatorStore,
  useCurrentUser,
  useIssueStore,
  useInstanceStore,
} from "@/store";
import { BuildNewIssueContext, VALIDATE_ONLY_SQL } from "../common";

export class IssueCreateHelper {
  issueCreate: IssueCreate | null;
  issue: Issue | null;
  context: BuildNewIssueContext;
  issueStore: ReturnType<typeof useIssueStore>;
  intanceStore: ReturnType<typeof useInstanceStore>;
  currentUser: Ref<Principal>;

  constructor(context: BuildNewIssueContext) {
    this.issueCreate = null;
    this.issue = null;
    this.context = context;
    this.issueStore = useIssueStore();
    this.intanceStore = useInstanceStore();
    this.currentUser = useCurrentUser();
  }

  async prepare(): Promise<IssueCreate> {
    const { context, currentUser } = this;
    const { route } = context;
    const actuatorStore = useActuatorStore();

    const baseTemplate = context.template.value.buildIssue({
      environmentList: [],
      approvalPolicyList: [],
      databaseList: [],
      currentUser: currentUser.value,
    });

    const templateType = route.query.template as TemplateType;

    const issueCreate: IssueCreate = {
      projectId: parseInt(route.query.project as string) || UNKNOWN_ID,
      name: (route.query.name as string) || baseTemplate.name,
      type:
        templateType === "bb.issue.database.schema.baseline"
          ? "bb.issue.database.schema.update" // use schema.update to establish baseline
          : templateType,
      description: baseTemplate.description,
      // validateOnly does not support assigneeId=-1
      // so we need to specify here
      // but will reset this to -1 if route.query.assignee is empty
      assigneeId: SYSTEM_BOT_ID,
      createContext: {},
      payload: {},
    };

    // For demo mode, we assign the issue to the current user, so it can also experience the assignee user flow.
    if (actuatorStore.isDemo) {
      issueCreate.assigneeId = currentUser.value.id;
    }
    if (route.query.name) {
      issueCreate.name = route.query.name as string;
    }
    if (route.query.description) {
      issueCreate.description = route.query.description as string;
    }
    if (route.query.assignee) {
      issueCreate.assigneeId = parseInt(route.query.assignee as string);
    }

    this.issueCreate = issueCreate;

    return issueCreate;
  }

  async validate(): Promise<[IssueCreate, Issue]> {
    const { issueStore } = this;
    const issue = await issueStore.validateIssue(this.issueCreate!);

    this.issue = issue;

    return [this.issueCreate!, this.issue];
  }

  async generate(): Promise<IssueCreate> {
    const { route, template } = this.context;
    const issueCreate = this.issueCreate!;
    const issue = this.issue!;

    if (!route.query.assignee) {
      issueCreate.assigneeId =
        parseInt(route.query.assignee as string) || UNKNOWN_ID;
    }

    // copy the generated pipeline to issueCreate
    // issueCreate is an editable object for the whole issue UI
    issueCreate.pipeline = {
      name: issue.pipeline.name,
      stageList: issue.pipeline.stageList.map((stage) => ({
        name: stage.name,
        environmentId: stage.environment.id,
        taskList: stage.taskList.map((task) => {
          const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
          // if we are using VALIDATE_ONLY_SQL, set it back to empty
          // otherwise keep it as-is
          const statement =
            payload.statement === VALIDATE_ONLY_SQL ? "" : payload.statement;
          return {
            name: task.name,
            status: task.status,
            type: task.type,
            instanceId: task.instance.id,
            databaseId: task.database?.id,
            databaseName: task.database?.name,
            migrationType: payload.migrationType,
            statement,
            earliestAllowedTs: task.earliestAllowedTs,
          };
        }),
      })),
    };

    // cleanup input fields, not used yet.
    for (const field of template.value.inputFieldList) {
      const value = route.query[field.slug] as string;
      if (value) {
        if (field.type == "Boolean") {
          // false if "0" or "false", true otherwise
          const bool = value !== "0" && value.toLowerCase() !== "false";
          issueCreate.payload[field.id] = bool;
        } else {
          issueCreate.payload[field.id] = value;
        }
      }
    }

    return issueCreate;
  }
}
