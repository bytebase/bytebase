import type { RenderFunction } from "vue";
import type { SearchScopeId } from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  options: ValueOption[];
  description?: string;
  allowMultiple?: boolean;
};

export type ValueOption = {
  value: string;
  keywords: string[];
  custom?: boolean;
  render: RenderFunction;
};
