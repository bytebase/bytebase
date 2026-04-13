import { useTranslation } from "react-i18next";

import { IssueDetailCommentList } from "./IssueDetailCommentList";

export function IssueDetailActivity() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-y-4">
      <h3 className="text-base font-medium">{t("common.activity")}</h3>
      <IssueDetailCommentList />
    </div>
  );
}
