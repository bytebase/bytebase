// The scrolling part of the plan header — description + metadata. Split out from
// the title/action row (PlanDetailHeader) so only that first row stays sticky on
// scroll, following GitHub's issue-header pattern (BYT-9722). Owns its own
// description-edit state; the title row owns the title + lifecycle slot.
import { create } from "@bufbuild/protobuf";
import { ChevronUp, Plus } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import { Textarea } from "@/react/components/ui/textarea";
import { cn } from "@/react/lib/utils";
import { pushNotification } from "@/store";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { PlanDetailMeta } from "./PlanDetailMeta";

export function PlanDetailHeaderDetails() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, setEditing } = page;
  const currentUser = page.currentUser;
  const project = page.project;
  const draftIssue = page.issue?.draft === true;
  const persistedDescription =
    page.issue && !draftIssue ? page.issue.description : page.plan.description;
  const [description, setDescription] = useState(persistedDescription);
  const [editingDescription, setEditingDescription] = useState(false);
  const [showFullDescription, setShowFullDescription] = useState(false);
  const [updating, setUpdating] = useState(false);
  const pageKeyRef = useRef(page.pageKey);
  pageKeyRef.current = page.pageKey;

  useEffect(() => {
    setDescription(persistedDescription);
    setEditingDescription(false);
    setShowFullDescription(false);
    setUpdating(false);
    setEditing("description", false);
  }, [page.pageKey]);

  useEffect(() => {
    if (!editingDescription) {
      setDescription((prev) =>
        prev === persistedDescription ? prev : persistedDescription
      );
    }
  }, [editingDescription, persistedDescription]);

  const allowDescriptionEdit = useMemo(() => {
    if (page.readonly) return false;
    if (page.isCreating) return true;
    if (!page.issue && page.plan.hasRollout) return false;
    const canUpdatePlan =
      page.plan.creator === currentUser.name ||
      hasProjectPermissionV2(project, "bb.plans.update");
    if (draftIssue) return canUpdatePlan;
    if (page.issue) {
      return hasProjectPermissionV2(project, "bb.issues.update");
    }
    return canUpdatePlan;
  }, [
    currentUser.name,
    draftIssue,
    page.isCreating,
    page.issue,
    page.plan.creator,
    page.plan.hasRollout,
    page.readonly,
    project,
  ]);

  const isDescriptionLong =
    (description?.length ?? 0) > 150 ||
    (description?.split("\n").length ?? 0) > 3;

  const saveDescription = async () => {
    if (page.isCreating) {
      patchState({ plan: { ...page.plan, description } });
      setEditingDescription(false);
      setEditing("description", false);
      return;
    }

    const actionPageKey = page.pageKey;
    let saved = false;
    try {
      setUpdating(true);
      if (page.issue && !draftIssue) {
        const issuePatch = create(IssueSchema, {
          ...page.issue,
          description,
        });
        const response = await issueServiceClientConnect.updateIssue(
          create(UpdateIssueRequestSchema, {
            issue: issuePatch,
            updateMask: { paths: ["description"] },
          })
        );
        if (pageKeyRef.current !== actionPageKey) return;
        patchState({ issue: response });
      } else {
        const planPatch = create(PlanSchema, { ...page.plan, description });
        const response = await planServiceClientConnect.updatePlan(
          create(UpdatePlanRequestSchema, {
            plan: planPatch,
            updateMask: { paths: ["description"] },
          })
        );
        if (pageKeyRef.current !== actionPageKey) return;
        patchState({ plan: response });
      }
      saved = true;
    } catch (error) {
      if (pageKeyRef.current !== actionPageKey) return;
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      if (pageKeyRef.current === actionPageKey) {
        setUpdating(false);
        if (saved) {
          setEditingDescription(false);
          setEditing("description", false);
        }
      }
    }
  };

  return (
    <div className="shrink-0 border-b bg-white px-2 pb-2 sm:px-4">
      <div className="min-w-0">
        {editingDescription ? (
          <div className="py-2">
            <div className="flex items-center justify-between">
              <span className="text-base font-medium">
                {t("common.description")}
              </span>
              <div className="flex items-center gap-2">
                {!page.isCreating ? (
                  <Button
                    disabled={updating}
                    onClick={() => void saveDescription()}
                    size="xs"
                    appearance="outline"
                  >
                    {t("common.save")}
                  </Button>
                ) : (
                  <Button
                    onClick={() => {
                      setDescription(persistedDescription);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    appearance="secondary"
                  >
                    <ChevronUp className="mr-1 size-4" />
                    {t("common.collapse")}
                  </Button>
                )}
                {!page.isCreating && (
                  <Button
                    onClick={() => {
                      setDescription(persistedDescription);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    appearance="secondary"
                  >
                    {t("common.cancel")}
                  </Button>
                )}
              </div>
            </div>
            <Textarea
              className="mt-2 min-h-28"
              onChange={(e) => {
                setDescription(e.target.value);
                if (page.isCreating) {
                  patchState({
                    plan: { ...page.plan, description: e.target.value },
                  });
                }
              }}
              value={description}
            />
          </div>
        ) : description ? (
          <div className="mt-1">
            <div
              aria-disabled={!allowDescriptionEdit}
              className={cn(
                "relative w-full rounded-md border border-transparent px-2 py-1 text-left text-sm text-control-light transition-all duration-200",
                !showFullDescription && "max-h-[4.5rem] overflow-hidden",
                allowDescriptionEdit &&
                  "cursor-pointer hover:border-control-border"
              )}
              onClick={() => {
                if (!allowDescriptionEdit) return;
                setEditingDescription(true);
                setEditing("description", true);
              }}
              onKeyDown={(event) => {
                if (!allowDescriptionEdit) return;
                if (event.key !== "Enter" && event.key !== " ") return;
                event.preventDefault();
                setEditingDescription(true);
                setEditing("description", true);
              }}
              role={allowDescriptionEdit ? "button" : undefined}
              tabIndex={allowDescriptionEdit ? 0 : undefined}
            >
              <div className="pointer-events-none">
                <MarkdownEditor content={description} mode="preview" />
              </div>
              {!showFullDescription && isDescriptionLong && (
                <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-6 bg-gradient-to-t from-background to-transparent" />
              )}
            </div>
            {isDescriptionLong && (
              <Button
                className="mt-1 px-2 text-control-placeholder hover:text-control"
                onClick={(event) => {
                  event.stopPropagation();
                  setShowFullDescription((value) => !value);
                }}
                size="xs"
                appearance="link"
              >
                {showFullDescription
                  ? t("common.show-less")
                  : t("common.show-more")}
              </Button>
            )}
          </div>
        ) : allowDescriptionEdit ? (
          <Button
            className="italic opacity-60"
            onClick={() => {
              setEditingDescription(true);
              setEditing("description", true);
            }}
            size="xs"
            appearance="secondary"
          >
            <Plus className="mr-1 size-4" />
            {t("plan.description.placeholder")}
          </Button>
        ) : null}
      </div>

      <PlanDetailMeta />
    </div>
  );
}
