import { useMemo } from "react";
import { useAppStore } from "@/react/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

/**
 * Reactively resolve a project by resource name from the app store, always
 * returning a non-null `Project` (the store synthesizes the unknown/default
 * placeholder on a cache miss).
 *
 * `getProjectByName` returns a FRESH placeholder object on every cache miss, so
 * it must NOT be used directly as a Zustand selector — `Object.is` would never
 * match and the component would re-render forever. Instead we subscribe to the
 * stable `projectsByName` map and recompute through `getProjectByName` only
 * when the map (or the name) changes, yielding a stable reference per render.
 */
export function useProjectByName(name: string): Project {
  const projectsByName = useAppStore((s) => s.projectsByName);
  return useMemo(
    () => useAppStore.getState().getProjectByName(name),
    [projectsByName, name]
  );
}
