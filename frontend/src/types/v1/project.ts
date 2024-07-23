import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { IamPolicy } from "../proto/v1/iam_policy";
import { Project, Workflow } from "../proto/v1/project_service";

export const DEFAULT_PROJECT_UID = 1;
export const EMPTY_PROJECT_NAME = `projects/${EMPTY_ID}`;
export const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;
export const DEFAULT_PROJECT_NAME = "projects/default";

export interface ComposedProject extends Project {
  iamPolicy: IamPolicy;
}

export const emptyProject = (): ComposedProject => {
  return {
    ...Project.fromJSON({
      name: EMPTY_PROJECT_NAME,
      uid: String(EMPTY_ID),
      title: "",
      key: "",
      state: State.ACTIVE,
      workflow: Workflow.UI,
    }),
    iamPolicy: IamPolicy.fromJSON({}),
  };
};

export const unknownProject = (): ComposedProject => {
  return {
    ...emptyProject(),
    name: UNKNOWN_PROJECT_NAME,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown project>>",
  };
};

export const defaultProject = (): ComposedProject => {
  return {
    ...unknownProject(),
    name: DEFAULT_PROJECT_NAME,
    uid: String(DEFAULT_PROJECT_UID),
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
