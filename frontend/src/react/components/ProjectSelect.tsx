import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Combobox } from "@/react/components/ui/combobox";
import { useProjectV1Store } from "@/store";
import { isValidProjectName } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, getDefaultPagination } from "@/utils";

export interface ProjectSelectProps {
  value: string;
  onChange: (value: string, project: Project | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  /** When true, the workspace's default project is excluded. Defaults to true. */
  excludeDefault?: boolean;
  /** Optional rich empty-state node — used by surfaces that need to
   *  surface workspace-level guidance (e.g. AsidePanel's "not a member
   *  of any projects" + "go to create" prompt). Falls back to the
   *  generic "no data" string when omitted. */
  emptyContent?: ReactNode;
  /** Render the dropdown via a portal so it isn't clipped by an
   *  `overflow:hidden` ancestor (e.g. inside a Sheet body that scopes
   *  its own scroll region). */
  portal?: boolean;
}

export function ProjectSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  excludeDefault = true,
  emptyContent,
  portal,
}: ProjectSelectProps) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const [projects, setProjects] = useState<Project[]>([]);
  // Projects referenced by `value` that aren't in the paged fetch.
  // Mirrors Vue's `additionalOptions` so deep-linked URLs (e.g.
  // `/sql-editor/projects/foo/...`) show the correct project label
  // when the user lands on the page before the matching page-1 fetch
  // returns it.
  const [additionalProjects, setAdditionalProjects] = useState<Project[]>([]);

  const fetchProjects = useCallback(
    (query: string) => {
      projectStore
        .fetchProjectList({
          filter: { query, excludeDefault },
          pageSize: getDefaultPagination(),
        })
        .then(({ projects: result }) => setProjects(result));
    },
    [projectStore, excludeDefault]
  );

  useEffect(() => {
    fetchProjects("");
  }, [fetchProjects]);

  // Hydrate the selected project so the trigger always renders its
  // label, even before the paged list loads or when the project is
  // outside the first page window.
  useEffect(() => {
    if (!isValidProjectName(value)) return;
    let cancelled = false;
    void projectStore.getOrFetchProjectByName(value).then((p) => {
      if (cancelled) return;
      if (!isValidProjectName(p.name)) return;
      setAdditionalProjects((prev) =>
        prev.some((existing) => existing.name === p.name) ? prev : [...prev, p]
      );
    });
    return () => {
      cancelled = true;
    };
  }, [value, projectStore]);

  const allProjects = useMemo(() => {
    const seen = new Set<string>();
    const out: Project[] = [];
    for (const p of [...projects, ...additionalProjects]) {
      if (seen.has(p.name)) continue;
      seen.add(p.name);
      out.push(p);
    }
    return out;
  }, [projects, additionalProjects]);

  const handleChange = useCallback(
    (name: string) => {
      const proj = allProjects.find((p) => p.name === name);
      onChange(name, proj);
    },
    [allProjects, onChange]
  );

  return (
    <Combobox
      value={value}
      onChange={handleChange}
      placeholder={placeholder ?? t("common.project")}
      noResultsText={t("common.no-data")}
      noResultsContent={emptyContent}
      onSearch={fetchProjects}
      disabled={disabled}
      className={className}
      portal={portal}
      options={allProjects.map((p) => ({
        value: p.name,
        label: p.title,
        description: extractProjectResourceName(p.name),
      }))}
    />
  );
}
