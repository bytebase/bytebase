import type { MouseEvent } from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { cn } from "@/react/lib/utils";
import {
  type Release_File,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";

export interface ReleaseFileTableProps {
  files: Release_File[];
  releaseType: Release_Type;
  showSelection?: boolean;
  rowClickable?: boolean;
  selectedFiles?: Release_File[];
  onRowClick?: (file: Release_File, e: MouseEvent) => void;
  onSelectedFilesChange?: (files: Release_File[]) => void;
}

export function ReleaseFileTable({
  files,
  releaseType,
  showSelection = false,
  rowClickable = true,
  selectedFiles,
  onRowClick,
  onSelectedFilesChange,
}: ReleaseFileTableProps) {
  const { t } = useTranslation();

  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(
    () => new Set((selectedFiles ?? []).map((f) => f.path))
  );

  useEffect(() => {
    setSelectedPaths(new Set((selectedFiles ?? []).map((f) => f.path)));
  }, [selectedFiles]);

  const typeText = useMemo(() => {
    switch (releaseType) {
      case Release_Type.DECLARATIVE:
        return "SDL";
      case Release_Type.VERSIONED:
        return t("issue.title.change-database");
      case Release_Type.TYPE_UNSPECIFIED:
        return "";
      default:
        releaseType satisfies never;
        return "";
    }
  }, [releaseType, t]);

  const applySelection = (nextPaths: Set<string>) => {
    setSelectedPaths(nextPaths);
    onSelectedFilesChange?.(
      [...nextPaths]
        .map((path) => files.find((f) => f.path === path))
        .filter((f): f is Release_File => !!f)
    );
  };

  const handleRowClick = (file: Release_File, e: MouseEvent) => {
    if (showSelection) {
      const next = new Set(selectedPaths);
      if (next.has(file.path)) {
        next.delete(file.path);
      } else {
        next.add(file.path);
      }
      applySelection(next);
    } else if (rowClickable) {
      onRowClick?.(file, e);
    }
  };

  const handleCheckboxChange = (file: Release_File, checked: boolean) => {
    const next = new Set(selectedPaths);
    if (checked) {
      next.add(file.path);
    } else {
      next.delete(file.path);
    }
    applySelection(next);
  };

  const rowCursor = rowClickable || showSelection ? "cursor-pointer" : "";
  // Show the "Detail" affordance only when the row itself opens a detail
  // panel — in selection mode the row click toggles the checkbox instead.
  const showDetailButton = !showSelection && rowClickable && !!onRowClick;

  return (
    <div className="w-full border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {showSelection && <TableHead className="w-10" />}
            <TableHead className="w-40">{t("common.version")}</TableHead>
            <TableHead className="w-16">{t("common.type")}</TableHead>
            <TableHead className="w-32">
              {t("database.revision.filename")}
            </TableHead>
            {showDetailButton && <TableHead className="w-24" />}
          </TableRow>
        </TableHeader>
        <TableBody>
          {files.map((file) => {
            const checked = selectedPaths.has(file.path);
            return (
              <TableRow
                key={file.path}
                className={cn(rowCursor)}
                onClick={(e) => handleRowClick(file, e)}
              >
                {showSelection && (
                  <TableCell
                    className="w-10"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(checked) =>
                        handleCheckboxChange(file, checked)
                      }
                    />
                  </TableCell>
                )}
                <TableCell className="truncate">{file.version}</TableCell>
                <TableCell>{typeText}</TableCell>
                <TableCell className="truncate">{file.path || "-"}</TableCell>
                {showDetailButton && (
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onRowClick?.(file, e);
                      }}
                    >
                      {t("common.detail")}
                    </Button>
                  </TableCell>
                )}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
