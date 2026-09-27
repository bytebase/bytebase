import { create as createProto } from "@bufbuild/protobuf";
import { UNKNOWN_ID } from "@/types/const";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Project,
  ProjectSchema,
} from "@/types/proto-es/v1/project_service_pb";
import type { ProjectFilter } from "./types";

const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;

export function createUnknownProject(): Project {
  return createProto(ProjectSchema, {
    name: UNKNOWN_PROJECT_NAME,
    state: State.ACTIVE,
    enforceIssueTitle: true,
    enforceSqlReview: true,
    requireIssueApproval: true,
    requirePlanCheckNoError: true,
    allowRequestRole: true,
  });
}

type ProjectAccess = {
  getProjectByName: (name: string) => Project;
  batchGetOrFetchProjects: (names: string[]) => Promise<Project[]>;
  fetchProjectList: (params: {
    pageSize?: number;
    pageToken?: string;
    silent?: boolean;
    filter?: ProjectFilter;
    orderBy?: string;
    cache?: boolean;
  }) => Promise<{ projects: Project[]; nextPageToken?: string }>;
};

// Indirection layer so low-level shared utilities (which sit inside the app
// store's own load graph via the `@/utils` barrel) can read project state
// without statically importing the store index — that would form an
// initialization cycle. The slice registers the real implementation via
// `setProjectAccess` at store-creation time.
let projectAccess: ProjectAccess = {
  getProjectByName: () => createUnknownProject(),
  batchGetOrFetchProjects: async () => [],
  fetchProjectList: async () => ({ projects: [], nextPageToken: "" }),
};

export const setProjectAccess = (access: ProjectAccess) => {
  projectAccess = access;
};

export const getProjectByName = (name: string) =>
  projectAccess.getProjectByName(name);

export const batchGetOrFetchProjects = (names: string[]) =>
  projectAccess.batchGetOrFetchProjects(names);

export const fetchProjectList = (
  params: Parameters<ProjectAccess["fetchProjectList"]>[0]
) => projectAccess.fetchProjectList(params);
