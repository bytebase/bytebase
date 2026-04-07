import type { LucideIcon } from "lucide-react";
import type {
  TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";

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
  duration?: string;
}

export interface Section {
  id: string;
  type: TaskRunLogEntry_Type;
  label: string;
  status: SectionStatus;
  statusIcon: LucideIcon;
  statusClass: string;
  duration: string;
  entryCount: number;
  items: DisplayItem[];
}

export interface EntryGroup {
  type: TaskRunLogEntry_Type;
  entries: TaskRunLogEntry[];
}

export interface ReleaseFileInfo {
  version: string;
  filePath: string;
}

export interface ReleaseFileEntriesGroup {
  file: ReleaseFileInfo | null;
  entries: TaskRunLogEntry[];
}

export interface ReleaseFileGroup {
  id: string;
  version: string;
  filePath: string;
  isOrphan?: boolean;
  sections: Section[];
}

export interface ReplicaGroup {
  replicaId: string;
  releaseFileGroups: ReleaseFileGroup[];
  sections: Section[];
}
