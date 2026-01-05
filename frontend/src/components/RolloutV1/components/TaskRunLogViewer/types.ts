import type { Component } from "vue";
import type { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";

export type SectionStatus = "success" | "error" | "running" | "pending";

export interface DisplayItem {
  key: string;
  time: string;
  relativeTime: string;
  levelIndicator: string;
  levelClass: string;
  detail: string;
  detailClass: string;
  affectedRows?: number;
}

export interface Section {
  id: string;
  type: TaskRunLogEntry_Type;
  label: string;
  status: SectionStatus;
  statusIcon: Component;
  statusClass: string;
  duration: string;
  entryCount: number;
  items: DisplayItem[];
}

export interface EntryGroup {
  type: TaskRunLogEntry_Type;
  entries: import("@/types/proto-es/v1/rollout_service_pb").TaskRunLogEntry[];
}

export interface ReleaseFileGroup {
  version: string;
  filePath: string;
  sections: Section[];
}

export interface DeployGroup {
  deployId: string;
  releaseFileGroups: ReleaseFileGroup[];
  // Sections not associated with any release file (e.g., logs before file execution)
  sections: Section[];
}
