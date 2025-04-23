import { defineStore } from "pinia";
import { projectServiceClient } from "@/grpcweb";
import type { IdType } from "@/types";
import type { Project, Webhook } from "@/types/proto/api/v1alpha/project_service";
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
    const updatedProject = await projectServiceClient.addWebhook({
      project,
      webhook,
    });
    return updatedProject;
  };
  const updateProjectWebhook = async (
    webhook: Webhook,
    updateMask: string[]
  ) => {
    const updatedProject = await projectServiceClient.updateWebhook({
      webhook,
      updateMask,
    });
    return updatedProject;
  };
  const deleteProjectWebhook = async (webhook: Webhook) => {
    const updatedProject = await projectServiceClient.removeWebhook({
      webhook,
    });
    return updatedProject;
  };
  const testProjectWebhook = async (project: Project, webhook: Webhook) => {
    const response = await projectServiceClient.testWebhook({
      project: project.name,
      webhook,
    });
    return response;
  };

  return {
    getProjectWebhookFromProjectById,
    createProjectWebhook,
    updateProjectWebhook,
    deleteProjectWebhook,
    testProjectWebhook,
  };
});
