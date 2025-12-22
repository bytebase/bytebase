import { create } from "@bufbuild/protobuf";
import { getProjectNameReleaseId } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "./const";
import type { Release } from "./proto-es/v1/release_service_pb";
import { ReleaseSchema } from "./proto-es/v1/release_service_pb";
import { UNKNOWN_PROJECT_NAME } from "./v1/project";

export const UNKNOWN_RELEASE_NAME = `${UNKNOWN_PROJECT_NAME}/releases/${UNKNOWN_ID}`;

export const unknownRelease = (): Release => {
  return create(ReleaseSchema, {
    name: UNKNOWN_RELEASE_NAME,
  });
};

export const isValidReleaseName = (name: unknown): name is string => {
  if (typeof name !== "string") return false;
  const [projectName, releaseName] = getProjectNameReleaseId(name);
  return Boolean(
    projectName &&
      projectName !== String(UNKNOWN_ID) &&
      releaseName &&
      releaseName !== String(UNKNOWN_ID)
  );
};
