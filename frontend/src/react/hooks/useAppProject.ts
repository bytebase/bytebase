import { useEffect, useMemo } from "react";
import { isValidProjectName } from "@/react/lib/resourceName";
import { useAppStore } from "@/react/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

/**
 * Reactively reads a project from the Zustand app store, self-fetching
 * when the current route hasn't preloaded it (the SQL editor route does
 * not mount the dashboard shells that hydrate the app store). Returns the
 * Pinia-compatible fallback (`unknownProject` / default project) so
 * callers can read `.title` etc. without null checks — mirroring the
 * legacy `useProjectV1Store().getProjectByName`.
 */
export const useAppProject = (name: string): Project => {
  const getProjectByName = useAppStore((s) => s.getProjectByName);
  const fetchProject = useAppStore((s) => s.fetchProject);
  // Subscribe to the specific cache entry so the value recomputes once
  // the project resolves. Selecting the raw entry (not the fallback)
  // keeps the snapshot stable for `useSyncExternalStore`.
  const cached = useAppStore((s) => s.projectsByName[name]);
  useEffect(() => {
    if (isValidProjectName(name)) {
      void fetchProject(name);
    }
  }, [fetchProject, name]);
  return useMemo(
    () => getProjectByName(name),
    [getProjectByName, name, cached]
  );
};
