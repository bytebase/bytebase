import { UNKNOWN_ID } from "./const";
import type { CommandId, CommandRegisterId } from "./id";
import type { SQLReviewPolicy } from "./sqlReview";

// System bot id
export const SYSTEM_BOT_ID = 1;
// System bot email
export const SYSTEM_BOT_EMAIL = "support@bytebase.com";

// The delay for debouncing search input.
export const DEBOUNCE_SEARCH_DELAY = 300;

// Link to inquire enterprise plan
export const ENTERPRISE_INQUIRE_LINK =
  "https://www.bytebase.com/contact-us?source=console";

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
