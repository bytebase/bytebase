import * as stylex from "@stylexjs/stylex";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { Alert, type AlertProps } from "./ui/alert";

const styles = stylex.create({
  root: {
    display: "flex",
    flexDirection: "column",
    overflowX: "clip",
    paddingBlock: 16,
    rowGap: 16,
    width: "100%",
  },
  pagePadding: {
    paddingInline: 16,
  },
  content: {
    minWidth: 0,
  },
  footer: {
    marginInline: 8,
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

type WorkspacePageLayoutProps = ComponentProps<"div"> & {
  padding?: "page" | "flush";
};

function WorkspacePageLayout({
  className,
  padding = "page",
  ref,
  style,
  ...props
}: WorkspacePageLayoutProps) {
  const stylexProps = stylex.props(
    styles.root,
    padding === "page" && styles.pagePadding
  );
  return (
    <div
      ref={ref}
      data-slot="workspace-page-layout"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

type WorkspacePageToolbarProps = ComponentProps<"div"> & {
  align?: "start" | "end" | "between";
};

function WorkspacePageToolbar({
  align = "between",
  className,
  ref,
  style,
  ...props
}: WorkspacePageToolbarProps) {
  const stylexProps = stylex.props(
    styles.toolbar,
    align === "start" && styles.toolbarStart,
    align === "end" && styles.toolbarEnd,
    align === "between" && styles.toolbarBetween
  );
  return (
    <div
      ref={ref}
      data-slot="workspace-page-toolbar"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function WorkspacePageContent({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(styles.content);
  return (
    <div
      ref={ref}
      data-slot="workspace-page-content"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function WorkspacePageFooter({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(styles.footer);
  return (
    <div
      ref={ref}
      data-slot="workspace-page-footer"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

type WorkspacePageInfoProps = Omit<AlertProps, "onDismiss" | "variant"> & {
  variant?: Extract<AlertProps["variant"], "info" | "warning" | "error">;
};

function WorkspacePageInfo({
  variant = "info",
  ...props
}: WorkspacePageInfoProps) {
  return <Alert role="note" variant={variant} {...props} />;
}

export {
  WorkspacePageContent,
  WorkspacePageFooter,
  WorkspacePageInfo,
  WorkspacePageLayout,
  WorkspacePageToolbar,
};
