import { describe, expect, test } from "vitest";
import {
  canAddResourceMention,
  deleteResourceMention,
  getDraftResourceMentions,
  getMentionTriggerRange,
  hasAdditionalResourceMention,
  hasUnboundResourceMention,
  insertResourceMention,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
  removeMentionTrigger,
  updateDraftResourceMentions,
  updateResourceMentionRange,
} from "./resourceMention";

const projectReference = {
  kind: "project" as const,
  name: "projects/orders",
  project: "projects/orders",
  label: "Orders",
};
const databaseReference = {
  kind: "database" as const,
  name: "instances/main/databases/orders",
  project: "projects/orders",
  label: "orders_db",
};
const issueReference = {
  kind: "issue" as const,
  name: "projects/orders/issues/123",
  project: "projects/orders",
  label: "Update orders table",
};

describe("getMentionTriggerRange", () => {
  test("recognizes a mention at the caret after a word boundary", () => {
    expect(getMentionTriggerRange("@", 1)).toEqual({ start: 0, end: 1 });
    expect(getMentionTriggerRange("Please check @ord", 17)).toEqual({
      start: 13,
      end: 17,
    });
    expect(getMentionTriggerRange("See (@orders)", 12)).toEqual({
      start: 5,
      end: 12,
    });
  });

  test("does not interpret email addresses or earlier text as mentions", () => {
    expect(getMentionTriggerRange("ed@bytebase.com", 15)).toBeNull();
    expect(getMentionTriggerRange("hello @orders now", 17)).toBeNull();
    expect(getMentionTriggerRange("@one@two", 8)).toBeNull();
    expect(getMentionTriggerRange("@orders", 20)).toBeNull();
  });
});

describe("removeMentionTrigger", () => {
  test("removes only the unfinished trigger and returns its caret", () => {
    expect(removeMentionTrigger("Ask @", { start: 4, end: 5 })).toEqual({
      text: "Ask ",
      caret: 4,
    });
    expect(
      removeMentionTrigger("Ask @Billing and @ord", { start: 17, end: 21 })
    ).toEqual({ text: "Ask @Billing and ", caret: 17 });
  });

  test("does not delete stale ranges or email text", () => {
    expect(
      removeMentionTrigger("Ask Billing", { start: 4, end: 8 })
    ).toBeNull();
    expect(
      removeMentionTrigger("ed@example.com", { start: 2, end: 10 })
    ).toBeNull();
    expect(removeMentionTrigger("Ask @", null)).toBeNull();
  });
});

