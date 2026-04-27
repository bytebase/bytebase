import type { ReactNode } from "react";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { cn } from "@/react/lib/utils";

type Props = {
  readonly text: string;
  readonly indent?: number;
  readonly keyword?: string;
  readonly highlight?: boolean;
  readonly hideIcon?: boolean;
  readonly icon?: ReactNode;
  readonly suffix?: ReactNode;
  /**
   * Renders when `text` is empty (e.g. SchemaNode falling back to "default").
   */
  readonly fallbackText?: ReactNode;
  /**
   * Override the default rendered children entirely. Mirrors Vue's
   * `<template #default>` slot — used by `TextNode` to inject custom
   * content when `node.meta.target.render` is set.
   */
  readonly children?: ReactNode;
  readonly className?: string;
};

/**
 * Replaces `frontend/src/views/sql-editor/AsidePanel/SchemaPane/TreeNode/CommonNode.vue`.
 *
 * Layout: [optional indent spacers] [icon] [text|highlight|fallback] [suffix].
 * Mirrors the Vue widths (20×20 indent + icon, 2px text padding) so rows
 * align across the Vue→React swap.
 */
export function CommonNode({
  text,
  indent,
  keyword,
  highlight,
  hideIcon,
  icon,
  suffix,
  fallbackText,
  children,
  className,
}: Props) {
  if (children !== undefined) {
    return (
      <div
        className={cn(
          "flex items-center max-w-full overflow-hidden",
          className
        )}
      >
        {children}
      </div>
    );
  }
  return (
    <div
      className={cn("flex items-center max-w-full overflow-hidden", className)}
    >
      {indent
        ? Array.from({ length: indent }, (_, i) => (
            <span
              key={`indent-#${i}`}
              className="inline-block w-[20px] h-[20px] shrink-0 invisible"
              data-indent={i + 1}
            />
          ))
        : null}
      {!hideIcon ? (
        <span className="flex items-center justify-center shrink-0 w-[20px] h-[20px]">
          {icon}
        </span>
      ) : null}
      {text ? (
        highlight ? (
          <HighlightLabelText
            text={text}
            keyword={keyword ?? ""}
            className="flex-1 truncate pl-[2px] min-w-16"
          />
        ) : (
          <span className="pl-[2px]">{text}</span>
        )
      ) : (
        fallbackText
      )}
      {suffix}
    </div>
  );
}
