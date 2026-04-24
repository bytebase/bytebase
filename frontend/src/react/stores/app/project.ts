import { create as createProto } from "@bufbuild/protobuf";
import { projectServiceClientConnect } from "@/connect";
import { isValidProjectName } from "@/react/lib/resourceName";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchGetProjectsRequestSchema,
  CreateProjectRequestSchema,
  GetProjectRequestSchema,
  ProjectSchema,
  SearchProjectsRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import type { AppSliceCreator, ProjectSlice } from "./types";
import { buildProjectFilter, defaultProjectName } from "./utils";

export const createProjectSlice: AppSliceCreator<ProjectSlice> = (
  set,
  get
) => ({
  projectsByName: {},
  projectRequests: {},

  fetchProject: async (name) => {
    if (!isValidProjectName(name) || name === defaultProjectName(get)) {
      return undefined;
    }
    const existing = get().projectsByName[name];
    if (existing) return existing;
    const pending = get().projectRequests[name];
    if (pending) return pending;

    const request = projectServiceClientConnect
      .getProject(createProto(GetProjectRequestSchema, { name }))
      .then((project) => {
        set((state) => {
          const { [name]: _, ...projectRequests } = state.projectRequests;
          return {
            projectsByName: {
              ...state.projectsByName,
              [project.name]: project,
            },
            projectRequests,
          };
        });
        return project;
      })
      .catch(() => {
        set((state) => {
          const { [name]: _, ...projectRequests } = state.projectRequests;
          return { projectRequests };
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
    set((state) => ({
      projectsByName: {
        ...state.projectsByName,
        [created.name]: created,
      },
    }));
    return created;
  },
});
