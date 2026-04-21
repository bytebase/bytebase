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
}

export function ProjectSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  excludeDefault = true,
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
