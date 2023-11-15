export type SemanticIssueStatus = "OPEN" | "CLOSED";

export const UIIssueFilterScopeIdList = [
  "approver",
  "approval",
  "releaser",
] as const;
export type UIIssueFilterScopeId = typeof UIIssueFilterScopeIdList[number];

export type SearchScopeId =
  | "project"
  | "instance"
  | "database"
  | "type"
  | "creator"
  | "assignee"
  | "subscriber"
  | "status"
  | "created"
  | UIIssueFilterScopeId;

export type SearchScope = {
  id: SearchScopeId;
  value: string;
};

export interface SearchParams {
  query: string;
  scopes: SearchScope[];
}

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
