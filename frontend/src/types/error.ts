export enum GeneralErrorCode {
  OK = 0,
}

export enum SQLReviewPolicyErrorCode {
  EMPTY_POLICY = 2,
}

export type ErrorCode = GeneralErrorCode | SQLReviewPolicyErrorCode | number;
