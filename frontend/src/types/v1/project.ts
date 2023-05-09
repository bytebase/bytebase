import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { IamPolicy, Project } from "../proto/v1/project_service";

export const DEFAULT_PROJECT_V1_NAME = "projects/default";

export interface ComposedProject extends Project {
  iamPolicy: IamPolicy;
}

export const emptyProject = (): ComposedProject => {
  return {
    ...Project.fromJSON({
      name: `projects/${EMPTY_ID}`,
      uid: EMPTY_ID,
    }),
    iamPolicy: IamPolicy.fromJSON({}),
  };
};

export const unknownProject = (): ComposedProject => {
  return {
    ...Project.fromJSON({
      name: `projects/${UNKNOWN_ID}`,
      uid: UNKNOWN_ID,
      title: "<<Unknown project>>",
    }),
    iamPolicy: IamPolicy.fromJSON({}),
  };
};
