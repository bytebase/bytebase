import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { cva, type VariantProps } from "class-variance-authority";
import { X } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

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
// Dialog, Select, Tooltip, AlertDialog, DropdownMenu all share `z-50`. Within
// that layer, stacking falls back to DOM portal mount order — later mounts
// win — which correctly places a Select/Tooltip opened *inside* a Sheet on
// top of the sheet backdrop. Do not bump Sheet above z-50 (or other overlays
// below) without updating all siblings together. See BYT-9226 / PR #19824.
function SheetOverlay({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Backdrop>) {
  return (
    <BaseDialog.Backdrop
      ref={ref}
      className={cn(
        "fixed inset-0 z-50 bg-overlay/50",
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
    "fixed inset-y-0 right-0 z-50 flex h-full flex-col bg-background shadow-lg",
    "max-w-[100vw] outline-hidden",
    "data-[starting-style]:translate-x-full data-[ending-style]:translate-x-full",
    "transition-transform duration-200 ease-out"
  ),
  {
    variants: {
      width: {
        narrow: "w-[24rem]",
        standard: "w-[44rem]",
        wide: "w-[52rem]",
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
  return (
    <BaseDialog.Portal>
      <SheetOverlay />
      <BaseDialog.Popup
        ref={ref}
        className={cn(sheetContentVariants({ width }), className)}
        {...props}
      >
        {children}
        {/* Built-in close affordance in the top-right corner. Callers should
            not render their own close button — Base UI's Close component
            handles click and dismisses the Sheet via the Root's onOpenChange. */}
        <BaseDialog.Close
          aria-label="Close"
          className="absolute right-4 top-4 rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent cursor-pointer"
        >
          <X className="size-4" />
        </BaseDialog.Close>
      </BaseDialog.Popup>
    </BaseDialog.Portal>
  );
}

// ---- Header ----
// Sticky top region with a bottom border. Typically contains SheetTitle and
// an optional SheetDescription. `pr-12` reserves space on the right edge for
// the absolute-positioned close button in SheetContent.
function SheetHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex flex-col gap-y-1 border-b border-control-border px-6 py-4 pr-12",
        className
      )}
      {...props}
    />
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
