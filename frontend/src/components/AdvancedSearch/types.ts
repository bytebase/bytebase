import type { RenderFunction } from "vue";
import type { SearchScopeId } from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  options?: ValueOption[];
  description?: string;
  allowMultiple?: boolean;
  search?: (op: {
    keyword: string;
    nextPageToken?: string;
  }) => Promise<{ nextPageToken?: string; options: ValueOption[] }>;
};

export type ValueOption = {
  value: string;
  keywords: string[];
  custom?: boolean;
  render?: RenderFunction;
};
