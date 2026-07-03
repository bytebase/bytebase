import * as stylex from "@stylexjs/stylex";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";
import {
  stickyActionFooterRightStyle,
  stickyActionFooterSideStyle,
  stickyActionFooterStyle,
} from "./styles.stylex";

interface StickyActionFooterProps extends ComponentProps<"div"> {
  left?: ReactNode;
  right?: ReactNode;
  leftClassName?: string;
  rightClassName?: string;
}

function StickyActionFooter({
  className,
  left,
  leftClassName,
  right,
  rightClassName,
  ref,
  style,
  ...props
}: StickyActionFooterProps) {
  const rootStylexProps = stylex.props(stickyActionFooterStyle());
  const leftStylexProps = stylex.props(stickyActionFooterSideStyle());
  const rightStylexProps = stylex.props(
    stickyActionFooterSideStyle(),
    stickyActionFooterRightStyle()
  );

  return (
    <div
      ref={ref}
      data-slot="sticky-action-footer"
      className={cn(
        "sticky bottom-0 z-10 border-t border-block-border bg-background py-4",
        rootStylexProps.className,
        className
      )}
      style={{ ...rootStylexProps.style, ...style }}
      {...props}
    >
      <div
        data-slot="sticky-action-footer-left"
        className={cn(leftStylexProps.className, leftClassName)}
        style={leftStylexProps.style}
      >
        {left}
      </div>
      <div
        data-slot="sticky-action-footer-right"
        className={cn("gap-x-2", rightStylexProps.className, rightClassName)}
        style={rightStylexProps.style}
      >
        {right}
      </div>
    </div>
  );
}

export { StickyActionFooter };
