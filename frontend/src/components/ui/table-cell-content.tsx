import * as stylex from "@stylexjs/stylex";
import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/lib/utils";
import { tableStyles } from "./styles.stylex";

export function TableCellContent({
  className,
  style,
  ...props
}: ComponentPropsWithoutRef<"div">) {
  const stylexProps = stylex.props(tableStyles.cellContent);
  return (
    <div
      {...props}
      className={cn(
        "flex min-w-0 items-center",
        stylexProps.className,
        className
      )}
      style={{ ...stylexProps.style, ...style }}
    />
  );
}
