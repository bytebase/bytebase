import { defineStore } from "pinia";
import { create } from "@bufbuild/protobuf";
import { projectServiceClientConnect } from "@/grpcweb";
import type { IdType } from "@/types";
import type { Project, Webhook } from "@/types/proto/v1/project_service";
import {
  AddWebhookRequestSchema,
  UpdateWebhookRequestSchema,
  RemoveWebhookRequestSchema,
  TestWebhookRequestSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { convertNewProjectToOld, convertOldWebhookToNew } from "@/utils/v1/project-conversions";
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
      webhook: convertOldWebhookToNew(webhook),
    });
    const response = await projectServiceClientConnect.addWebhook(request);
    return convertNewProjectToOld(response);
  };
  const updateProjectWebhook = async (
    webhook: Webhook,
    updateMask: string[]
  ) => {
    const request = create(UpdateWebhookRequestSchema, {
      webhook: convertOldWebhookToNew(webhook),
      updateMask: { paths: updateMask },
    });
    const response = await projectServiceClientConnect.updateWebhook(request);
    return convertNewProjectToOld(response);
  };
  const deleteProjectWebhook = async (webhook: Webhook) => {
    const request = create(RemoveWebhookRequestSchema, {
      webhook: convertOldWebhookToNew(webhook),
    });
    const response = await projectServiceClientConnect.removeWebhook(request);
    return convertNewProjectToOld(response);
  };
  const testProjectWebhook = async (project: Project, webhook: Webhook) => {
    const request = create(TestWebhookRequestSchema, {
      project: project.name,
      webhook: convertOldWebhookToNew(webhook),
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
