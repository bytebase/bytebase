import type { ReactNode } from "react";

type Props = {
  readonly title?: string;
  readonly titleSlot?: ReactNode;
  readonly children: ReactNode;
};

/**
 * Replaces `HoverPanel/InfoItem.vue`. A 2-column grid row: gray title on
 * the left, value (right-aligned, ellipsis on overflow) on the right.
 *
 * Mirrors the Vue layout — `grid-template-columns: auto 1fr` + 16px gap
 * — so the title hugs its content and the value column truncates.
 */
export function InfoItem({ title, titleSlot, children }: Props) {
  return (
    <div
      className="w-full grid gap-x-4 items-center"
      style={{ gridTemplateColumns: "auto 1fr" }}
    >
      <div className="text-gray-500 font-medium whitespace-nowrap">
        {titleSlot ?? title}
      </div>
      <div className="text-right flex items-center justify-end min-w-0">
        <span className="truncate flex items-center leading-[20px] h-[20px]">
          {children}
        </span>
      </div>
    </div>
  );
}
