import { useEffect, useState } from "react";
import { useSQLReviewStore } from "@/react/stores/sqlReview";
import {
  ReviewCreation,
  type ReviewCreationProps,
} from "../../components/sql-review/ReviewCreation";

export function SQLReviewCreationPage(props: ReviewCreationProps) {
  return (
    <div className="px-4 pt-4 h-full min-h-0 flex flex-col">
      <ReviewCreation {...props} />
    </div>
  );
}

export function SQLReviewCreatePage() {
  useEffect(() => {
    useSQLReviewStore.getState().fetchReviewPolicyList();
  }, []);

  const [selectedResources] = useState(() => {
    const url = new URL(window.location.href);
    const resource = url.searchParams.get("attachedResource") ?? "";
    return resource ? [resource] : [];
  });

  return (
    <SQLReviewCreationPage
      selectedRuleList={[]}
      selectedResources={selectedResources}
    />
  );
}
