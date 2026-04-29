import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import type {
  ScopeOption,
  ValueOption,
} from "@/react/components/AdvancedSearch";
import { EngineIcon } from "@/react/components/EngineIcon";
import { useInstanceV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  getDefaultPagination,
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
  supportOptionIdList: SearchScopeId[]
): ScopeOption[] {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();

  const searchInstance = useCallback(
    async (keyword: string): Promise<ValueOption[]> => {
      const resp = await instanceStore.fetchInstanceList({
        pageToken: undefined,
        pageSize: getDefaultPagination(),
        filter: { query: keyword },
        silent: true,
      });
      return resp.instances.map<ValueOption>((ins) => {
        const name = extractInstanceResourceName(ins.name);
        const env = extractEnvironmentResourceName(ins.environment ?? "");
        return {
          value: name,
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
    [instanceStore]
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
