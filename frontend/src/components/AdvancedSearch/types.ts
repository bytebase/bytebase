import type { RenderFunction } from "vue";
import type { SearchScopeId } from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  description: string;
  options: ValueOption[];
};

export type ValueOption = {
  value: string;
  keywords: string[];
  custom?: boolean;
  render: RenderFunction;
};
