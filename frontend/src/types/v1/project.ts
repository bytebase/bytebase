import { create as createProto } from "@bufbuild/protobuf";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { Project } from "../proto-es/v1/project_service_pb";
import { ProjectSchema } from "../proto-es/v1/project_service_pb";

export const DEFAULT_PROJECT_UID = 1;
export const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;
// Default project resource_id is "default-{workspaceID}", so we match by prefix.
const DEFAULT_PROJECT_PREFIX = "projects/default-";

// Check if a project name is the default project for the given workspace.
// workspaceResourceName is "workspaces/{workspaceID}".
export const isDefaultProject = (
  name: string,
  workspaceResourceName: string
): boolean => {
  return name === getDefaultProjectName(workspaceResourceName);
};

// Derive the default project name from the workspace resource name.
// workspaceResourceName is "workspaces/{workspaceID}".
export const getDefaultProjectName = (
  workspaceResourceName: string
): string => {
  const id = workspaceResourceName.replace(/^workspaces\//, "");
  return `${DEFAULT_PROJECT_PREFIX}${id}`;
};

export const unknownProject = (): Project => {
  return createProto(ProjectSchema, {
    name: UNKNOWN_PROJECT_NAME,
    state: State.ACTIVE,
    enforceIssueTitle: true,
    enforceSqlReview: true,
    requireIssueApproval: true,
    requirePlanCheckNoError: true,
    allowRequestRole: true,
  });
};

export const defaultProject = (name: string): Project => {
  return {
    ...unknownProject(),
    name,
    title: "Default project",
  };
};

export const isValidProjectName = (name: string | undefined) => {
  return (
    !!name && name.startsWith("projects/") && name !== UNKNOWN_PROJECT_NAME
  );
};
