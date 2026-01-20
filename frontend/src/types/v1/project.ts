import { create as createProto } from "@bufbuild/protobuf";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { Project } from "../proto-es/v1/project_service_pb";
import { ProjectSchema } from "../proto-es/v1/project_service_pb";

export const DEFAULT_PROJECT_UID = 1;
export const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;
export const DEFAULT_PROJECT_NAME = "projects/default";

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

export const defaultProject = (): Project => {
  return {
    ...unknownProject(),
    name: DEFAULT_PROJECT_NAME,
    title: "Default project",
  };
};

export const isValidProjectName = (name: string | undefined) => {
  return (
    !!name && name.startsWith("projects/") && name !== UNKNOWN_PROJECT_NAME
  );
};
