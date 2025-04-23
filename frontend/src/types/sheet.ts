import type { VNode } from "vue";
import type { Worksheet_Visibility } from "@/types/proto/api/v1alpha/worksheet_service";

export type AccessOption = {
  label: string;
  description: string;
  value: Worksheet_Visibility;
  icon: VNode;
};
