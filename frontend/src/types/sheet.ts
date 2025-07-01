import type { VNode } from "vue";
import type { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";

export type AccessOption = {
  label: string;
  description: string;
  value: Worksheet_Visibility;
  icon: VNode;
};
