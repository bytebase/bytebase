import { create as createProto } from "@bufbuild/protobuf";
import { useActuatorV1Store } from "@/store/modules/v1/actuator";
import { UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { Project } from "../proto-es/v1/project_service_pb";
import { ProjectSchema } from "../proto-es/v1/project_service_pb";

export const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;

// Check if a project name is the default project.
// Reads the default project name from the actuator store.
export const isDefaultProject = (name: string): boolean => {
  const defaultProject = useActuatorV1Store().serverInfo?.defaultProject;
  return !!defaultProject && name === defaultProject;
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
