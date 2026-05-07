import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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

  // Read `excludeDefault` from a ref inside the callback so the
  // callback identity stays stable across re-renders. Without this, any
  // parent re-render that re-evaluates the `excludeDefault` prop
  // expression (e.g. the SQL Editor's `AsidePanel` re-rendering on
  // tab.connection mutations) recreates the callback, which causes the
  // mount-`useEffect` below to re-fire and hit `ListProjects` on every
  // re-render. The boolean value itself is read live each call, so
  // updates still take effect — just without re-running the effect.
  const excludeDefaultRef = useRef(excludeDefault);
  excludeDefaultRef.current = excludeDefault;

  const fetchProjects = useCallback(
    (query: string) => {
      projectStore
        .fetchProjectList({
          filter: { query, excludeDefault: excludeDefaultRef.current },
          pageSize: getDefaultPagination(),
        })
        .then(({ projects: result }) => setProjects(result));
    },
    [projectStore]
  );

  // Fetch the first page on mount, and re-fetch only when
  // `excludeDefault` actually flips (rare — typically once after
  // permissions hydrate). `fetchProjects` is stable so this effect
  // never re-fires from parent re-renders alone.
  useEffect(() => {
    fetchProjects("");
  }, [excludeDefault, fetchProjects]);

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
