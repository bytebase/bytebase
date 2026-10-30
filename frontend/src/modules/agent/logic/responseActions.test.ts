import { describe, expect, test } from "vitest";
import {
  acceptAgentResponseActions,
  normalizeAgentResponseActions,
} from "./responseActions";
import type { AgentResponseAction } from "./types";

const open = {
  kind: "open",
  target: { kind: "create-project-database", project: "projects/sample" },
} satisfies AgentResponseAction;
const followUp = {
  kind: "continue",
  prompt: "What should I configure first?",
} satisfies AgentResponseAction;
const planDraft = {
  kind: "draft-plan",
  project: "projects/sample",
  database: "instances/prod/databases/app",
} as const;
const issueCommentDraft = {
  kind: "draft-issue-comment",
  issue: "projects/sample/issues/7",
  comment: "Please confirm the migration does not expose sensitive columns.",
} as const;
const ddlAnswer = "I suggest:\n\n```sql\nCREATE TABLE notes (id INT);\n```";

describe("response actions", () => {
  test("accepts one typed Open and one unsent Continue action", () => {
    expect(normalizeAgentResponseActions([open, followUp])).toEqual([
      open,
      followUp,
    ]);
    expect(
      normalizeAgentResponseActions([
        {
          ...open,
          prompt: "",
          target: { ...open.target, workspace: "", plan: "" },
        },
        { ...followUp, target: {} },
      ])
    ).toEqual([open, followUp]);
  });

  test("requires fresh evidence for the exact Open target", () => {
    expect(
      acceptAgentResponseActions(
        [open, followUp],
        [{ kind: "create-project-database", project: "projects/other" }]
      )
    ).toEqual([followUp]);
    expect(acceptAgentResponseActions([open, followUp], [open.target])).toEqual(
      [open, followUp]
    );
  });

  test("accepts exact Issue destinations only when observed in this turn", () => {
    const targets = [
      { kind: "open-issue", issue: "projects/sample/issues/7" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/4" },
      {
        kind: "view-failed-task",
        task: "projects/sample/plans/4/rollout/stages/prod/tasks/9",
      },
    ] as const;
    for (const target of targets) {
      const action = { kind: "open", target } as const;
      expect(normalizeAgentResponseActions([action])).toEqual([action]);
      expect(acceptAgentResponseActions([action], [target])).toEqual([action]);
      expect(acceptAgentResponseActions([action], [])).toEqual([]);
    }
    expect(
      normalizeAgentResponseActions([
        {
          kind: "open",
          target: { ...targets[0], url: "https://example.com" },
        },
      ])
    ).toEqual([]);
  });

  test("offers an unsent comment draft only for the Issue observed this turn", () => {
    const issue = {
      kind: "open-issue",
      issue: issueCommentDraft.issue,
    } as const;
    expect(normalizeAgentResponseActions([issueCommentDraft])).toEqual([
      issueCommentDraft,
    ]);
    expect(acceptAgentResponseActions([issueCommentDraft], [issue])).toEqual([
      issueCommentDraft,
    ]);
    expect(acceptAgentResponseActions([issueCommentDraft], [])).toEqual([]);
    expect(
      acceptAgentResponseActions(
        [issueCommentDraft],
        [{ kind: "open-issue", issue: "projects/sample/issues/8" }]
      )
    ).toEqual([]);
    expect(
      normalizeAgentResponseActions([
        { ...issueCommentDraft, comment: "   " },
        { ...issueCommentDraft, issue: "projects/-/issues/7" },
        { ...issueCommentDraft, url: "https://example.com" },
      ])
    ).toEqual([]);
  });

  test("rejects untyped routes and unsafe follow-up text", () => {
    expect(
      normalizeAgentResponseActions([
        { kind: "open", target: { url: "https://example.com" } },
        { kind: "continue", prompt: "Ask about @other" },
      ])
    ).toEqual([]);
    expect(
      normalizeAgentResponseActions([
        { ...open, target: { ...open.target, url: "https://example.com" } },
      ])
    ).toEqual([]);
    expect(
      normalizeAgentResponseActions([
        { kind: "continue", prompt: "\nWhat should I do next?" },
      ])
    ).toEqual([]);
    expect(
      normalizeAgentResponseActions([
        { kind: "continue", prompt: "What next?", apiRequest: {} },
      ])
    ).toEqual([]);
    expect(
      normalizeAgentResponseActions([
        { kind: "continue", prompt: "Set up an instance instead" },
      ])
    ).toEqual([]);
  });

  test("keeps one primary action and at most three distinct questions in canonical order", () => {
    const questions = [
      followUp,
      { kind: "continue", prompt: "What are the risks?" },
      { kind: "continue", prompt: "How can I verify it?" },
    ] as const;
    expect(
      normalizeAgentResponseActions([
        ...questions,
        { kind: "continue", prompt: "What else?" },
        open,
      ])
    ).toEqual([open, ...questions]);
  });

  test("ignores invalid and case-insensitive duplicate questions without consuming a slot", () => {
    expect(
      normalizeAgentResponseActions([
        { kind: "continue", prompt: "Ask about @other?" },
        followUp,
        { kind: "continue", prompt: "  what should i configure first?  " },
        { kind: "continue", prompt: "What are the risks?" },
        { kind: "continue", prompt: "How can I verify it?" },
      ])
    ).toEqual([
      followUp,
      { kind: "continue", prompt: "What are the risks?" },
      { kind: "continue", prompt: "How can I verify it?" },
    ]);
  });

  test("accepts a Plan draft only for the verified database and one visible SQL block", () => {
    expect(normalizeAgentResponseActions([planDraft])).toEqual([planDraft]);
    expect(
      acceptAgentResponseActions([planDraft], [], {
        verifiedDatabase: {
          name: planDraft.database,
          project: planDraft.project,
        },
        answer: ddlAnswer,
      })
    ).toEqual([planDraft]);
    expect(
      acceptAgentResponseActions([planDraft], [], {
        verifiedDatabase: {
          name: "instances/prod/databases/other",
          project: planDraft.project,
        },
        answer: ddlAnswer,
      })
    ).toEqual([]);
    expect(acceptAgentResponseActions([planDraft], [])).toEqual([]);
    expect(
      acceptAgentResponseActions([planDraft], [], {
        verifiedDatabase: {
          name: planDraft.database,
          project: planDraft.project,
        },
        answer: ddlAnswer.replace("```sql", "```SQL"),
      })
    ).toEqual([planDraft]);
    expect(
      acceptAgentResponseActions([planDraft], [], {
        verifiedDatabase: {
          name: planDraft.database,
          project: planDraft.project,
        },
        answer: "No DDL in this answer.",
      })
    ).toEqual([]);
  });

  test("offers a Plan draft for verified DDL when the Agent omits the action", () => {
    const context = {
      verifiedDatabase: {
        name: planDraft.database,
        project: planDraft.project,
      },
      answer: ddlAnswer,
    };
    expect(acceptAgentResponseActions([], [], context)).toEqual([planDraft]);
    expect(acceptAgentResponseActions([followUp], [], context)).toEqual([
      planDraft,
      followUp,
    ]);
    expect(
      acceptAgentResponseActions([], [], {
        ...context,
        answer: "```sql\nSELECT * FROM notes;\n```",
      })
    ).toEqual([]);
    expect(
      acceptAgentResponseActions([], [], {
        ...context,
        answer: `${ddlAnswer}\n\n\`\`\`sql\nDROP TABLE notes;\n\`\`\``,
      })
    ).toEqual([]);
    expect(acceptAgentResponseActions([], [])).toEqual([]);
  });

  test("rejects ambiguous SQL and competing primary actions", () => {
    expect(
      acceptAgentResponseActions([planDraft], [], {
        verifiedDatabase: {
          name: planDraft.database,
          project: planDraft.project,
        },
        answer: `${ddlAnswer}\n\n\`\`\`sql\nDROP TABLE notes;\n\`\`\``,
      })
    ).toEqual([]);
    expect(normalizeAgentResponseActions([open, planDraft])).toEqual([open]);
    expect(normalizeAgentResponseActions([planDraft, followUp])).toEqual([
      planDraft,
      followUp,
    ]);
    expect(
      normalizeAgentResponseActions([
        { ...planDraft, database: "instances/prod/databases/-" },
      ])
    ).toEqual([]);
  });
});
