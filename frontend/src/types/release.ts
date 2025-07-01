import { create } from "@bufbuild/protobuf";
import { getProjectNameReleaseId } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "./const";
import type { Release } from "./proto-es/v1/release_service_pb";
import { ReleaseSchema } from "./proto-es/v1/release_service_pb";
import type { User } from "./proto-es/v1/user_service_pb";
import { unknownUser } from "./v1";
import {
  UNKNOWN_PROJECT_NAME,
  unknownProject,
  type ComposedProject,
} from "./v1/project";

export interface ComposedRelease extends Release {
  // Format: projects/{project}
  project: string;
  projectEntity: ComposedProject;
  creatorEntity: User;
}

export const UNKNOWN_RELEASE_NAME = `${UNKNOWN_PROJECT_NAME}/releases/${UNKNOWN_ID}`;

export const unknownRelease = (): ComposedRelease => {
  const projectEntity = unknownProject();
  const release = create(ReleaseSchema, {
    name: `${projectEntity.name}/releases/${UNKNOWN_ID}`,
  });
  return {
    ...release,
    project: projectEntity.name,
    projectEntity,
    creatorEntity: unknownUser(),
  };
};

export const isValidReleaseName = (name: any): name is string => {
  if (typeof name !== "string") return false;
  const [projectName, releaseName] = getProjectNameReleaseId(name);
  return Boolean(
    projectName &&
      projectName !== String(UNKNOWN_ID) &&
      releaseName &&
      releaseName !== String(UNKNOWN_ID)
  );
};
