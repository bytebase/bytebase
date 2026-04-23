import { describe, expect, test, vi } from "vitest";
import { WebhookType } from "../proto-es/v1/common_pb";

vi.mock("@/plugins/i18n", () => ({
  t: (key: string) => (key === "common.google-chat" ? "Google Chat" : key),
}));

import { projectWebhookV1TypeItemList } from "./projectWebhook";

describe("projectWebhookV1TypeItemList", () => {
  test("includes Google Chat as a channel-only webhook destination", () => {
    const item = projectWebhookV1TypeItemList().find(
      (item) => item.type === WebhookType.GOOGLE_CHAT
    );

    expect(item).toMatchObject({
      name: "Google Chat",
      urlPrefix: "https://chat.googleapis.com",
      urlPlaceholder:
        "https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...",
      docUrl:
        "https://developers.google.com/workspace/chat/quickstart/webhooks",
      supportDirectMessage: false,
    });
  });
});
