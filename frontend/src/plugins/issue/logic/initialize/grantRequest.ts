import { head } from "lodash-es";
import {
  GrantRequestContext,
  IssueCreate,
  IssueType,
  PresetRoleType,
} from "@/types";
import { extractRoleResourceName } from "@/utils";
import { BuildNewIssueContext, findDatabaseListByQuery } from "../common";
import { IssueCreateHelper } from "./helper";

export const maybeBuildGrantRequestIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate | undefined> => {
  const { route } = context;
  const issueType = route.query.template as IssueType;
  const role = route.query.role;
  if (issueType !== "bb.issue.grant.request") {
    return undefined;
  }
  // We only allow to create grant request issue for exporter and querier roles.
  const exporterRole = extractRoleResourceName(PresetRoleType.EXPORTER);
  const querierRole = extractRoleResourceName(PresetRoleType.QUERIER);
  if (role !== exporterRole && role !== querierRole) {
    return undefined;
  }

  return buildNewGrantRequestIssue(context);
};

const buildNewGrantRequestIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate> => {
  const { route } = context;
  const helper = new IssueCreateHelper(context);
  await helper.prepare();
  const issueCreate = await helper.generate();
  const role = route.query.role as string;
  (issueCreate.createContext as GrantRequestContext).role = role as any;
  const project = route.query.project as string;
  if (project) {
    issueCreate.projectId = Number(project);
  }
  if (role === "EXPORTER") {
    const createContext = issueCreate.createContext as GrantRequestContext;
    const statement = (route.query.sql as string) || "";
    createContext.statement = statement;
    const databaseList = findDatabaseListByQuery(context);
    const database = head(databaseList);
    if (database) {
      createContext.databaseResources = [
        {
          databaseId: database.uid,
          databaseName: database.name,
        },
      ];
    }
  }
  return issueCreate;
};
