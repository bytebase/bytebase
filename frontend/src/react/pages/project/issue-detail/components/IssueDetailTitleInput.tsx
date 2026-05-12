import { create } from "@bufbuild/protobuf";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailTitleInput() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const { setEditing } = page;
  const projectStore = useProjectV1Store();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const [title, setTitle] = useState(page.issue?.title || "");
  const [isEditing, setIsEditing] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    setTitle(page.issue?.title || page.plan?.title || "");
  }, [page.issue?.title, page.plan?.title]);

  useEffect(() => {
    return () => {
      setEditing("title", false);
    };
  }, [setEditing]);

  const allowEdit = useMemo(() => {
    if (page.readonly || !page.issue || !project) {
      return false;
    }
    return hasProjectPermissionV2(project, "bb.issues.update");
  }, [page.issue, page.readonly, project]);
  const isReadOnly = !allowEdit || isUpdating;

  const handleBlur = async () => {
    setEditing("title", false);
    setIsEditing(false);
    if (!page.issue || title === page.issue.title) {
      setTitle(page.issue?.title || "");
      return;
    }

    try {
      setIsUpdating(true);
      const issuePatch = create(IssueSchema, {
        ...page.issue,
        title,
      });
      const request = create(UpdateIssueRequestSchema, {
        issue: issuePatch,
        updateMask: { paths: ["title"] },
      });
      const response = await issueServiceClientConnect.updateIssue(request);
      page.patchState({ issue: response });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      console.error("Failed to update issue title:", error);
      setTitle(page.issue.title);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <div className="relative min-w-0 flex-1">
      <input
        className={[
          "h-10 w-full bg-transparent text-xl! font-bold text-main transition-colors outline-hidden",
          "placeholder:text-control-placeholder",
          "caret-main",
          isReadOnly ? "cursor-default" : "cursor-text",
          isEditing
            ? "border border-control-border bg-transparent px-3"
            : "border border-transparent bg-transparent px-0",
        ].join(" ")}
        maxLength={200}
        onBlur={handleBlur}
        onChange={(e) => setTitle(e.target.value)}
        onFocus={() => {
          if (isReadOnly) {
            return;
          }
          setEditing("title", true);
          setIsEditing(true);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.nativeEvent.isComposing) {
            e.currentTarget.blur();
          }
        }}
        required
        readOnly={isReadOnly}
        value={title}
      />
      {isUpdating && (
        <div className="pointer-events-none absolute inset-y-0 right-2 flex items-center">
          <Loader2 className="h-4 w-4 animate-spin text-control-light" />
        </div>
      )}
    </div>
  );
}
