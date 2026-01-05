import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { projectServiceClientConnect } from "@/connect";
import type { IdType } from "@/types";
import {
  AddWebhookRequestSchema,
  type Project,
  RemoveWebhookRequestSchema,
  TestWebhookRequestSchema,
  UpdateWebhookRequestSchema,
  type Webhook,
} from "@/types/proto-es/v1/project_service_pb";
import { extractProjectWebhookID } from "@/utils";

export const useProjectWebhookV1Store = defineStore("projectWebhook_v1", () => {
  const getProjectWebhookFromProjectById = (
    project: Project,
    webhookId: IdType
  ) => {
    if (typeof webhookId === "string") {
      webhookId = parseInt(webhookId, 10);
    }
    return project.webhooks.find((webhook) => {
      return parseInt(extractProjectWebhookID(webhook.name), 10) === webhookId;
    });
  };
  const createProjectWebhook = async (project: string, webhook: Webhook) => {
    const request = create(AddWebhookRequestSchema, {
      project,
      webhook,
    });
    const response = await projectServiceClientConnect.addWebhook(request);
    return response;
  };
  const updateProjectWebhook = async (
    webhook: Webhook,
    updateMask: string[]
  ) => {
    const request = create(UpdateWebhookRequestSchema, {
      webhook,
      updateMask: { paths: updateMask },
    });
    const response = await projectServiceClientConnect.updateWebhook(request);
    return response;
  };
  const deleteProjectWebhook = async (webhook: Webhook) => {
    const request = create(RemoveWebhookRequestSchema, {
      webhook,
    });
    const response = await projectServiceClientConnect.removeWebhook(request);
    return response;
  };
  const testProjectWebhook = async (project: Project, webhook: Webhook) => {
    const request = create(TestWebhookRequestSchema, {
      project: project.name,
      webhook,
    });
    const response = await projectServiceClientConnect.testWebhook(request);
    return {
      error: response.error,
    };
  };

  return {
    getProjectWebhookFromProjectById,
    createProjectWebhook,
    updateProjectWebhook,
    deleteProjectWebhook,
    testProjectWebhook,
  };
});
