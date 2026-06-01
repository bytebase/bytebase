import { getProjectNameReleaseId } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "./const";

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
