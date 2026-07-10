import * as stylex from "@stylexjs/stylex";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { Alert, type AlertProps } from "./ui/alert";

const styles = stylex.create({
  root: {
    display: "flex",
    flexDirection: "column",
    overflowX: "hidden",
    paddingBlock: 16,
    paddingInline: 16,
    rowGap: 16,
    width: "100%",
  },
  content: {
    minWidth: 0,
  },
  footer: {
    marginTop: 16,
  },
  toolbar: {
    alignItems: "center",
    columnGap: 8,
    display: "flex",
    justifyContent: "space-between",
  },
  toolbarStart: {
    justifyContent: "flex-start",
  },
  toolbarEnd: {
    justifyContent: "flex-end",
  },
  toolbarBetween: {
    justifyContent: "space-between",
  },
});

function ProjectPageLayout({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(styles.root);
  return (
    <div
      ref={ref}
      data-slot="project-page-layout"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

type ProjectPageToolbarProps = ComponentProps<"div"> & {
  align?: "start" | "end" | "between";
};

function ProjectPageToolbar({
  align = "between",
  className,
  ref,
  style,
  ...props
}: ProjectPageToolbarProps) {
  const stylexProps = stylex.props(
    styles.toolbar,
    align === "start" && styles.toolbarStart,
    align === "end" && styles.toolbarEnd,
    align === "between" && styles.toolbarBetween
  );
  return (
    <div
      ref={ref}
      data-slot="project-page-toolbar"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

type ProjectPageInfoProps = Omit<AlertProps, "onDismiss" | "variant"> & {
  variant?: Extract<AlertProps["variant"], "info" | "warning" | "error">;
};

function ProjectPageInfo({ variant = "info", ...props }: ProjectPageInfoProps) {
  return <Alert variant={variant} {...props} />;
}

function ProjectPageContent({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(styles.content);
  return (
    <div
      ref={ref}
      data-slot="project-page-content"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function ProjectPageFooter({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(styles.footer);
  return (
    <div
      ref={ref}
      data-slot="project-page-footer"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

export {
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageInfo,
  ProjectPageLayout,
  ProjectPageToolbar,
};
