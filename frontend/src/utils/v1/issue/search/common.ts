import dayjs from "dayjs";
import { cloneDeep, pullAt } from "lodash-es";
import { hasFeature } from "@/store";

// TimeRangeLimitForFreePlanInTs is the search time limit in ts format.
// should be 60 days.
export const TimeRangeLimitForFreePlanInTs = 60 * 24 * 60 * 60 * 1000;

export type SemanticIssueStatus = "OPEN" | "CLOSED";

export const UIIssueFilterScopeIdList = [
  "approver",
  "approval",
  "releaser",
] as const;
export type UIIssueFilterScopeId = typeof UIIssueFilterScopeIdList[number];

export const SearchScopeIdList = [
  "project",
  "instance",
  "database",
  "type",
  "creator",
  "assignee",
  "subscriber",
  "status",
  "created",
] as const;

export type SearchScopeId =
  | typeof SearchScopeIdList[number]
  | UIIssueFilterScopeId;

export type SearchScope = {
  id: SearchScopeId;
  value: string;
};

export interface SearchParams {
  query: string;
  scopes: SearchScope[];
}

export const isValidSearchScopeId = (id: string): id is SearchScopeId => {
  return (
    SearchScopeIdList.includes(id as any) ||
    UIIssueFilterScopeIdList.includes(id as any)
  );
};

export const buildSearchTextBySearchParams = (
  params: SearchParams | undefined
): string => {
  const parts: string[] = [];
  params?.scopes.forEach((scope) => {
    parts.push(`${scope.id}:${scope.value.trim()}`);
  });
  const query = (params?.query ?? "").trim();
  if (params?.query) {
    parts.push(query);
  }
  return parts.join(" ");
};

export const buildSearchParamsBySearchText = (text: string): SearchParams => {
  const params = defaultSearchParams();
  const segments = text.split(/\s+/g);
  const querySegments: string[] = [];

  for (let i = 0; i < segments.length; i++) {
    const seg = segments[i];
    const parts = seg.split(":");
    if (parts.length === 2 && isValidSearchScopeId(parts[0])) {
      params.scopes.push({
        id: parts[0],
        value: parts[1],
      });
    } else {
      querySegments.push(seg);
    }
  }
  params.query = querySegments.join(" ");
  params.scopes = params.scopes.filter((scope) => scope.id && scope.value);

  return params;
};

export const getValueFromSearchParams = (
  params: SearchParams,
  scopeId: SearchScopeId,
  prefix: string = "",
  validValues: readonly string[] = []
): string => {
  const scope = params.scopes.find((s) => s.id === scopeId);
  if (!scope) {
    return "";
  }
  const value = scope.value;
  if (validValues.length !== 0) {
    if (!validValues.includes(value)) {
      return "";
    }
  }
  return `${prefix}${scope.value}`;
};

export const getTsRangeFromSearchParams = (
  params: SearchParams,
  scopeId: SearchScopeId
) => {
  const scope = params.scopes.find((s) => s.id === scopeId);
  if (!scope) return undefined;
  const parts = scope.value.split(",");
  if (parts.length !== 2) return undefined;
  const range = [parseInt(parts[0], 10), parseInt(parts[1], 10)];
  return range;
};

/**
 * @param scope will remove `scope` from `params.scopes` if `scope.value` is empty.
 * @param mutate true to mutate `params`. false to create a deep cloned copy. Default to false.
 * @returns `params` itself or a deep-cloned copy.
 */
export const upsertScope = (
  params: SearchParams,
  scopes: SearchScope | SearchScope[],
  mutate = false
) => {
  const target = mutate ? params : cloneDeep(params);
  if (!Array.isArray(scopes)) {
    scopes = [scopes];
  }
  scopes.forEach((scope) => {
    const index = target.scopes.findIndex((s) => s.id === scope.id);
    if (index >= 0) {
      if (!scope.value) {
        pullAt(target.scopes, index);
      } else {
        target.scopes[index].value = scope.value;
      }
    } else {
      if (scope.value) {
        target.scopes.push(scope);
      }
    }
  });
  return target;
};

export const defaultSearchParams = (): SearchParams => {
  return {
    query: "",
    scopes: [],
  };
};

export const maybeApplyDefaultTsRange = (
  params: SearchParams,
  key: SearchScopeId,
  mutate = false
) => {
  if (hasFeature("bb.feature.issue-advanced-search")) {
    return mutate ? params : cloneDeep(params);
  }

  const endOfToday = dayjs().add(1, "day").endOf("day").valueOf();
  const begin = endOfToday - TimeRangeLimitForFreePlanInTs;
  return upsertScope(
    params,
    {
      id: key,
      value: `${begin},${endOfToday}`,
    },
    mutate
  );
};
