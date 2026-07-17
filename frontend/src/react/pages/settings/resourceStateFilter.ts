import type { SearchParams } from "@/react/components/AdvancedSearch";
import { legacyStateSearchParams } from "@/react/hooks/useURLSearchParam";
import { State } from "@/types/proto-es/v1/common_pb";

export const defaultActiveStateSearchParams = (
  state: unknown
): SearchParams => {
  const legacy = legacyStateSearchParams(state);
  if (legacy.scopes.length > 0) {
    return legacy;
  }
  return { query: "", scopes: [{ id: "state", value: "ACTIVE" }] };
};

export const getResourceStateFilter = (
  stateFilterValue: string | undefined
): State | undefined => {
  if (stateFilterValue === "ACTIVE") {
    return State.ACTIVE;
  }
  if (stateFilterValue === "DELETED") {
    return State.DELETED;
  }
  return undefined;
};
