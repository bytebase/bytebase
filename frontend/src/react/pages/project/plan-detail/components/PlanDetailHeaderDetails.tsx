// The scrolling part of the plan header — description + metadata. Split out from
// the title/action row (PlanDetailHeader) so only that first row stays sticky on
// scroll, following GitHub's issue-header pattern (BYT-9722). Owns its own
// description-edit state; the title row owns the title + lifecycle slot.
import { create } from "@bufbuild/protobuf";
import { ChevronUp, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/connect";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import { Textarea } from "@/react/components/ui/textarea";
import { cn } from "@/react/lib/utils";
import { pushNotification } from "@/store";
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
  const [description, setDescription] = useState(page.plan.description);
  const [editingDescription, setEditingDescription] = useState(false);
  const [showFullDescription, setShowFullDescription] = useState(false);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    setDescription((prev) =>
      prev === page.plan.description ? prev : page.plan.description
    );
  }, [page.plan.description]);

  const allowDescriptionEdit = useMemo(() => {
    if (page.readonly) return false;
    if (page.isCreating) return true;
    if (!page.issue && page.plan.hasRollout) return false;
    return (
      page.plan.creator === currentUser.name ||
      hasProjectPermissionV2(project, "bb.plans.update")
    );
  }, [
    currentUser.name,
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

    let saved = false;
    try {
      setUpdating(true);
      const planPatch = create(PlanSchema, { ...page.plan, description });
      const response = await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: planPatch,
          updateMask: { paths: ["description"] },
        })
      );
      patchState({ plan: response });
      saved = true;
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setUpdating(false);
      if (saved) {
        setEditingDescription(false);
        setEditing("description", false);
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
                    variant="outline"
                  >
                    {t("common.save")}
                  </Button>
                ) : (
                  <Button
                    onClick={() => {
                      setDescription(page.plan.description);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    variant="ghost"
                  >
                    <ChevronUp className="mr-1 h-4 w-4" />
                    {t("common.collapse")}
                  </Button>
                )}
                {!page.isCreating && (
                  <Button
                    onClick={() => {
                      setDescription(page.plan.description);
                      setEditingDescription(false);
                      setEditing("description", false);
                    }}
                    size="xs"
                    variant="ghost"
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
                allowDescriptionEdit && "cursor-pointer hover:border-gray-200"
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
                <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-6 bg-gradient-to-t from-white to-transparent" />
              )}
            </div>
            {isDescriptionLong && (
              <button
                className="mt-1 px-2 text-xs text-control-placeholder hover:text-control"
                onClick={(event) => {
                  event.stopPropagation();
                  setShowFullDescription((value) => !value);
                }}
                type="button"
              >
                {showFullDescription
                  ? t("common.show-less")
                  : t("common.show-more")}
              </button>
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
            variant="ghost"
          >
            <Plus className="mr-1 h-4 w-4" />
            {t("plan.description.placeholder")}
          </Button>
        ) : null}
      </div>

      <PlanDetailMeta />
    </div>
  );
}
