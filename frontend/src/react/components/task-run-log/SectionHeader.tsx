import { ChevronDown, ChevronRight } from "lucide-react";
import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/react/lib/utils";
import type { Section } from "./types";

export interface SectionHeaderProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "onClick"> {
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
  const StatusIcon = section.statusIcon;

  return (
    <button
      type="button"
      aria-expanded={isExpanded}
      className={cn(
        "flex w-full items-center gap-x-2 py-1.5 bg-white hover:bg-gray-50 cursor-pointer select-none text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        indent ? "px-6" : "px-3",
        className
      )}
      onClick={onToggle}
      {...props}
    >
      {isExpanded ? (
        <ChevronDown className="h-3.5 w-3.5 shrink-0 text-gray-400" />
      ) : (
        <ChevronRight className="h-3.5 w-3.5 shrink-0 text-gray-400" />
      )}
      <StatusIcon
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          section.statusClass,
          section.status === "running" && "animate-spin"
        )}
      />
      <span className="text-gray-700">{section.label}</span>
      {section.entryCount > 1 ? (
        <span className="text-gray-400">({section.entryCount})</span>
      ) : null}
      <span className="flex-1" />
      {section.duration ? (
        <span className="text-gray-500 tabular-nums">{section.duration}</span>
      ) : null}
    </button>
  );
}

export default SectionHeader;
