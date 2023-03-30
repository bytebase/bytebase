import { ReviewFlow } from "@/types";
import { Review } from "@/types/proto/v1/review_service";
import { inject, provide, type InjectionKey, type Ref } from "vue";

export type IssueReviewContext = {
  review: Ref<Review>;

  /// derived states
  // true if the approval flow is generated
  ready: Ref<boolean>;
  // The review flow.
  // Now we have only one flow in an issue
  flow: Ref<ReviewFlow>;
  // Whether the review flow is finished successfully.
  done: Ref<boolean>;
  // Whether the review finding has error.
  error: Ref<string | undefined>;
};

export const KEY = Symbol(
  "bb.issue.review"
) as InjectionKey<IssueReviewContext>;

export const useIssueReviewContext = () => {
  return inject(KEY)!;
};

export const provideIssueReviewContext = (context: IssueReviewContext) => {
  provide(KEY, context);
};
