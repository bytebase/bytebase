import { ChevronDown, ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { LAYER_ROOT_ID } from "@/react/components/ui/layer";
import { Popover, PopoverContent } from "@/react/components/ui/popover";
import type { TreeDataNode } from "@/react/components/ui/tree";
import { Tree } from "@/react/components/ui/tree";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { TreeNodePrefix } from "./TreeNodePrefix";

type Props = {
  readonly folder: string;
  readonly onFolderChange: (folder: string) => void;
};

function toTreeData(
  node: WorksheetFolderNode
): TreeDataNode<WorksheetFolderNode> {
  return {
    id: node.key,
    data: node,
    children: node.children.map(toTreeData),
  };
}

export function FolderForm({ folder, onFolderChange }: Props) {
  const { t } = useTranslation();

  const viewContext = useSheetContextByView("my");

  const folderTree = useVueState(() => viewContext.folderTree.value);
  const rootPath = useVueState(() => viewContext.folderContext.rootPath.value);

  const [folderPath, setFolderPath] = useState(folder);
  const [showPopover, setShowPopover] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Close the popover on a mousedown outside the input AND outside the
  // overlay layer (where the portalled popup lives). Base UI's PopoverTrigger
  // toggles on click, which means a second click on the input would close the
  // popover even though the user wants to keep typing — so we bypass Trigger
  // entirely and manage open state ourselves.
  useEffect(() => {
    if (!showPopover) return;
    const overlayRoot = document.getElementById(LAYER_ROOT_ID.overlay);
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (containerRef.current?.contains(target)) return;
      if (overlayRoot?.contains(target)) return;
      setShowPopover(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showPopover]);

  // Sync incoming folder prop → local state
  useEffect(() => {
    setFolderPath(folder);
  }, [folder]);

  // Emit changes upward whenever folderPath changes
  useEffect(() => {
    onFolderChange(folderPath);
  }, [folderPath, onFolderChange]);

  const formattedFolderPath = (() => {
    let val = folderPath.replace(rootPath, "");
    if (val[0] === "/") {
      val = val.slice(1);
    }
    return val.split("/").join(" / ");
  })();

  const handleInput = (raw: string) => {
    let changedVal = raw;
    if (changedVal.endsWith(" /")) {
      changedVal = changedVal.slice(0, -2);
    }
    if (changedVal.endsWith(".")) {
      changedVal = changedVal.slice(0, -1);
    }
    const rawPath = changedVal
      .split("/")
      .map((p) => p.trim())
      .join("/");
    setFolderPath(rawPath);
  };

  // User explicitly clicked a row. Set the folder and close the popover.
  const handleUserClick = (id: string) => {
    setFolderPath(id);
    queueMicrotask(() => setShowPopover(false));
  };

  const treeData = folderTree.children.map(toTreeData);

  const searchMatch = (node: TreeDataNode<WorksheetFolderNode>) => {
    if (!folderPath) return true;
    return node.data.key.includes(folderPath);
  };

  return (
    <div className="flex flex-col gap-y-2">
      <div>
        <p>{t("sql-editor.choose-folder")}</p>
        <span className="textinfolabel">
          {t("sql-editor.choose-folder-tips")}
        </span>
      </div>
      <div ref={containerRef}>
        <Input
          ref={inputRef}
          data-testid="folder-input"
          value={formattedFolderPath}
          placeholder={t("sql-editor.choose-folder")}
          onFocus={() => setShowPopover(true)}
          onClick={() => setShowPopover(true)}
          onChange={(e) => handleInput(e.target.value)}
        />
        <Popover open={showPopover} onOpenChange={setShowPopover}>
          <PopoverContent
            side="bottom"
            align="start"
            anchor={inputRef}
            initialFocus={false}
            finalFocus={false}
            style={{ width: "var(--anchor-width)" }}
            className="p-1"
          >
            <Tree<WorksheetFolderNode>
              data={treeData}
              selectedIds={folderPath ? [folderPath] : []}
              searchTerm={folderPath || undefined}
              searchMatch={searchMatch}
              height={240}
              renderNode={({ node, style }) => {
                const data = node.data.data;
                const hasChildren = data.children.length > 0;
                return (
                  <div
                    key={node.id}
                    style={style}
                    className={cn(
                      "flex items-center gap-x-1 px-1.5 py-1 cursor-pointer select-none rounded-xs text-sm text-control",
                      node.isSelected ? "bg-accent/10" : "hover:bg-accent/5"
                    )}
                    onClick={() => handleUserClick(node.id)}
                  >
                    {hasChildren ? (
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          node.toggle();
                        }}
                        className="flex items-center justify-center size-4 shrink-0 text-control hover:text-main"
                      >
                        {node.isOpen ? (
                          <ChevronDown className="size-3" />
                        ) : (
                          <ChevronRight className="size-3" />
                        )}
                      </button>
                    ) : (
                      <span className="size-4 shrink-0" />
                    )}
                    <TreeNodePrefix
                      node={data}
                      isOpen={node.isOpen}
                      rootPath={rootPath}
                      view="my"
                    />
                    <span className="truncate">{data.label}</span>
                  </div>
                );
              }}
            />
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}