describe("inline resource mentions", () => {
  test.each([
    ["Backspace", 12],
    ["Backspace", 8],
    ["Delete", 4],
    ["Delete", 8],
  ] as const)(
    "deletes the selected resource atomically with %s at %i",
    (key, caret) => {
      expect(
        deleteResourceMention(
          "Ask @Billing about policy",
          { start: 4, end: 12 },
          "Billing",
          caret,
          caret,
          key
        )
      ).toEqual({ text: "Ask about policy", caret: 4 });
    }
  );

  test("backspace after the picker-inserted separator removes the resource and separator", () => {
    expect(
      deleteResourceMention(
        "Ask @Billing ",
        { start: 4, end: 12 },
        "Billing",
        13,
        13,
        "Backspace"
      )
    ).toEqual({ text: "Ask ", caret: 4 });
  });

  test("expands a selection overlapping a resource without dropping other text", () => {
    expect(
      deleteResourceMention(
        "Ask @Billing about policy",
        { start: 4, end: 12 },
        "Billing",
        2,
        7,
        "Delete"
      )
    ).toEqual({ text: "Asabout policy", caret: 2 });
  });

  test("leaves text and unbound or invalid mentions to native editing", () => {
    const text = "Ask @Billing about policy";
    const mention = { start: 4, end: 12 };
    expect(
      deleteResourceMention(text, mention, "Billing", 13, 13, "Delete")
    ).toBeNull();
    expect(
      deleteResourceMention(text, mention, "Billing", 3, 3, "Backspace")
    ).toBeNull();
    expect(
      deleteResourceMention(text, mention, "Billing", 4, 4, "Backspace")
    ).toBeNull();
    expect(
      deleteResourceMention(text, mention, "Billing", 12, 12, "Delete")
    ).toBeNull();
    expect(
      deleteResourceMention(text, null, "Billing", 12, 12, "Backspace")
    ).toBeNull();
    expect(
      deleteResourceMention(text, mention, "Orders", 12, 12, "Backspace")
    ).toBeNull();
    expect(
      deleteResourceMention(text, mention, "Billing", -1, -1, "Backspace")
    ).toBeNull();
  });

  test("rejects a second unselected mention without mistaking an email for one", () => {
    const selected = { start: 4, end: 12 };
    expect(
      hasAdditionalResourceMention("Ask @Billing and @Orders", selected)
    ).toBe(true);
    expect(hasAdditionalResourceMention("Ask @Billing and @", selected)).toBe(
      true
    );
    expect(
      hasAdditionalResourceMention("Ask @Billing and ed@example.com", selected)
    ).toBe(false);
    expect(
      hasAdditionalResourceMention("Ask @Billing about this", selected)
    ).toBe(false);
    expect(
      hasAdditionalResourceMention("Ask @QA @Prod about this", {
        start: 4,
        end: 13,
      })
    ).toBe(false);
  });

  test("replaces the typed trigger in place and leaves a separator for more text", () => {
    expect(
      insertResourceMention(
        "Check @bill against this issue",
        { start: 6, end: 11 },
        "Billing"
      )
    ).toEqual({
      text: "Check @Billing against this issue",
      mention: { start: 6, end: 14 },
      caret: 15,
    });
    expect(insertResourceMention("@", { start: 0, end: 1 }, "Billing")).toEqual(
      {
        text: "@Billing ",
        mention: { start: 0, end: 8 },
        caret: 9,
      }
    );
  });

  test("shifts a selected mention after edits before it", () => {
    expect(
      updateResourceMentionRange(
        "Ask @Billing about this",
        "Please ask @Billing about this",
        { start: 4, end: 12 },
        "Billing"
      )
    ).toEqual({ start: 11, end: 19 });
  });

  test("drops reference identity when the selected token is changed or removed", () => {
    expect(
      updateResourceMentionRange(
        "Ask @Billing about this",
        "Ask @Biling about this",
        { start: 4, end: 12 },
        "Billing"
      )
    ).toBeNull();
    expect(
      updateResourceMentionRange(
        "Ask @Billing about this",
        "Ask about this",
        { start: 4, end: 12 },
        "Billing"
      )
    ).toBeNull();
    expect(
      updateResourceMentionRange(
        "Ask @Billing and @Billing",
        "Ask and @Billing",
        { start: 4, end: 12 },
        "Billing"
      )
    ).toBeNull();
  });
});

describe("normalizeAgentResourceReference", () => {
  test("accepts a canonical Project reference only when its owner matches", () => {
    expect(
      normalizeAgentResourceReference({
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      })
    ).toEqual({
      kind: "project",
      name: "projects/billing",
      project: "projects/billing",
      label: "Billing",
    });
    expect(
      normalizeAgentResourceReference({
        kind: "project",
        name: "projects/billing",
        project: "projects/orders",
        label: "Billing",
      })
    ).toBeUndefined();
  });

  test("accepts an Issue only under its owning Project", () => {
    expect(
      normalizeAgentResourceReference({
        kind: "issue",
        name: "projects/orders/issues/123",
        project: "projects/orders",
        label: "Fix order table",
      })
    ).toEqual({
      kind: "issue",
      name: "projects/orders/issues/123",
      project: "projects/orders",
      label: "Fix order table",
    });
    expect(
      normalizeAgentResourceReference({
        kind: "issue",
        name: "projects/orders/issues/123",
        project: "projects/billing",
        label: "Wrong owner",
      })
    ).toBeUndefined();
  });

  test("accepts workspace- and project-instance Database names", () => {
    expect(
      normalizeAgentResourceReference({
        kind: "database",
        name: "instances/main/databases/orders",
        project: "projects/orders",
        label: "orders",
      })
    ).toMatchObject({ name: "instances/main/databases/orders" });
    expect(
      normalizeAgentResourceReference({
        kind: "database",
        name: "projects/orders/instances/main/databases/orders",
        project: "projects/orders",
        label: "orders",
      })
    ).toMatchObject({
      name: "projects/orders/instances/main/databases/orders",
    });
    expect(
      normalizeAgentResourceReference({
        kind: "database",
        name: "projects/billing/instances/main/databases/orders",
        project: "projects/orders",
        label: "Wrong owner",
      })
    ).toBeUndefined();
  });

  test("rejects malformed or incomplete cached references", () => {
    for (const raw of [
      null,
      {},
      { kind: "project", name: "projects/orders", project: "projects/orders" },
      {
        kind: "project",
        name: "projects/orders/issues/1",
        project: "projects/orders/issues/1",
        label: "Not a project",
      },
      {
        kind: "issue",
        name: "projects/orders/issues/",
        project: "projects/orders",
        label: "Issue",
      },
      {
        kind: "issue",
        name: "projects/orders/issues/1",
        project: "projects/orders",
        label: "",
      },
      {
        kind: "database",
        name: "instances/main/databases/orders",
        project: "projects/",
        label: "Orders",
      },
    ]) {
      expect(normalizeAgentResourceReference(raw)).toBeUndefined();
    }
  });
});

