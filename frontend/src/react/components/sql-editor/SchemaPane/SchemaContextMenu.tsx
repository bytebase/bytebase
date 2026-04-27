import { Menu as BaseMenu } from "@base-ui/react/menu";
import { forwardRef, useImperativeHandle, useRef, useState } from "react";
import { flushSync } from "react-dom";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSubmenu,
  DropdownMenuSubmenuContent,
  DropdownMenuSubmenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  type SchemaMenuDeps,
  type SchemaMenuItem,
  useSchemaPaneContextMenu,
} from "./actions";
import type { TreeNode } from "./schemaTree";

export type SchemaContextMenuHandle = {
  show: (node: TreeNode, e: React.MouseEvent) => void;
  hide: () => void;
};

type Target = {
  x: number;
  y: number;
  node: TreeNode;
};

type Props = SchemaMenuDeps;

/**
 * Replaces `SchemaPane/actions.tsx`'s NDropdown imperative use in
 * `SchemaPane.vue`. Mirrors the Stage 14 `ConnectionContextMenu` pattern:
 * a 0×0 fixed-position trigger at the cursor is programmatically clicked
 * so Base UI records a click-type open event and the popup stays open
 * while the pointer moves to a menu item.
 *
 * `open` is controlled so `hide()` can dismiss the menu deterministically
 * (matches the same fix applied to TabContextMenu after the codex review).
 */
export const SchemaContextMenu = forwardRef<SchemaContextMenuHandle, Props>(
  function SchemaContextMenu(deps, ref) {
    const [target, setTarget] = useState<Target | null>(null);
    const [open, setOpen] = useState(false);
    const triggerRef = useRef<HTMLButtonElement>(null);

    const items = useSchemaPaneContextMenu(target?.node ?? null, deps);

    useImperativeHandle(
      ref,
      () => ({
        show(node, e) {
          e.preventDefault();
          e.stopPropagation();
          flushSync(() => {
            setTarget({ x: e.clientX, y: e.clientY, node });
          });
          triggerRef.current?.click();
        },
        hide() {
          setOpen(false);
          setTarget(null);
        },
      }),
      []
    );

    if (items.length === 0) return null;

    return (
      <DropdownMenu
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) setTarget(null);
        }}
      >
        <BaseMenu.Trigger
          ref={triggerRef}
          aria-hidden
          tabIndex={-1}
          render={
            <button
              type="button"
              style={{
                position: "fixed",
                top: target?.y ?? 0,
                left: target?.x ?? 0,
                width: 0,
                height: 0,
                pointerEvents: "none",
                opacity: 0,
              }}
            />
          }
        />
        <DropdownMenuContent
          align="start"
          sideOffset={4}
          positionMethod="fixed"
        >
          {items.map((item) => renderItem(item, () => setOpen(false)))}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
);

function renderItem(item: SchemaMenuItem, close: () => void) {
  if (item.children && item.children.length > 0) {
    return (
      <DropdownMenuSubmenu key={item.key}>
        <DropdownMenuSubmenuTrigger>
          <span className="inline-flex items-center gap-x-2">
            {item.icon}
            {item.label}
          </span>
        </DropdownMenuSubmenuTrigger>
        <DropdownMenuSubmenuContent>
          {item.children.map((child) => renderItem(child, close))}
        </DropdownMenuSubmenuContent>
      </DropdownMenuSubmenu>
    );
  }
  return (
    <DropdownMenuItem
      key={item.key}
      onClick={() => {
        item.onSelect?.();
        close();
      }}
    >
      <span className="inline-flex items-center gap-x-2">
        {item.icon}
        {item.label}
      </span>
    </DropdownMenuItem>
  );
}
