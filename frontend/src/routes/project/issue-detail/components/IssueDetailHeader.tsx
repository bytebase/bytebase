import { Ban, CheckCircle2, Menu } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { IssueDetailActionBar } from "./IssueDetailActionBar";
import { IssueDetailTitleInput } from "./IssueDetailTitleInput";

export function IssueDetailHeader() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const showClosedTag = page.issue?.status === IssueStatus.CANCELED;
  const showDoneTag = page.issue?.status === IssueStatus.DONE;
  const doneTagLabel =
    page.issue?.approvalStatus === ApprovalStatus.APPROVED
      ? t("common.approved")
      : t("common.skipped");
  const doneTagVariant =
    page.issue?.approvalStatus === ApprovalStatus.APPROVED
      ? "success"
      : "default";
  const showMobileSidebarButton = page.sidebarMode === "MOBILE";

  return (
    <div className="px-2 py-2 sm:px-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex min-w-0 basis-full flex-1 items-center gap-x-2 sm:basis-auto">
          {showClosedTag && (
            <Badge className="shrink-0 gap-x-1.5" size="sm" variant="default">
              <Ban className="h-4 w-4" />
              {t("common.closed")}
            </Badge>
          )}
          {showDoneTag && !showClosedTag && (
            <Badge
              className="shrink-0 gap-x-1.5"
              size="sm"
              variant={doneTagVariant}
            >
              <CheckCircle2 className="h-4 w-4" />
              {doneTagLabel}
            </Badge>
          )}

          <IssueDetailTitleInput />
        </div>

        <div className="ml-auto flex w-full min-w-0 max-w-full flex-wrap items-center justify-end gap-x-2 gap-y-2 sm:w-auto">
          <IssueDetailActionBar />
          {showMobileSidebarButton && (
            <Button
              aria-label={t("common.open")}
              onClick={() => page.setMobileSidebarOpen(true)}
              size="sm"
              appearance="secondary"
            >
              <Menu className="h-5 w-5" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
