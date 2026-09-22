import { ChevronDown, ChevronRight } from "lucide-react";
import { Button, type ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Section } from "./types";

// The status glyph shared by the collapsible header and the sole-section
// (non-collapsible) label in TaskRunLogViewer, so the two can't drift.
export function SectionStatusIcon({ section }: { section: Section }) {
  const StatusIcon = section.statusIcon;
  return (
    <StatusIcon
      className={cn(
        "h-3.5 w-3.5 shrink-0",
        section.statusClass,
        section.status === "running" && "animate-spin"
      )}
    />
  );
}

export interface SectionHeaderProps extends Omit<ButtonProps, "onClick"> {
  section: Section;
  isExpanded: boolean;
  indent?: boolean;
  onToggle: () => void;
}

export function SectionHeader({
  section,
  isExpanded,
  indent = false,
  onToggle,
  className,
  ...props
}: SectionHeaderProps) {
  return (
    <Button
      type="button"
      aria-expanded={isExpanded}
      appearance="secondary"
      size="sm"
      className={cn(
        "w-full justify-start rounded-none bg-background text-left hover:bg-control-bg",
        indent ? "px-6" : "px-3",
        className
      )}
      onClick={onToggle}
      {...props}
    >
      {isExpanded ? (
        <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
      ) : (
        <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
      )}
      <SectionStatusIcon section={section} />
      <span className="text-control">{section.label}</span>
      {section.entryCount > 1 ? (
        <span className="text-control-placeholder">({section.entryCount})</span>
      ) : null}
      <span className="flex-1" />
      {section.duration ? (
        <span className="text-control-light tabular-nums">
          {section.duration}
        </span>
      ) : null}
    </Button>
  );
}

export default SectionHeader;
