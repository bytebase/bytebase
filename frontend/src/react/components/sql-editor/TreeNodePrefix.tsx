import {
  FileCode,
  FolderCode,
  FolderMinus,
  FolderOpen,
  FolderPen,
  FolderSync,
} from "lucide-react";
import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";
import type { SheetViewMode } from "@/views/sql-editor/Sheet/types";

type Props = {
  readonly node: WorksheetFolderNode;
  readonly isOpen: boolean;
  readonly rootPath: string;
  readonly view: SheetViewMode;
};

export function TreeNodePrefix({ node, isOpen, rootPath, view }: Props) {
  const cls = "size-4 text-control shrink-0";

  if (node.worksheet) {
    return <FileCode className={cls} />;
  }
  if (isOpen) {
    return <FolderOpen className={cls} />;
  }
  if (node.key === rootPath) {
    if (view === "draft") {
      return <FolderPen className={cls} />;
    }
    if (view === "shared") {
      return <FolderSync className={cls} />;
    }
    return <FolderCode className={cls} />;
  }
  if (node.empty) {
    return <FolderMinus className={cls} />;
  }
  return <FolderCode className={cls} />;
}
