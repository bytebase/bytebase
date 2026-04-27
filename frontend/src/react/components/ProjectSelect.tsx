import type { ReactNode } from "react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Combobox } from "@/react/components/ui/combobox";
import { useProjectV1Store } from "@/store";
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
}

export function ProjectSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  excludeDefault = true,
  emptyContent,
}: ProjectSelectProps) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const [projects, setProjects] = useState<Project[]>([]);

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

  const handleChange = useCallback(
    (name: string) => {
      const proj = projects.find((p) => p.name === name);
      onChange(name, proj);
    },
    [projects, onChange]
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
      options={projects.map((p) => ({
        value: p.name,
        label: p.title,
        description: extractProjectResourceName(p.name),
      }))}
    />
  );
}
