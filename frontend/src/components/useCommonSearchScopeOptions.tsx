import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { ScopeOption, ValueOption } from "@/components/AdvancedSearch";
import { EngineIcon } from "@/components/EngineIcon";
import { useAppStore } from "@/stores/app";
import { isDefaultProject, isValidProjectName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  getDefaultPagination,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  type SearchScopeId,
  supportedEngineV1List,
} from "@/utils";

/**
 * React port of `useCommonSearchScopeOptions` from
 * frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions.ts.
 * Initial scope: what SQL Editor ConnectionPane uses — `instance`, `label`,
 * and `engine`. Other scope ids (project, environment, state, etc.) remain
 * to be ported when a consumer needs them; the `scopeCreators` switch
 * short-circuits unknown ids rather than inventing a scope.
 */
export function useCommonSearchScopeOptions(
  supportOptionIdList: SearchScopeId[],
  project?: Project
): ScopeOption[] {
  const { t } = useTranslation();

  const searchInstance = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const params = {
        pageToken: undefined,
        pageSize: getDefaultPagination(),
        filter: { query: keyword },
        silent: true,
      };
      const results = await Promise.all([
        hasWorkspacePermissionV2("bb.instances.list")
          ? useAppStore.getState().fetchInstanceList(params)
          : Promise.resolve({ instances: [], nextPageToken: "" }),
        project &&
        isValidProjectName(project.name) &&
        !isDefaultProject(project.name) &&
        hasProjectPermissionV2(project, "bb.instances.list")
          ? useAppStore
              .getState()
              .fetchInstanceList({ ...params, parent: project.name })
          : Promise.resolve({ instances: [], nextPageToken: "" }),
      ]);
      const instances = [
        ...new Map(
          results
            .flatMap((result) => result.instances)
            .map((instance) => [instance.name, instance])
        ).values(),
      ];
      return instances.map<ValueOption>((ins) => {
        const name = extractInstanceResourceName(ins.name);
        const env = extractEnvironmentResourceName(ins.environment ?? "");
        return {
          value: ins.name,
          keywords: [name, ins.title, String(ins.engine), env],
          render: () => (
            <span className="flex items-center gap-x-1">
              <EngineIcon engine={ins.engine} className="size-4" />
              <span className="truncate">{ins.title}</span>
              {env && (
                <span className="text-control-light text-xs">({env})</span>
              )}
            </span>
          ),
        };
      });
    },
    [project]
  );

  return useMemo(() => {
    const scopes: ScopeOption[] = [];
    for (const id of supportOptionIdList) {
      switch (id) {
        case "instance":
          scopes.push({
            id: "instance",
            title: t("issue.advanced-search.scope.instance.title"),
            description: t("issue.advanced-search.scope.instance.description"),
            onSearch: searchInstance,
          });
          break;
        case "label":
          scopes.push({
            id: "label",
            title: t("common.labels"),
            description: t("issue.advanced-search.scope.label.description"),
            allowMultiple: true,
          });
          break;
        case "engine":
          scopes.push({
            id: "engine",
            title: t("issue.advanced-search.scope.engine.title"),
            description: t("issue.advanced-search.scope.engine.description"),
            allowMultiple: true,
            options: supportedEngineV1List().map((engine) => ({
              value: Engine[engine],
              keywords: [Engine[engine].toLowerCase()],
              render: () => <span>{Engine[engine]}</span>,
            })),
          });
          break;
        default:
          // Unknown/unsupported scope id. Silently drop rather than
          // inventing a scope — opens a clear TODO for whoever adds the
          // next consumer.
          break;
      }
    }
    return scopes;
  }, [supportOptionIdList, t, searchInstance]);
}
