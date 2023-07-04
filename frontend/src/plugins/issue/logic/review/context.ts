import { ReviewFlow } from "@/types";
import { Issue, Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { inject, provide, type InjectionKey, type Ref } from "vue";

export type IssueReviewContext = {
  issue: Ref<Issue>;

  /// derived states
  // true if the approval flow is generated
  ready: Ref<boolean>;
  // The review flow.
  // Now we have only one flow in an issue
  flow: Ref<ReviewFlow>;
  // The overall status of the entire review flow
  status: Ref<Issue_Approver_Status>;
  // Whether the review flow is finished successfully.
  // A shortcut to `status === Review_Approver_Status.APPROVED`
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
