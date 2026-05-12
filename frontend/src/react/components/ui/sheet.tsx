import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { cva, type VariantProps } from "class-variance-authority";
import { X } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
  usePreserveHigherLayerAccess,
} from "./layer";

// ---- Root ----
// A side sheet is a Dialog whose Popup is pinned to the right edge of the
// viewport. We reuse Base UI's Dialog primitive for focus trap, Escape key,
// outside-click dismissal, and ARIA wiring — a sheet is semantically a dialog,
// just with a different layout.
const Sheet = BaseDialog.Root;

// ---- Trigger ----
const SheetTrigger = BaseDialog.Trigger;

// ---- Close ----
const SheetClose = BaseDialog.Close;

// ---- Backdrop ----
function SheetOverlay({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Backdrop>) {
  return (
    <BaseDialog.Backdrop
      ref={ref}
      className={cn(
        `fixed inset-0 ${LAYER_BACKDROP_CLASS} bg-overlay/50`,
        "data-[starting-style]:opacity-0 data-[ending-style]:opacity-0",
        "transition-opacity duration-200",
        className
      )}
      {...props}
    />
  );
}

// ---- Content (Portal + Overlay + Popup) ----
// Width tiers codify our resource-edit drawer conventions:
//   narrow   (384px) — single-field pickers, short forms
//   standard (704px) — 3-6 field forms, permission transfer lists
//   wide     (832px) — forms with CEL builders, nested tables, multi-tab layouts
// Do not inline ad-hoc widths on SheetContent — add a tier here if a new
// legitimate size is needed so all consumers stay aligned.
const sheetContentVariants = cva(
  cn(
    "fixed inset-y-0 right-0 flex h-full flex-col bg-background shadow-lg",
    "max-w-[100vw] outline-hidden",
    "data-[starting-style]:translate-x-full data-[ending-style]:translate-x-full",
    "transition-transform duration-200 ease-out"
  ),
  {
    variants: {
      width: {
        narrow: "w-[24rem]",
        panel: "w-[31.25rem]",
        medium: "w-[40rem]",
        standard: "w-[44rem]",
        wide: "w-[52rem]",
        large: "w-[64rem]",
        xlarge: "w-[70rem]",
        // Maximized editor surfaces (e.g. plan-detail schema editor). Leaves
        // a ~5vw strip on the left as a visual anchor; clicking the strip
        // closes the sheet like any other scrim click.
        huge: "w-[95vw]",
        workspace: "w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)]",
      },
    },
    defaultVariants: {
      width: "standard",
    },
  }
);

interface SheetContentProps
  extends ComponentProps<typeof BaseDialog.Popup>,
    VariantProps<typeof sheetContentVariants> {}

function SheetContent({
  className,
  children,
  width,
  ref,
  ...props
}: SheetContentProps) {
  usePreserveHigherLayerAccess("overlay");

  return (
    <BaseDialog.Portal container={getLayerRoot("overlay")}>
      <SheetOverlay />
      <BaseDialog.Popup
        ref={ref}
        className={cn(
          sheetContentVariants({ width }),
          LAYER_SURFACE_CLASS,
          className
        )}
        {...props}
      >
        {children}
      </BaseDialog.Popup>
    </BaseDialog.Portal>
  );
}

// ---- Header ----
// Sticky top region with a bottom border, laid out as a row so the built-in
// close button sits on the right. Typically contains a `SheetTitle` and an
// optional `SheetDescription` — both are wrapped in a flex-col for the
// vertical stack layout while the close button remains flush right. `actions`
// renders secondary icon-buttons (maximize, settings, etc.) immediately
// before the close button.
function SheetHeader({
  className,
  children,
  actions,
  ...props
}: ComponentProps<"div"> & { actions?: ReactNode }) {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        "flex items-start justify-between gap-x-4 border-b border-control-border px-6 py-4",
        className
      )}
      {...props}
    >
      <div className="flex flex-col gap-y-1 min-w-0 flex-1">{children}</div>
      {actions ? (
        <div className="flex items-center gap-x-1 shrink-0">{actions}</div>
      ) : null}
      {/* Built-in close affordance. Callers should not render their own close
          button — Base UI's Close dismisses the Sheet via Root's onOpenChange. */}
      <BaseDialog.Close
        aria-label={t("common.close")}
        className="shrink-0 rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
      >
        <X className="size-4" />
      </BaseDialog.Close>
    </div>
  );
}

// ---- Body ----
// Scrollable middle region between header and footer. Uses flex-1 so it
// absorbs remaining vertical space.
function SheetBody({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex flex-1 flex-col overflow-y-auto px-6 py-4",
        className
      )}
      {...props}
    />
  );
}

// ---- Footer ----
// Sticky bottom region with a top border. Use `gap-x-2` for button groups,
// matching the BUTTON_SPACING_STANDARDIZATION convention.
function SheetFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex items-center justify-end gap-x-2 border-t border-control-border px-6 py-4",
        className
      )}
      {...props}
    />
  );
}

// ---- Title ----
function SheetTitle({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Title>) {
  return (
    <BaseDialog.Title
      ref={ref}
      className={cn("text-lg font-semibold text-control", className)}
      {...props}
    />
  );
}

// ---- Description ----
function SheetDescription({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Description>) {
  return (
    <BaseDialog.Description
      ref={ref}
      className={cn("text-sm text-control-light", className)}
      {...props}
    />
  );
}

export {
  Sheet,
  SheetBody,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetOverlay,
  SheetTitle,
  SheetTrigger,
};
