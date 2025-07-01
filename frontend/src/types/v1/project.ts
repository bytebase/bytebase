import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto-es/v1/common_pb";
import type { IamPolicy } from "../proto-es/v1/iam_policy_pb";
import { IamPolicySchema } from "../proto-es/v1/iam_policy_pb";
import type { Project } from "../proto-es/v1/project_service_pb";
import { ProjectSchema } from "../proto-es/v1/project_service_pb";
import { create as createProto } from "@bufbuild/protobuf";

export const DEFAULT_PROJECT_UID = 1;
export const EMPTY_PROJECT_NAME = `projects/${EMPTY_ID}`;
export const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;
export const DEFAULT_PROJECT_NAME = "projects/default";

export interface ComposedProject extends Project {
  iamPolicy: IamPolicy;
}

export const emptyProject = (): ComposedProject => {
  return {
    ...createProto(ProjectSchema, {
      name: EMPTY_PROJECT_NAME,
      title: "",
      state: State.ACTIVE,
    }),
    iamPolicy: createProto(IamPolicySchema, {}),
  };
};

export const unknownProject = (): ComposedProject => {
  return {
    ...emptyProject(),
    name: UNKNOWN_PROJECT_NAME,
    title: "<<Unknown project>>",
  };
};

export const defaultProject = (): ComposedProject => {
  return {
    ...unknownProject(),
    name: DEFAULT_PROJECT_NAME,
    title: "Default project",
  };
};

export const isValidProjectName = (name: string | undefined) => {
  return (
    !!name &&
    name.startsWith("projects/") &&
    name !== EMPTY_PROJECT_NAME &&
    name !== UNKNOWN_PROJECT_NAME
  );
};
