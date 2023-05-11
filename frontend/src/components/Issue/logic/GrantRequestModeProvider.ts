import { cloneDeep } from "lodash-es";
import { defineComponent } from "vue";
import { GrantRequestContext, IssueCreate } from "@/types";
import { useCurrentUserV1, useDatabaseStore } from "@/store";
import { provideIssueLogic, useCommonLogic, useIssueLogic } from "./index";

export default defineComponent({
  name: "GrantRequestModeProvider",
  setup() {
    const { issue, createIssue } = useIssueLogic();
    const databaseStore = useDatabaseStore();

    const doCreate = async () => {
      const currentUser = useCurrentUserV1();
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // Transform create context into CEL condition.
      const context = issueCreate.createContext as GrantRequestContext;
      const expression: string[] = [];
      if (context.role === "QUERIER") {
        if (Array.isArray(context.databases) && context.databases.length > 0) {
          const databaseNames = [];
          for (const databaseId of context.databases) {
            const database = await databaseStore.getOrFetchDatabaseById(
              databaseId
            );
            databaseNames.push(
              `/v1/environments/${database.instance.environment.resourceId}/instances/${database.instance.resourceId}/databases/${database.id}`
            );
          }
          expression.push(`databases in ${JSON.stringify(databaseNames)}`);
        }
        expression.push(
          `expired_time < timestamp("${new Date(
            Date.now() + context.expireDays * 1000 * 60 * 60 * 24
          ).toISOString()}")`
        );
      } else if (context.role === "EXPORTER") {
        if (
          !Array.isArray(context.databases) ||
          context.databases.length === 0
        ) {
          throw "Exporter must have at least one database";
        }
        const databaseId = context.databases[0];
        const database = await databaseStore.getOrFetchDatabaseById(databaseId);
        const databaseNames = [];
        databaseNames.push(
          `/v1/environments/${database.instance.environment.resourceId}/instances/${database.instance.resourceId}/databases/${database.id}`
        );
        expression.push(`databases in ${JSON.stringify(databaseNames)}`);
        expression.push(`statement == '${btoa(context.statement)}'`);
        expression.push(`max_row_count == ${context.maxRowCount}`);
        expression.push(`export_format == '${context.exportFormat}'`);
      } else {
        throw "Invalid role";
      }

      issueCreate.payload = {
        grantRequest: {
          role: `roles/${context.role}`,
          user: currentUser.value.name,
          condition: {
            expression: expression.join(" && "),
          },
        },
      };
      issueCreate.createContext = {};

      createIssue(issueCreate);
    };

    const logic = {
      ...useCommonLogic(),
      doCreate,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
