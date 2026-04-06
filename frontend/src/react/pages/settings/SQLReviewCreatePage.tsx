import { useEffect, useState } from "react";
import { useSQLReviewStore } from "@/store";
import { ReviewCreation } from "../../components/sql-review/ReviewCreation";

export function SQLReviewCreatePage() {
  const sqlReviewStore = useSQLReviewStore();

  useEffect(() => {
    sqlReviewStore.fetchReviewPolicyList();
  }, [sqlReviewStore]);

  const [selectedResources] = useState(() => {
    const url = new URL(window.location.href);
    const resource = url.searchParams.get("attachedResource") ?? "";
    return resource ? [resource] : [];
  });

  return (
    <div className="px-4 py-4 gap-y-4 h-full flex flex-col">
      <ReviewCreation
        selectedRuleList={[]}
        selectedResources={selectedResources}
      />
    </div>
  );
}
