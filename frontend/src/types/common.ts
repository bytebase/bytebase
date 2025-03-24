import { UNKNOWN_ID } from "./const";
import type { CommandId, CommandRegisterId } from "./id";
import type { SQLReviewPolicy } from "./sqlReview";

// System bot id
export const SYSTEM_BOT_ID = 1;
// System bot email
export const SYSTEM_BOT_EMAIL = "support@bytebase.com";

// For text input, we do validation if there is no further keystroke after 1s
export const TEXT_VALIDATION_DELAY = 1000;

// Link to inquire enterprise plan
export const ENTERPRISE_INQUIRE_LINK =
  "https://www.bytebase.com/contact-us?source=console";

// Router
export type RouterSlug = {
  issueSlug?: string;
  connectionSlug?: string;
  sheetSlug?: string;
};

// Quick Action Type
export type Command = {
  id: CommandId;
  registerId: CommandRegisterId;
  run: () => void;
};

export type ResourceType = "SQL_REVIEW";

interface ResourceMaker {
  (type: "SQL_REVIEW"): SQLReviewPolicy;
}

const makeUnknown = (type: ResourceType) => {
  const UNKNOWN_SQL_REVIEW_POLICY: SQLReviewPolicy = {
    id: `${UNKNOWN_ID}`,
    enforce: false,
    name: "",
    ruleList: [],
    resources: [],
  };

  switch (type) {
    case "SQL_REVIEW":
      return UNKNOWN_SQL_REVIEW_POLICY;
  }
};

export const unknown = makeUnknown as ResourceMaker;

export interface Pagination {
  pageSize?: number;
  pageToken?: string;
}
