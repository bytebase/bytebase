import type { GuideJourneyId } from "./types";

export const GUIDE_PROGRESS_KEYS = {
  databaseExplored: "workspace-setup-guide.database-explored",
  statementRun: "workspace-setup-guide.statement-run",
  changeIssueCreated: "workspace-setup-guide.change-issue-created",
  teammateAdded: "workspace-setup-guide.teammate-added",
  dismissed: "workspace-setup-guide.dismissed",
} as const;

export const guideCompletionAcknowledgedKey = (id: GuideJourneyId) =>
  `workspace-setup-guide.completed.${id}`;
