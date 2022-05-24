import { IssueCreate, IssueType, UpdateSchemaGhostContext } from "@/types";
import {
  findDatabaseListByQuery,
  BuildNewIssueContext,
  VALIDATE_ONLY_SQL,
} from "../common";
import { IssueCreateHelper } from "./helper";

export const maybeBuildGhostIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate | undefined> => {
  const { route } = context;

  if (parseInt(route.query.ghost as string, 10) !== 1) {
    return undefined;
  }
  const issueType = route.query.template as IssueType;
  if (issueType !== "bb.issue.database.schema.update") {
    // Only available for schema updates.
    return undefined;
  }
  return buildNewGhostIssue(context);
};

const buildNewGhostIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate> => {
  const helper = new IssueCreateHelper(context);
  await helper.prepare();

  helper.issueCreate!.type = "bb.issue.database.schema.update.ghost";

  const databaseList = findDatabaseListByQuery(context);
  const createContext: UpdateSchemaGhostContext = {
    updateSchemaGhostDetailList: databaseList.map((db) => {
      return {
        databaseId: db.id,
        databaseName: db.name,
        statement: VALIDATE_ONLY_SQL,
        earliestAllowedTs: 0,
      };
    }),
  };

  helper.issueCreate!.createContext = createContext;

  await helper.validate();

  return helper.generate();
};
