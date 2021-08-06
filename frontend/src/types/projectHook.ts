import { ActionType } from "./activity";
import { MemberId, ProjectId } from "./id";
import { Principal } from "./principal";

// Project Member
export type ProjectHook = {
  id: MemberId;

  // Related fields
  projectId: ProjectId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  url: string;
  activityList: ActionType[];
};

export type ProjectHookCreate = {
  // Domain specific fields
  name: string;
  url: string;
  activityList: ActionType[];
};

export type ProjectHookPatch = {
  // Domain specific fields
  name?: string;
  url?: string;
  activityList?: ActionType[];
};
