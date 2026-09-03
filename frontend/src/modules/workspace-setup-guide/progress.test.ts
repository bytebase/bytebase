import { describe, expect, test } from "vitest";
import {
  GUIDE_PROGRESS_KEYS,
  guideCompletionAcknowledgedKey,
} from "./progress";

describe("workspace setup guide progress keys", () => {
  test("keeps interaction evidence under the guide namespace", () => {
    expect(GUIDE_PROGRESS_KEYS).toEqual({
      databaseExplored: "workspace-setup-guide.database-explored",
      statementRun: "workspace-setup-guide.statement-run",
      changeIssueCreated: "workspace-setup-guide.change-issue-created",
      teammateAdded: "workspace-setup-guide.teammate-added",
      dismissed: "workspace-setup-guide.dismissed",
    });
  });

  test("scopes completion acknowledgement by journey", () => {
    expect(guideCompletionAcknowledgedKey("query-data")).toBe(
      "workspace-setup-guide.completed.query-data"
    );
    expect(guideCompletionAcknowledgedKey("workspace-setup")).toBe(
      "workspace-setup-guide.completed.workspace-setup"
    );
  });
});
