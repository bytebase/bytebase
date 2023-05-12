import {
  GrantRequestContext,
  IssueCreate,
  IssueType,
  ProjectRoleTypeExporter,
  ProjectRoleTypeQuerier,
} from "@/types";
import { BuildNewIssueContext } from "../common";
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
  if (role !== ProjectRoleTypeExporter && role !== ProjectRoleTypeQuerier) {
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
  return issueCreate;
};
