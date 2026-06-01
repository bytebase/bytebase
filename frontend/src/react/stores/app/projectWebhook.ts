import { create as createProto } from "@bufbuild/protobuf";
import { projectServiceClientConnect } from "@/connect";
import {
  AddWebhookRequestSchema,
  RemoveWebhookRequestSchema,
  TestWebhookRequestSchema,
  UpdateWebhookRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { extractProjectWebhookID } from "@/utils/v1/projectWebhook";
import type { AppSliceCreator, ProjectWebhookSlice } from "./types";

export const createProjectWebhookSlice: AppSliceCreator<ProjectWebhookSlice> = (
  set
) => ({
  getProjectWebhookFromProjectById: (project, webhookId) => {
    return project.webhooks.find(
      (webhook) => extractProjectWebhookID(webhook.name) === webhookId
    );
  },

  createProjectWebhook: async (project, webhook) => {
    const response = await projectServiceClientConnect.addWebhook(
      createProto(AddWebhookRequestSchema, {
        project,
        webhook,
      })
    );
    set((state) => ({
      projectsByName: { ...state.projectsByName, [response.name]: response },
    }));
    return response;
  },

  updateProjectWebhook: async (webhook, updateMask) => {
    const response = await projectServiceClientConnect.updateWebhook(
      createProto(UpdateWebhookRequestSchema, {
        webhook,
        updateMask: { paths: updateMask },
      })
    );
    set((state) => ({
      projectsByName: { ...state.projectsByName, [response.name]: response },
    }));
    return response;
  },

  deleteProjectWebhook: async (webhook) => {
    const response = await projectServiceClientConnect.removeWebhook(
      createProto(RemoveWebhookRequestSchema, { webhook })
    );
    set((state) => ({
      projectsByName: { ...state.projectsByName, [response.name]: response },
    }));
    return response;
  },

  testProjectWebhook: async (project, webhook) => {
    const response = await projectServiceClientConnect.testWebhook(
      createProto(TestWebhookRequestSchema, {
        project: project.name,
        webhook,
      })
    );
    return { error: response.error };
  },
});
