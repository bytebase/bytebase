import dayjs from "dayjs";
import { cloneDeep } from "lodash-es";
import { defineComponent, onMounted } from "vue";
import { GrantRequestContext, IssueCreate } from "@/types";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { provideIssueLogic, useCommonLogic, useIssueLogic } from "./index";
import { stringifyDatabaseResources } from "@/utils/issue/cel";

export default defineComponent({
  name: "GrantRequestModeProvider",
  setup() {
    const { issue, createIssue } = useIssueLogic();

    onMounted(() => {
      useProjectV1Store().fetchProjectList();
    });

    const doCreate = async () => {
      const currentUser = useCurrentUserV1();
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      const context: GrantRequestContext = {
        ...{
          databaseResources: [],
          expireDays: 7,
          maxRowCount: 1000,
          statement: "",
          exportFormat: "CSV",
        },
        ...(issueCreate.createContext as GrantRequestContext),
      };
      const expression: string[] = [];
      if (context.role === "QUERIER") {
        if (
          Array.isArray(context.databaseResources) &&
          context.databaseResources.length > 0
        ) {
          const cel = stringifyDatabaseResources(context.databaseResources);
          expression.push(cel);
        }
        if (context.expireDays > 0) {
          expression.push(
            `request.time < timestamp("${dayjs()
              .add(context.expireDays, "days")
              .toISOString()}")`
          );
        }
      } else if (context.role === "EXPORTER") {
        if (
          !Array.isArray(context.databaseResources) ||
          context.databaseResources.length === 0
        ) {
          throw "Exporter must have at least one database";
        }
        if (
          Array.isArray(context.databaseResources) &&
          context.databaseResources.length > 0
        ) {
          const cel = stringifyDatabaseResources(context.databaseResources);
          expression.push(cel);
        }
        if (context.expireDays > 0) {
          expression.push(
            `request.time < timestamp("${dayjs()
              .add(context.expireDays, "days")
              .toISOString()}")`
          );
        }
        expression.push(`request.statement == "${btoa(context.statement)}"`);
        expression.push(`request.row_limit == ${context.maxRowCount}`);
        expression.push(`request.export_format == "${context.exportFormat}"`);
      } else {
        throw "Invalid role";
      }

      const celExpressionString = expression.join(" && ");
      issueCreate.payload = {
        grantRequest: {
          role: `roles/${context.role}`,
          user: currentUser.value.name,
          condition: {
            expression: celExpressionString,
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
