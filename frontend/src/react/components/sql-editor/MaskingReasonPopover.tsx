import { EyeOff } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { hasFeature, useProjectV1Store } from "@/store";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";

interface Props {
  readonly reason: MaskingReason;
  readonly statement?: string;
  readonly database?: string;
  readonly onClick?: () => void;
}

export function MaskingReasonPopover({
  reason,
  statement,
  database,
  onClick,
}: Props) {
  const { t } = useTranslation();
  const [showDrawer, setShowDrawer] = useState(false);

  const projectStore = useProjectV1Store();
  const editorStore = useSQLEditorVueState();

  const projectName = useVueState(() => editorStore.project);
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const hasJITFeature = useMemo(
    () =>
      !!project?.allowJustInTimeAccess && hasFeature(PlanFeature.FEATURE_JIT),
    [project]
  );

  const targets = useMemo(() => (database ? [database] : []), [database]);

  const formatAlgorithm = (algorithm: string): string => {
    const algorithmKey = algorithm.toLowerCase().replace(/\s+/g, "-");
    const key = `masking.algorithms.${algorithmKey}`;
    const translated = t(key);
    if (translated === key) {
      return algorithm;
    }
    return translated;
  };

  const handleTriggerClick = () => {
    onClick?.();
  };

  return (
    <>
      <Popover>
        <PopoverTrigger
          openOnHover
          delay={100}
          render={
            <div className="inline-flex items-center gap-0.5 cursor-pointer">
              {reason.semanticTypeIcon && (
                <img
                  src={reason.semanticTypeIcon}
                  className="size-3 object-contain"
                  alt=""
                />
              )}
              <EyeOff
                className="size-3 text-control-placeholder hover:text-control"
                onClick={handleTriggerClick}
              />
            </div>
          }
        />
        <PopoverContent
          side="bottom"
          align="start"
          className="min-w-80 max-w-md"
        >
          <div className="w-full flex flex-col gap-y-2">
            <div className="font-medium flex items-center gap-2">
              {reason.semanticTypeIcon && (
                <img
                  src={reason.semanticTypeIcon}
                  className="size-4 object-contain"
                  alt=""
                />
              )}
              {t("masking.reason.title")}
            </div>

            {reason.semanticTypeTitle && (
              <div className="text-sm">
                <span className="text-control-placeholder">
                  {t("masking.reason.semantic-type")}:
                </span>
                <span className="ml-1">{reason.semanticTypeTitle}</span>
              </div>
            )}

            {reason.algorithm && (
              <div className="text-sm">
                <span className="text-control-placeholder">
                  {t("masking.reason.algorithm")}:
                </span>
                <span className="ml-1">
                  {formatAlgorithm(reason.algorithm)}
                </span>
              </div>
            )}

            {reason.context && (
              <div className="text-sm">
                <span className="text-control-placeholder">
                  {t("masking.reason.context")}:
                </span>
                <span className="ml-1">{reason.context}</span>
              </div>
            )}

            {!!reason.classificationLevel && (
              <div className="text-sm">
                <span className="text-control-placeholder">
                  {t("masking.reason.classification")}:
                </span>
                <span className="ml-1">{reason.classificationLevel}</span>
              </div>
            )}

            {hasJITFeature && statement && (
              <Button
                size="sm"
                variant="default"
                className="self-start"
                onClick={() => setShowDrawer(true)}
              >
                {t("sql-editor.request-jit")}
              </Button>
            )}
          </div>
        </PopoverContent>
      </Popover>

      {showDrawer && (
        <AccessGrantRequestDrawer
          query={statement}
          targets={targets}
          unmask={true}
          onClose={() => setShowDrawer(false)}
        />
      )}
    </>
  );
}
