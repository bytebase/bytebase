import { ChevronRight, ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MCPCapabilityList } from "@/components/mcp/MCPCapabilityList";
import { type MCPServingMode, mcpSummaryKey } from "@/components/mcp/mcpPolicy";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

/** The docs table that lists what each capability row covers. */
export const MCP_CAPABILITY_DOCS_URL =
  "https://docs.bytebase.com/integrations/mcp#what-each-mode-allows";

interface Props {
  readonly mode: MCPServingMode;
  readonly expanded: boolean;
  readonly onExpandedChange: (expanded: boolean) => void;
}

/**
 * What a mode allows and what it does not, as capability rows in one ordered
 * list of which every mode's allowed group is a prefix.
 *
 * The served set comes from the mode's tier alone, so there is no comparison
 * against a stored mode and the edit state renders exactly what the view will
 * show after saving. Rows a mode does not serve stay visible in the second
 * group, which is what lets an admin compare two modes without opening a second
 * surface.
 */
export function MCPCapabilityLadder({
  mode,
  expanded,
  onExpandedChange,
}: Props) {
  const { t } = useTranslation();

  return (
    <Collapsible open={expanded} onOpenChange={onExpandedChange}>
      {/* Expanded, the trigger is the first group's heading; it stays inside
          the heading element collapsed too, so toggling keeps its focus. */}
      <h4 className="py-2">
        <CollapsibleTrigger
          className="min-w-0 rounded-xs"
          data-testid="mcp-ladder-trigger"
        >
          <ChevronRight
            className={cn(
              "size-4 shrink-0 text-control-light transition-transform",
              expanded && "rotate-90"
            )}
          />
          <span
            className={cn(
              "min-w-0",
              expanded
                ? "text-base font-semibold text-main"
                : "text-sm text-control-light"
            )}
          >
            {/* The chip or the selector above already names the mode, so
                neither label repeats it. */}
            {expanded
              ? t("settings.mcp.ladder.heading")
              : t(mcpSummaryKey(mode))}
          </span>
        </CollapsibleTrigger>
      </h4>

      <CollapsiblePanel className="flex flex-col gap-y-3">
        <MCPCapabilityList
          mode={mode}
          refusedHeading={t("settings.mcp.ladder.heading-refused")}
          headingLevel={4}
        />
        <a
          href={MCP_CAPABILITY_DOCS_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex w-fit items-center gap-x-1 text-sm text-accent hover:underline"
        >
          {t("settings.mcp.ladder.docs-link")}
          <ExternalLink className="size-3" aria-hidden="true" />
        </a>
      </CollapsiblePanel>
    </Collapsible>
  );
}