describe("paired resource mentions", () => {
  test("accepts one Project and its Database regardless of selection order", () => {
    expect(
      normalizeAgentResourceReferences([databaseReference, projectReference])
    ).toEqual([projectReference, databaseReference]);
    expect(
      normalizeAgentResourceReferences(undefined, databaseReference)
    ).toEqual([databaseReference]);
    expect(
      canAddResourceMention(
        [{ resource: projectReference, range: { start: 0, end: 7 } }],
        databaseReference
      )
    ).toBe(true);
  });

  test("accepts an Issue alone or with its Project in either selection order", () => {
    expect(normalizeAgentResourceReferences([issueReference])).toEqual([
      issueReference,
    ]);
    expect(
      normalizeAgentResourceReferences([issueReference, projectReference])
    ).toEqual([projectReference, issueReference]);
    expect(
      normalizeAgentResourceReferences([projectReference, issueReference])
    ).toEqual([projectReference, issueReference]);
    expect(
      canAddResourceMention(
        [{ resource: issueReference, range: { start: 0, end: 20 } }],
        projectReference
      )
    ).toBe(true);
  });

  test("rejects an Issue with a Database, another Project, or a duplicate", () => {
    const otherProject = {
      ...projectReference,
      name: "projects/billing",
      project: "projects/billing",
    };
    expect(
      normalizeAgentResourceReferences([databaseReference, issueReference])
    ).toEqual([]);
    expect(
      normalizeAgentResourceReferences([otherProject, issueReference])
    ).toEqual([]);
    expect(
      normalizeAgentResourceReferences([issueReference, issueReference])
    ).toEqual([]);
    expect(
      canAddResourceMention(
        [{ resource: databaseReference, range: { start: 0, end: 10 } }],
        issueReference
      )
    ).toBe(false);
    expect(
      normalizeAgentResourceReferences([
        projectReference,
        databaseReference,
        issueReference,
      ])
    ).toEqual([]);
  });

  test("rejects mismatched, duplicate, malformed, or excessive references", () => {
    const otherDatabase = { ...databaseReference, project: "projects/other" };
    expect(
      normalizeAgentResourceReferences([projectReference, otherDatabase])
    ).toEqual([]);
    expect(
      normalizeAgentResourceReferences([projectReference, projectReference])
    ).toEqual([]);
    expect(
      normalizeAgentResourceReferences([
        projectReference,
        databaseReference,
        otherDatabase,
      ])
    ).toEqual([]);
    expect(
      normalizeAgentResourceReferences(
        [projectReference, {}],
        databaseReference
      )
    ).toEqual([]);
  });

  test("tracks independent ranges through edits and atomically drops an edited binding", () => {
    const initial = "@Orders inspect @orders_db";
    const mentions = [
      { resource: projectReference, range: { start: 0, end: 7 } },
      { resource: databaseReference, range: { start: 16, end: 26 } },
    ];
    expect(getDraftResourceMentions({ text: initial, mentions })).toEqual(
      mentions
    );
    expect(hasUnboundResourceMention(initial, mentions)).toBe(false);

    const prefixed = `Please ${initial}`;
    expect(updateDraftResourceMentions(initial, prefixed, mentions)).toEqual([
      { resource: projectReference, range: { start: 7, end: 14 } },
      { resource: databaseReference, range: { start: 23, end: 33 } },
    ]);

    const edited = "@Orders inspect @orders_d";
    const remaining = updateDraftResourceMentions(initial, edited, mentions);
    expect(remaining).toEqual([mentions[0]]);
    expect(hasUnboundResourceMention(edited, remaining)).toBe(true);

    expect(
      updateDraftResourceMentions(initial, "inspect @orders_db", mentions)
    ).toEqual([{ resource: databaseReference, range: { start: 8, end: 18 } }]);
  });

  test("recognizes a stray second mention but not an email address", () => {
    const text = "@Orders inspect @orders_db and @another ed@example.com";
    const mentions = [
      { resource: projectReference, range: { start: 0, end: 7 } },
      { resource: databaseReference, range: { start: 16, end: 26 } },
    ];
    expect(hasUnboundResourceMention(text, mentions)).toBe(true);
    expect(
      hasUnboundResourceMention(
        "@Orders inspect @orders_db ed@example.com",
        mentions
      )
    ).toBe(false);
  });
});
