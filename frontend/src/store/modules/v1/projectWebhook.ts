import { defineStore } from "pinia";
import { projectServiceClient } from "@/grpcweb";
import { IdType } from "@/types";
import { Project, Webhook } from "@/types/proto/v1/project_service";
import { extractProjectWebhookID } from "@/utils";
import { useProjectV1Store } from "./project";

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
  const createProjectWebhook = async (project: Project, webhook: Webhook) => {
    const updatedProject = await projectServiceClient.addWebhook({
      project: project.name,
      webhook,
    });
    await useProjectV1Store().upsertProjectMap([updatedProject]);
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
    await useProjectV1Store().upsertProjectMap([updatedProject]);
    return updatedProject;
  };
  const deleteProjectWebhook = async (webhook: Webhook) => {
    const updatedProject = await projectServiceClient.removeWebhook({
      webhook,
    });
    await useProjectV1Store().upsertProjectMap([updatedProject]);
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
