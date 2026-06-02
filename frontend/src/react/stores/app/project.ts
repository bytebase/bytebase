import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { projectServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidProjectName } from "@/react/lib/resourceName";
import { UNKNOWN_ID } from "@/types/const";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchDeleteProjectsRequestSchema,
  BatchGetProjectsRequestSchema,
  CreateProjectRequestSchema,
  DeleteProjectRequestSchema,
  GetProjectRequestSchema,
  ListProjectsRequestSchema,
  type Project,
  ProjectSchema,
  SearchProjectsRequestSchema,
  UndeleteProjectRequestSchema,
  UpdateProjectRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { setProjectAccess } from "./projectAccess";
import type { AppSliceCreator, ProjectFilter, ProjectSlice } from "./types";
import {
  buildProjectFilter,
  defaultProjectName,
  getLabelFilter,
  toError,
} from "./utils";

const getListProjectFilter = (params: ProjectFilter): string => {
  const list: string[] = [];
  const search = params.query?.trim().toLowerCase();
  if (search) {
    list.push(
      `(name.contains("${search}") || resource_id.contains("${search}"))`
    );
  }
  if (params.labels) {
    list.push(...getLabelFilter(params.labels));
  }
  if (params.excludeDefault) {
    list.push("exclude_default == true");
  }
  if (params.state === State.DELETED) {
    list.push(`state == "${State[params.state]}"`);
  }
  return list.join(" && ");
};

const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;

// The default project's feature gates. Inlined here (rather than reusing
// the `@/types/v1/project` helpers) so the app store's load graph stays
// free of the Pinia actuator store those helpers pull in.
const PROJECT_DEFAULTS = {
  state: State.ACTIVE,
  enforceIssueTitle: true,
  enforceSqlReview: true,
  requireIssueApproval: true,
  requirePlanCheckNoError: true,
  allowRequestRole: true,
} as const;

function createDefaultProject(name: string) {
  return createProto(ProjectSchema, {
    name,
    title: "Default project",
    ...PROJECT_DEFAULTS,
  });
}

function createUnknownProject() {
  return createProto(ProjectSchema, {
    name: UNKNOWN_PROJECT_NAME,
    ...PROJECT_DEFAULTS,
  });
}

