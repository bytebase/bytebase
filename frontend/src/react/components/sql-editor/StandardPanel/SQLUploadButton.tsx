import type { ChangeEvent, ReactNode } from "react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { pushNotification } from "@/store";
import { MAX_UPLOAD_FILE_SIZE_MB } from "@/utils";
import { FileContentPreviewModal } from "./FileContentPreviewModal";

interface SQLUploadButtonProps {
  iconOnly?: boolean;
  className?: string;
  children?: ReactNode;
  onUpdateSql: (text: string, filename: string) => void;
}

/**
 * React port of `frontend/src/components/misc/SQLUploadButton.vue`.
 *
 * Hidden `<input type="file">` triggered by clicking the button. After
 * the user picks a file, mounts a preview modal that decodes + shows
 * the content; the parent receives the final text via `onUpdateSql`.
 */
export function SQLUploadButton({
  iconOnly,
  className,
  children,
  onUpdateSql,
}: SQLUploadButtonProps) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const cleanup = useCallback(() => {
    setSelectedFile(null);
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  }, []);

  const handleClick = useCallback(() => {
    inputRef.current?.click();
  }, []);

  const handleUpload = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      const cleanupInput = () => {
        // Selecting the same file twice doesn't fire `change` again,
        // so reset the value here too.
        if (inputRef.current) {
          inputRef.current.value = "";
        }
      };
      if (!file) {
        cleanupInput();
        return;
      }
      if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("issue.upload-sql-file-max-size-exceeded", {
            size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
          }),
        });
        cleanupInput();
        return;
      }
      setSelectedFile(file);
      cleanupInput();
    },
    [t]
  );

  const handleConfirm = useCallback(
    (statement: string) => {
      if (selectedFile) {
        onUpdateSql(statement, selectedFile.name);
      }
      cleanup();
    },
    [onUpdateSql, selectedFile, cleanup]
  );

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className={cn("h-7 px-1", className)}
        onClick={handleClick}
      >
        {children}
        {iconOnly ? null : (
          <span className="ml-1">{t("sql-editor.upload-file")}</span>
        )}
      </Button>
      <input
        ref={inputRef}
        type="file"
        accept=".sql,.txt,application/sql,text/plain"
        className="hidden"
        onChange={handleUpload}
      />
      {selectedFile ? (
        <FileContentPreviewModal
          file={selectedFile}
          open
          onCancel={cleanup}
          onConfirm={handleConfirm}
        />
      ) : null}
    </>
  );
}
