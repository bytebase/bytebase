import { defineStore } from "pinia";
import axios from "axios";
import {
  ProjectId,
  ProjectWebhook,
  ProjectWebhookCreate,
  ProjectWebhookId,
  ProjectWebhookPatch,
  ProjectWebhookState,
  ProjectWebhookTestResult,
  ResourceObject,
  unknown,
} from "../../types";
import { getPrincipalFromIncludedList } from "@/store/modules/principal";

function convert(
  projectWebhook: ResourceObject,
  includedList: ResourceObject[]
): ProjectWebhook {
  return {
    ...(projectWebhook.attributes as Omit<
      ProjectWebhook,
      "id" | "creator" | "updater"
    >),
    id: parseInt(projectWebhook.id),
    creator: getPrincipalFromIncludedList(
      projectWebhook.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      projectWebhook.relationships!.updater.data,
      includedList
    ),
  };
}

function convertTestResult(
  testResult: ResourceObject
): ProjectWebhookTestResult {
  return {
    ...(testResult.attributes as ProjectWebhookTestResult),
  };
}

export const useProjectWebhookStore = defineStore("projectWebhook", {
  state: (): ProjectWebhookState => ({
    projectWebhookList: new Map(),
  }),

  actions: {
    projectWebhookListByProjectId(projectId: ProjectId): ProjectWebhook[] {
      return this.projectWebhookList.get(projectId) || [];
    },
    projectWebhookById(
      projectId: ProjectId,
      projectWebhookId: ProjectWebhookId
    ): ProjectWebhook {
      const list = this.projectWebhookList.get(projectId);
      if (list) {
        for (const hook of list) {
          if (hook.id == projectWebhookId) {
            return hook;
          }
        }
      }
      return unknown("PROJECT_HOOK") as ProjectWebhook;
    },
    async createProjectWebhook({
      projectId,
      projectWebhookCreate,
    }: {
      projectId: ProjectId;
      projectWebhookCreate: ProjectWebhookCreate;
    }): Promise<ProjectWebhook> {
      const data = (
        await axios.post(`/api/project/${projectId}/webhook`, {
          data: {
            type: "projectWebhookCreate",
            attributes: projectWebhookCreate,
          },
        })
      ).data;
      const createdProjectWebhook = convert(data.data, data.included);

      this.upsertProjectWebhookByProjectId({
        projectId,
        projectWebhook: createdProjectWebhook,
      });

      return createdProjectWebhook;
    },

    async fetchProjectWebhookListByProjectId(
      projectId: ProjectId
    ): Promise<ProjectWebhook[]> {
      const data = (await axios.get(`/api/project/${projectId}/webhook`)).data;
      const projectWebhookList = data.data.map(
        (projectWebhook: ResourceObject) => {
          return convert(projectWebhook, data.included);
        }
      );

      this.setProjectWebhookListByProjectId({
        projectId,
        projectWebhookList,
      });

      return projectWebhookList;
    },

    async fetchProjectWebhookById({
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }): Promise<ProjectWebhook> {
      const data = (
        await axios.get(`/api/project/${projectId}/webhook/${projectWebhookId}`)
      ).data;
      const projectWebhook = convert(data.data, data.included);

      this.upsertProjectWebhookByProjectId({
        projectId,
        projectWebhook,
      });

      return projectWebhook;
    },

    async updateProjectWebhookById({
      projectId,
      projectWebhookId,
      projectWebhookPatch,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
      projectWebhookPatch: ProjectWebhookPatch;
    }) {
      const data = (
        await axios.patch(
          `/api/project/${projectId}/webhook/${projectWebhookId}`,
          {
            data: {
              type: "projectWebhookPatch",
              attributes: projectWebhookPatch,
            },
          }
        )
      ).data;
      const updatedProjectWebhook = convert(data.data, data.included);

      this.upsertProjectWebhookByProjectId({
        projectId,
        projectWebhook: updatedProjectWebhook,
      });

      return updatedProjectWebhook;
    },

    async deleteProjectWebhookById({
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }) {
      await axios.delete(
        `/api/project/${projectId}/webhook/${projectWebhookId}`
      );

      const list = this.projectWebhookList.get(projectId);
      if (list) {
        const i = list.findIndex((item) => item.id == projectWebhookId);
        if (i >= 0) {
          list.splice(i, 1);
        }
      }
    },

    async testProjectWebhookById({
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }) {
      const data = (
        await axios.get(
          `/api/project/${projectId}/webhook/${projectWebhookId}/test`
        )
      ).data;

      return convertTestResult(data.data);
    },
    setProjectWebhookListByProjectId({
      projectId,
      projectWebhookList,
    }: {
      projectId: ProjectId;
      projectWebhookList: ProjectWebhook[];
    }) {
      this.projectWebhookList.set(projectId, projectWebhookList);
    },

    upsertProjectWebhookByProjectId({
      projectId,
      projectWebhook,
    }: {
      projectId: ProjectId;
      projectWebhook: ProjectWebhook;
    }) {
      const list = this.projectWebhookList.get(projectId);
      if (list) {
        const i = list.findIndex((item) => item.id == projectWebhook.id);
        if (i >= 0) {
          list[i] = projectWebhook;
        } else {
          list.push(projectWebhook);
        }
      } else {
        this.projectWebhookList.set(projectId, [projectWebhook]);
      }
    },
  },
});
