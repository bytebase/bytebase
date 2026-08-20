import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { WebhookType } from "@/types/proto-es/v1/common_pb";
import { WebhookSchema } from "@/types/proto-es/v1/project_service_pb";
import { findAddedWebhook } from "./webhookCreation";

const webhook = (name: string, title: string, type = WebhookType.SLACK) =>
  create(WebhookSchema, { name, title, type });

const submitted = { title: "release channel", type: WebhookType.SLACK };

describe("findAddedWebhook", () => {
  test("finds the webhook a create call added", () => {
    const added = webhook("projects/p/webhooks/2", "release channel");
    expect(
      findAddedWebhook(
        [webhook("projects/p/webhooks/1", "alerts")],
        [webhook("projects/p/webhooks/1", "alerts"), added],
        submitted
      )
    ).toEqual(added);
  });

  test("finds it beside an existing one that shares its title", () => {
    const added = webhook("projects/p/webhooks/2", "release channel");
    expect(
      findAddedWebhook(
        [webhook("projects/p/webhooks/1", "release channel")],
        [webhook("projects/p/webhooks/1", "release channel"), added],
        submitted
      )
    ).toEqual(added);
  });

  test("finds the first webhook in a project that had none", () => {
    const added = webhook("projects/p/webhooks/1", "release channel");
    expect(findAddedWebhook([], [added], submitted)).toEqual(added);
  });

  test("ignores a webhook somebody else created while this form was open", () => {
    // Both names are new since the form loaded, so the name alone cannot tell
    // them apart and picking the first would navigate to the other webhook.
    const theirs = webhook("projects/p/webhooks/2", "somebody else's");
    const ours = webhook("projects/p/webhooks/3", "release channel");
    expect(findAddedWebhook([], [theirs, ours], submitted)).toEqual(ours);
  });

  test("ignores a new webhook of another type with the same title", () => {
    const theirs = webhook(
      "projects/p/webhooks/2",
      "release channel",
      WebhookType.DISCORD
    );
    const ours = webhook("projects/p/webhooks/3", "release channel");
    expect(findAddedWebhook([], [theirs, ours], submitted)).toEqual(ours);
  });

  test("returns undefined when nothing was added", () => {
    const existing = [webhook("projects/p/webhooks/1", "alerts")];
    expect(findAddedWebhook(existing, existing, submitted)).toBeUndefined();
  });

  test("refuses to guess when two clients created the same title and type", () => {
    // The response ordering is unspecified and both names are new, so either
    // pick is a coin flip. The caller sends the user to the list rather than
    // opening somebody else's webhook for editing.
    const ours = webhook("projects/p/webhooks/2", "release channel");
    const theirs = webhook("projects/p/webhooks/3", "release channel");
    expect(findAddedWebhook([], [ours, theirs], submitted)).toBeUndefined();
    expect(findAddedWebhook([], [theirs, ours], submitted)).toBeUndefined();
  });

  test("still answers when the concurrent webhook differs in title or type", () => {
    // Ambiguity is only ambiguity when the other webhook is indistinguishable.
    const ours = webhook("projects/p/webhooks/2", "release channel");
    const otherTitle = webhook("projects/p/webhooks/3", "somebody else's");
    const otherType = webhook(
      "projects/p/webhooks/4",
      "release channel",
      WebhookType.DISCORD
    );
    expect(
      findAddedWebhook([], [otherTitle, otherType, ours], submitted)
    ).toEqual(ours);
  });
});
