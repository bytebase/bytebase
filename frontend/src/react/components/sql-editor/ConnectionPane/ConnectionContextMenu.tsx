import { forwardRef, useImperativeHandle, useRef, useState } from "react";
import { flushSync } from "react-dom";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import type { SQLEditorTreeNode } from "@/types";
import { useConnectionMenu } from "./actions";

export type ConnectionContextMenuHandle = {
  show: (node: SQLEditorTreeNode, e: React.MouseEvent) => void;
  hide: () => void;
};

type Target = {
  x: number;
  y: number;
  node: SQLEditorTreeNode;
};

/**
 * Replaces the `NDropdown placement="bottom-start" trigger="manual"` in
 * frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/ConnectionPane.vue.
 * Mirrors the TabContextMenu pattern: a 0×0 position-fixed trigger at the
 * cursor is programmatically .click()-ed so Base UI records a click-type
 * open event and the popup stays open when the pointer moves onto items.
 */
export const ConnectionContextMenu = forwardRef<ConnectionContextMenuHandle>(
  function ConnectionContextMenu(_, ref) {
    const [target, setTarget] = useState<Target | null>(null);
    const triggerRef = useRef<HTMLButtonElement>(null);

    const { items, handleSelect } = useConnectionMenu(target?.node ?? null);

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
          setTarget(null);
        },
      }),
      []
    );

    if (items.length === 0) {
      // Nothing to show for this node (disabled, or an unsupported type).
      return null;
    }

    return (
      <DropdownMenu
        onOpenChange={(open) => {
          if (!open) setTarget(null);
        }}
      >
        <DropdownMenuTrigger
          ref={triggerRef}
          aria-hidden
          tabIndex={-1}
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
        <DropdownMenuContent
          align="start"
          sideOffset={4}
          positionMethod="fixed"
        >
          {items.map((item) => (
            <DropdownMenuItem
              key={item.key}
              onClick={() => {
                handleSelect(item.key);
                setTarget(null);
              }}
            >
              <span className="inline-flex items-center gap-x-2">
                {item.icon}
                {item.label}
              </span>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
);
