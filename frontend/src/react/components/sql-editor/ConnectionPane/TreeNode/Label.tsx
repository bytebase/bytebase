import { ShieldAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { featureToRef } from "@/store";
import type { SQLEditorTreeNode } from "@/types";
import { NULL_ENVIRONMENT_NAME, UNKNOWN_ENVIRONMENT_NAME } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hexToRgb } from "@/utils";
import { DatabaseNode } from "./DatabaseNode";
import { InstanceNode } from "./InstanceNode";
import { LabelNode } from "./LabelNode";

type Props = {
  readonly node: SQLEditorTreeNode;
  readonly keyword: string;
  readonly checked: boolean;
  readonly checkDisabled?: boolean;
  readonly checkTooltip?: string;
  readonly onCheckedChange?: (checked: boolean) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/TreeNode/Label.vue.
 * Dispatches on `node.meta.type` to render the right leaf component; only
 * the `database` case consumes the checkbox/click callbacks.
 */
export function Label(props: Props) {
  const { node, keyword } = props;
  const type = node.meta.type;

  if (type === "instance") {
    return <InstanceNode node={node} keyword={keyword} />;
  }
  if (type === "environment") {
    return <EnvironmentLabel node={node} keyword={keyword} />;
  }
  if (type === "database") {
    return (
      <DatabaseNode
        node={node}
        keyword={keyword}
        checked={props.checked}
        checkDisabled={props.checkDisabled}
        checkTooltip={props.checkTooltip}
        onCheckedChange={props.onCheckedChange}
      />
    );
  }
  if (type === "label") {
    return <LabelNode node={node} keyword={keyword} />;
  }
  return null;
}

/**
 * Inline environment-name renderer — mirrors the subset of
 * `EnvironmentV1Name` props the tree uses (link=false, keyword highlight,
 * optional color tint, production shield).
 */
function EnvironmentLabel({
  node,
  keyword,
}: {
  node: SQLEditorTreeNode;
  keyword: string;
}) {
  const { t } = useTranslation();
  const environment = (node as SQLEditorTreeNode<"environment">).meta.target;
  const isUnset =
    environment.name === UNKNOWN_ENVIRONMENT_NAME ||
    environment.name === NULL_ENVIRONMENT_NAME;
  const hasEnvTierFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_ENVIRONMENT_TIERS).value
  );
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";
  const bgColorRgb =
    !isUnset && environment.color ? hexToRgb(environment.color) : null;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-x-1 select-none truncate",
        bgColorRgb && "px-1.5 rounded-xs"
      )}
      style={
        bgColorRgb
          ? {
              backgroundColor: `rgba(${bgColorRgb.join(", ")}, 0.1)`,
              color: `rgb(${bgColorRgb.join(", ")})`,
            }
          : undefined
      }
    >
      {isUnset ? (
        // Matches the shared <EnvironmentLabel> treatment: the store's
        // getEnvironmentByName fallback sets `title = id` for unknown envs,
        // so rendering `environment.title` would surface raw "-1" in the
        // tree. Show a localized italic placeholder instead.
        <span className="text-control-light italic">
          {t("common.unassigned")}
        </span>
      ) : (
        <HighlightLabelText text={environment.title} keyword={keyword} />
      )}
      {isProtected && !isUnset && (
        <ShieldAlert className="size-3.5 shrink-0 text-current" />
      )}
    </span>
  );
}
