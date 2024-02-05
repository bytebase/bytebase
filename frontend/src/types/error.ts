export enum GeneralErrorCode {
  OK = 0,
}

export enum SQLReviewPolicyErrorCode {
  EMPTY_POLICY = 2,
}

export type ErrorCode = GeneralErrorCode | SQLReviewPolicyErrorCode | number;

export type ErrorTag = "General" | "Compatibility";

export type ErrorMeta = {
  code: ErrorCode;
  hash: string;
};