export const createProjectSlice: AppSliceCreator<ProjectSlice> = (set, get) => {
  // Immutable bulk upsert into the by-name cache.
  const upsertProjects = (projects: Project[]): void => {
    set((state) => {
      const next = { ...state.projectsByName };
      for (const project of projects) {
        next[project.name] = project;
      }
      return { projectsByName: next };
    });
  };

  const slice: ProjectSlice = {
    projectsByName: {},
    projectRequests: {},
    projectErrorsByName: {},

    resetProjects: () => {
      set({ projectsByName: {}, projectRequests: {}, projectErrorsByName: {} });
    },

    // Mirrors the Pinia `useProjectV1Store().getProjectByName`: always
    // returns a non-null Project, synthesizing the default-project or
    // unknown-project placeholder when the name is not in the cache.
    getProjectByName: (name) => {
      if (name === UNKNOWN_PROJECT_NAME) return createUnknownProject();
      const defaultProject = defaultProjectName(get);
      if (name && name === defaultProject) {
        return get().projectsByName[name] ?? createDefaultProject(name);
      }
      return get().projectsByName[name] ?? createUnknownProject();
    },

    fetchProject: async (name, silent = false) => {
      const defaultProject = defaultProjectName(get);
      if (name && name === defaultProject) {
        const project = createDefaultProject(name);
        set((state) => ({
          projectsByName: {
            ...state.projectsByName,
            [name]: state.projectsByName[name] ?? project,
          },
          projectErrorsByName: {
            ...state.projectErrorsByName,
            [name]: undefined,
          },
        }));
        return project;
      }
      if (!isValidProjectName(name) || name === UNKNOWN_PROJECT_NAME) {
        return undefined;
      }
      const existing = get().projectsByName[name];
      if (existing) return existing;
      const pending = get().projectRequests[name];
      if (pending) return pending;

      const request = projectServiceClientConnect
        .getProject(createProto(GetProjectRequestSchema, { name }), {
          contextValues: createContextValues().set(silentContextKey, silent),
        })
        .then((project) => {
          set((state) => {
            const { [name]: _, ...projectRequests } = state.projectRequests;
            return {
              projectsByName: {
                ...state.projectsByName,
                [project.name]: project,
              },
              projectErrorsByName: {
                ...state.projectErrorsByName,
                [name]: undefined,
              },
              projectRequests,
            };
          });
          return project;
        })
        .catch((error) => {
          set((state) => {
            const { [name]: _, ...projectRequests } = state.projectRequests;
            return {
              projectErrorsByName: {
                ...state.projectErrorsByName,
                [name]: toError(error),
              },
              projectRequests,
            };
          });
          return undefined;
        });

      set((state) => ({
        projectRequests: { ...state.projectRequests, [name]: request },
      }));
      return request;
    },

    batchFetchProjects: async (names) => {
      const validNames = [...new Set(names)].filter(
        (name) => isValidProjectName(name) && name !== defaultProjectName(get)
      );
      if (validNames.length === 0) return [];

      try {
        const response = await projectServiceClientConnect.batchGetProjects(
          createProto(BatchGetProjectsRequestSchema, { names: validNames })
        );
        set((state) => ({
          projectsByName: {
            ...state.projectsByName,
            ...Object.fromEntries(
              response.projects.map((project) => [project.name, project])
            ),
          },
        }));
        return response.projects;
      } catch {
        const projects = await Promise.all(
          validNames.map((name) => get().fetchProject(name))
        );
        return projects.filter(
          (project): project is NonNullable<typeof project> => Boolean(project)
        );
      }
    },

    batchGetOrFetchProjects: async (names) => {
      const validNames = [...new Set(names)].filter(
        (name) => isValidProjectName(name) && name !== defaultProjectName(get)
      );
      const pending = validNames.filter(
        (name) => !isValidProjectName(get().getProjectByName(name).name)
      );
      await get().batchFetchProjects(pending);
      return validNames.map((name) => get().getProjectByName(name));
    },

    searchProjects: async ({ pageSize, pageToken, query }) => {
      const response = await projectServiceClientConnect.searchProjects(
        createProto(SearchProjectsRequestSchema, {
          pageSize,
          pageToken,
          filter: buildProjectFilter(query),
          orderBy: "title",
          showDeleted: false,
        })
      );
      set((state) => ({
        projectsByName: {
          ...state.projectsByName,
          ...Object.fromEntries(
            response.projects.map((project) => [project.name, project])
          ),
        },
      }));
      return {
        projects: response.projects.filter(
          (project) =>
            project.state === State.ACTIVE &&
            project.name !== defaultProjectName(get)
        ),
        nextPageToken: response.nextPageToken,
      };
    },

    createProject: async (title, resourceId) => {
      const project = createProto(ProjectSchema, {
        title,
        state: State.ACTIVE,
        enforceIssueTitle: true,
        enforceSqlReview: true,
        requireIssueApproval: true,
        requirePlanCheckNoError: true,
        allowRequestRole: true,
      });
      const created = await projectServiceClientConnect.createProject(
        createProto(CreateProjectRequestSchema, {
          project,
          projectId: resourceId,
        })
      );
      upsertProjects([created]);
      return created;
    },

    getOrFetchProjectByName: async (name, silent = true) => {
      const cached = get().getProjectByName(name);
      if (cached && cached.name !== UNKNOWN_PROJECT_NAME) {
        return cached;
      }
      return (await get().fetchProject(name, silent)) ?? createUnknownProject();
    },

    fetchProjectList: async (params) => {
      const filter = getListProjectFilter(params.filter ?? {});
      const showDeleted = params.filter?.state !== State.ACTIVE;
      const canList = hasWorkspacePermissionV2("bb.projects.list");
      let pageToken = params.pageToken;
      let result: { projects: Project[]; nextPageToken: string };
      // The API can return an empty page with a non-empty next token; keep
      // paging until we get rows or run out (mirrors the legacy Pinia store).
      while (true) {
        const request = {
          pageSize: params.pageSize,
          pageToken,
          filter,
          orderBy: params.orderBy,
          showDeleted,
        };
        const response = canList
          ? await projectServiceClientConnect.listProjects(
              createProto(ListProjectsRequestSchema, request),
              {
                contextValues: createContextValues().set(
                  silentContextKey,
                  params.silent ?? true
                ),
              }
            )
          : await projectServiceClientConnect.searchProjects(
              createProto(SearchProjectsRequestSchema, request),
              {
                contextValues: createContextValues().set(
                  silentContextKey,
                  params.silent ?? true
                ),
              }
            );
        result = {
          projects: response.projects,
          nextPageToken: response.nextPageToken,
        };
        if (result.nextPageToken !== "" && result.projects.length === 0) {
          pageToken = result.nextPageToken;
          continue;
        }
        break;
      }
      if (params.cache) {
        upsertProjects(result.projects);
      }
      return result;
    },

    updateProject: async (project, updateMask) => {
      const response = await projectServiceClientConnect.updateProject(
        createProto(UpdateProjectRequestSchema, {
          project,
          updateMask: { paths: updateMask },
        })
      );
      upsertProjects([response]);
      return response;
    },

    archiveProject: async (project) => {
      await projectServiceClientConnect.deleteProject(
        createProto(DeleteProjectRequestSchema, { name: project.name })
      );
      upsertProjects([{ ...project, state: State.DELETED }]);
    },

    restoreProject: async (project) => {
      const response = await projectServiceClientConnect.undeleteProject(
        createProto(UndeleteProjectRequestSchema, { name: project.name })
      );
      upsertProjects([response]);
    },

    deleteProject: async (project) => {
      await projectServiceClientConnect.deleteProject(
        createProto(DeleteProjectRequestSchema, { name: project, purge: true })
      );
      set((state) => {
        const { [project]: _removed, ...projectsByName } = state.projectsByName;
        return { projectsByName };
      });
    },

    batchDeleteProjects: async (projectNames) => {
      await projectServiceClientConnect.batchDeleteProjects(
        createProto(BatchDeleteProjectsRequestSchema, { names: projectNames })
      );
      const deleted = projectNames
        .map((name) => get().projectsByName[name])
        .filter((project): project is Project => Boolean(project))
        .map((project) => ({ ...project, state: State.DELETED }));
      upsertProjects(deleted);
    },

    batchPurgeProjects: async (projectNames) => {
      await projectServiceClientConnect.batchDeleteProjects(
        createProto(BatchDeleteProjectsRequestSchema, {
          names: projectNames,
          purge: true,
        })
      );
      set((state) => {
        const next = { ...state.projectsByName };
        for (const name of projectNames) {
          delete next[name];
        }
        return { projectsByName: next };
      });
    },

    updateProjectCache: (project) => {
      upsertProjects([project]);
    },
  };

  // Wire the cycle-free access layer used by shared `@/utils/*` helpers.
  setProjectAccess({
    getProjectByName: slice.getProjectByName,
    batchGetOrFetchProjects: slice.batchGetOrFetchProjects,
    fetchProjectList: slice.fetchProjectList,
  });

  return slice;
};
