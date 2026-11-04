import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import { Combobox } from "@/components/ui/combobox";
import { usePaginatedSelect } from "@/hooks/usePaginatedSelect";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { isDefaultProject, isValidProjectName } from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import {
  extractInstanceResourceName,
  getDefaultPagination,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";

export interface InstanceSelectProps {
  value: string;
  onChange: (value: string, instance: Instance | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  /** Only show instances with these engines */
  engines?: Engine[];
  /** Also offer this project's project instances, listed before workspace instances */
  project?: string;
}

// One page token walks the project's instances and then the workspace's:
// "<source index>:<token from that source's last page>".
const encodePageToken = (sourceIndex: number, token: string) =>
  `${sourceIndex}:${token}`;

const decodePageToken = (pageToken: string) => {
  const separator = pageToken.indexOf(":");
  return {
    sourceIndex: Number(pageToken.slice(0, separator)),
    token: pageToken.slice(separator + 1),
  };
};

export function InstanceSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  portal,
  engines,
  project,
}: InstanceSelectProps) {
  const { t } = useTranslation();
  // Stabilize engines array to avoid re-fetching on every render
  const enginesRef = useRef(engines);
  const stableEngines = useMemo(() => {
    const prev = enginesRef.current;
    if (
      prev &&
      engines &&
      prev.length === engines.length &&
      prev.every((e, i) => e === engines[i])
    ) {
      return prev;
    }
    enginesRef.current = engines;
    return engines;
  }, [engines]);

  const projectEntity = useProjectByName(project ?? "");
  const projectParent =
    isValidProjectName(project) &&
    !isDefaultProject(project) &&
    projectEntity.name === project &&
    hasProjectPermissionV2(projectEntity, "bb.instances.list")
      ? project
      : undefined;
  const canListWorkspaceInstances =
    hasWorkspacePermissionV2("bb.instances.list");

  const fetchPage = useCallback(
    async (query: string, pageToken: string) => {
      const parents: (string | undefined)[] = [];
      if (projectParent) parents.push(projectParent);
      if (canListWorkspaceInstances) parents.push(undefined);

      let { sourceIndex, token } = pageToken
        ? decodePageToken(pageToken)
        : { sourceIndex: 0, token: "" };
      const pageSize = getDefaultPagination();
      const items: Instance[] = [];
      while (sourceIndex < parents.length) {
        const result = await useAppStore.getState().fetchInstanceList({
          parent: parents[sourceIndex],
          pageSize,
          pageToken: token,
          filter: { query, engines: stableEngines },
        });
        items.push(...result.instances);
        if (result.nextPageToken) {
          return {
            items,
            nextPageToken: encodePageToken(sourceIndex, result.nextPageToken),
          };
        }
        sourceIndex += 1;
        token = "";
        // Fill a short page from the next source so a project with few
        // instances does not hide the workspace instances behind "load more".
        if (items.length >= pageSize && sourceIndex < parents.length) {
          return { items, nextPageToken: encodePageToken(sourceIndex, "") };
        }
      }
      return { items, nextPageToken: "" };
    },
    [stableEngines, projectParent, canListWorkspaceInstances]
  );
  const {
    items: instances,
    search,
    hasMore,
    loadingMore,
    loadMore,
  } = usePaginatedSelect({ fetchPage });

  useEffect(() => {
    search("");
  }, [search]);

  const handleChange = useCallback(
    (name: string) => {
      const inst = instances.find((i) => i.name === name);
      onChange(name, inst);
    },
    [instances, onChange]
  );

  return (
    <Combobox
      value={value}
      onChange={handleChange}
      placeholder={placeholder ?? t("common.instance")}
      noResultsText={t("common.no-data")}
      onSearch={search}
      hasMore={hasMore}
      loadingMore={loadingMore}
      onLoadMore={loadMore}
      disabled={disabled}
      className={className}
      portal={portal}
      renderValue={(opt) => {
        const inst = instances.find((i) => i.name === opt.value);
        if (!inst) return opt.label;
        return (
          <span className="flex items-center gap-1.5 truncate">
            <EngineIcon engine={inst.engine} className="h-4 w-4" />
            {inst.title}
          </span>
        );
      }}
      options={instances.map((inst) => ({
        value: inst.name,
        label: inst.title,
        description: extractInstanceResourceName(inst.name),
        render: (keyword) => (
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-1.5">
              {inst.environment && (
                <EnvironmentLabel environmentName={inst.environment} />
              )}
              <EngineIcon engine={inst.engine} className="h-4 w-4" />
              <HighlightLabelText text={inst.title} keyword={keyword} />
            </div>
            <HighlightLabelText
              text={extractInstanceResourceName(inst.name)}
              keyword={keyword}
              className="text-xs text-control-placeholder"
            />
          </div>
        ),
      }))}
    />
  );
}
