import slug from "slug";
import { Webhook } from "@/types/proto/v1/project_service";

export const extractProjectWebhookID = (name: string) => {
  const pattern = /(?:^|\/)webhooks\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export function projectWebhookV1Slug(webhook: Webhook): string {
  const id = extractProjectWebhookID(webhook.name);
  return [slug(webhook.title), id].join("-");
}
