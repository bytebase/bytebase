import { cloneDeep, pullAt } from "lodash-es";
import { SearchParams, SearchScope } from "@/utils";

/**
 * @param scope will remove `scope` from `params.scopes` if `scope.value` is empty.
 * @param mutate true to mutate `params`. false to create a deep cloned copy. Default to false.
 * @returns `params` itself or a deep-cloned copy.
 */
export const upsertScope = (
  params: SearchParams,
  scope: SearchScope,
  mutate = false
) => {
  const target = mutate ? params : cloneDeep(params);
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
  return target;
};

export const defaultSearchParams = (): SearchParams => {
  return {
    query: "",
    scopes: [],
  };
};
