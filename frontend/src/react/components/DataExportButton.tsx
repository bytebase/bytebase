import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { ChevronDown, Download } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button, type ButtonProps } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { pushNotification } from "@/store";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { MaxRowCountSelect } from "./MaxRowCountSelect";

export type DownloadContent = {
  content: Uint8Array;
  filename: string;
};

export interface ExportOption {
  limit: number;
  format: ExportFormat;
  password: string;
}

export type DataExportRequest = {
  options: ExportOption;
  resolve: (content: DownloadContent[]) => void;
  reject: (reason?: unknown) => void;
};

type Size = "tiny" | "sm" | "md" | "lg";
type ViewMode = "DRAWER" | "DROPDOWN";

interface DataExportButtonProps {
  size?: Size;
  disabled?: boolean;
  supportFormats: ExportFormat[];
  supportPassword?: boolean;
  viewMode: ViewMode;
  tooltip?: string;
  text?: string;
  validate?: (option: ExportOption) => boolean;
  maximumExportCount?: number;
  /** Custom form content rendered above the row-count / format / password fields in DRAWER mode. */
  formContent?: ReactNode;
  onExport: (req: DataExportRequest) => void;
  className?: string;
}

const SIZE_TO_BUTTON: Record<Size, ButtonProps["size"]> = {
  tiny: "xs",
  sm: "sm",
  md: "default",
  lg: "lg",
};

const ROW_COUNT_PRESET = [1, 100, 500, 1000, 5000, 10000, 100000];

const computeMaximumPreset = (maximum: number) => {
  const list = ROW_COUNT_PRESET.filter((n) => n <= maximum);
  if (maximum !== Number.MAX_VALUE && !list.includes(maximum)) {
    list.push(maximum);
  }
  return list[list.length - 1] ?? 1000;
};

const getExportFileType = (format: ExportFormat) => {
  switch (format) {
    case ExportFormat.CSV:
      return "text/csv";
    case ExportFormat.JSON:
      return "application/json";
    case ExportFormat.SQL:
      return "application/sql";
    case ExportFormat.XLSX:
      return "application/vnd.ms-excel";
    default:
      return "application/octet-stream";
  }
};

const toBlob = (
  { content, filename }: DownloadContent,
  format: ExportFormat
) => {
  const isZip = filename.endsWith(".zip");
  const fileType = isZip ? "application/zip" : getExportFileType(format);
  const buffer = content.buffer.slice(
    content.byteOffset,
    content.byteOffset + content.byteLength
  ) as ArrayBuffer;
  return new Blob([buffer], { type: fileType });
};

const downloadSingle = (entry: DownloadContent, format: ExportFormat) => {
  const blob = toBlob(entry, format);
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.download = entry.filename;
  link.href = url;
  link.click();
  window.URL.revokeObjectURL(url);
};

const downloadZip = async (
  contents: DownloadContent[],
  format: ExportFormat
) => {
  const zip = new JSZip();
  for (const entry of contents) {
    zip.file(entry.filename, toBlob(entry, format));
  }
  const zipFile = await zip.generateAsync({ type: "blob" });
  saveAs(zipFile, `download_${dayjs().format("YYYY-MM-DDTHH-mm-ss")}.zip`);
};

const downloadAll = async (
  contents: DownloadContent[],
  format: ExportFormat
) => {
  if (contents.length === 0) return;
  if (contents.length === 1) {
    downloadSingle(contents[0], format);
    return;
  }
  await downloadZip(contents, format);
};

/**
 * React port of `frontend/src/components/DataExportButton.vue`.
 *
 * - DRAWER view: opens a Sheet with row-count / format / optional password
 *   fields, plus a `formContent` slot for caller-provided fields (e.g. the
 *   batch-export database picker).
 * - DROPDOWN view: a hover dropdown of formats; if `supportPassword` is set,
 *   selecting a format opens a small Dialog to capture an optional password.
 */
export function DataExportButton({
  size = "sm",
  disabled = false,
  supportFormats,
  supportPassword = false,
  viewMode,
  tooltip,
  text,
  validate,
  maximumExportCount = Number.MAX_VALUE,
  formContent,
  onExport,
  className,
}: DataExportButtonProps) {
  const { t } = useTranslation();

  const [isRequesting, setIsRequesting] = useState(false);
  const [showDrawer, setShowDrawer] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);

  const presetMax = useMemo(
    () => computeMaximumPreset(maximumExportCount),
    [maximumExportCount]
  );
  const defaultLimit = useMemo(() => Math.min(presetMax, 1000), [presetMax]);

  const [limit, setLimit] = useState<number>(defaultLimit);
  const [format, setFormat] = useState<ExportFormat>(supportFormats[0]);
  const [password, setPassword] = useState("");

  const resetForm = useCallback(() => {
    setLimit(defaultLimit);
    setFormat(supportFormats[0]);
    setPassword("");
  }, [defaultLimit, supportFormats]);

  // Reset form whenever the drawer opens, mirroring the Vue watcher.
  useEffect(() => {
    if (showDrawer) {
      resetForm();
    }
  }, [showDrawer, resetForm]);

  // Clear password each time the password dialog opens.
  useEffect(() => {
    if (showPasswordDialog) {
      setPassword("");
    }
  }, [showPasswordDialog]);

  const buttonText = text ?? t("common.export");

  const currentOption: ExportOption = useMemo(
    () => ({ limit, format, password }),
    [limit, format, password]
  );

  const validateOption = validate ?? (() => true);
  const formIsValid = limit > 0 && validateOption(currentOption);

  const doExport = useCallback(() => {
    if (isRequesting) return;
    setIsRequesting(true);
    const options: ExportOption = { ...currentOption };

    new Promise<DownloadContent[]>((resolve, reject) => {
      onExport({ options, resolve, reject });
    })
      .then((contents) => downloadAll(contents, options.format))
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.succeed"),
          description: t("audit-log.export-finished"),
        });
      })
      .catch((error: unknown) => {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Failed to export data",
          description: JSON.stringify(error),
        });
      })
      .finally(() => {
        setIsRequesting(false);
        setShowDrawer(false);
        setShowPasswordDialog(false);
      });
  }, [currentOption, isRequesting, onExport, t]);

  const handleSelectDropdownFormat = (selected: ExportFormat) => {
    setFormat(selected);
    if (supportPassword) {
      setShowPasswordDialog(true);
    } else {
      // Run after state batched.
      queueMicrotask(doExport);
    }
  };

  const triggerButton = (
    <Button
      variant="default"
      size={SIZE_TO_BUTTON[size]}
      disabled={disabled || isRequesting}
      onClick={(e) => {
        e.preventDefault();
        if (viewMode === "DRAWER") setShowDrawer(true);
      }}
      className={cn("gap-x-1", className)}
      aria-label={buttonText}
    >
      <Download className="size-4" />
      {size !== "tiny" && <span>{buttonText}</span>}
    </Button>
  );

  const dropdownTrigger = (
    <DropdownMenu>
      <Tooltip content={tooltip} side="bottom">
        <DropdownMenuTrigger
          render={
            <Button
              variant="default"
              size={SIZE_TO_BUTTON[size]}
              disabled={disabled || isRequesting}
              className={cn("gap-x-1", className)}
              aria-label={buttonText}
            >
              <Download className="size-4" />
              {size !== "tiny" && <span>{buttonText}</span>}
              <ChevronDown className="size-3" />
            </Button>
          }
        />
      </Tooltip>
      <DropdownMenuContent align="end">
        {supportFormats.map((fmt) => (
          <DropdownMenuItem
            key={fmt}
            onClick={() => handleSelectDropdownFormat(fmt)}
          >
            {t("sql-editor.download-as-file", {
              file: ExportFormat[fmt],
            })}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );

  return (
    <>
      {viewMode === "DROPDOWN" ? (
        dropdownTrigger
      ) : (
        <Tooltip content={tooltip} side="bottom">
          {triggerButton}
        </Tooltip>
      )}

      {viewMode === "DRAWER" && (
        <Sheet open={showDrawer} onOpenChange={setShowDrawer}>
          <SheetContent width="medium">
            <SheetHeader>
              <SheetTitle>
                {t("custom-approval.risk-rule.risk.namespace.data_export")}
              </SheetTitle>
            </SheetHeader>
            <SheetBody>
              <div className="flex flex-col gap-y-4">
                {formContent}

                <div className="flex flex-col gap-y-1">
                  <label className="text-sm text-control">
                    {t("export-data.export-rows")}
                  </label>
                  <MaxRowCountSelect
                    value={limit}
                    onChange={setLimit}
                    maximum={presetMax}
                  />
                </div>

                <div className="flex flex-col gap-y-1">
                  <label className="text-sm text-control">
                    {t("export-data.export-format")}
                  </label>
                  <RadioGroup
                    value={String(format)}
                    onValueChange={(v) => setFormat(Number(v) as ExportFormat)}
                  >
                    {supportFormats.map((fmt) => (
                      <RadioGroupItem key={fmt} value={String(fmt)}>
                        {ExportFormat[fmt]}
                      </RadioGroupItem>
                    ))}
                  </RadioGroup>
                </div>

                {supportPassword && (
                  <div className="flex flex-col gap-y-1">
                    <label className="text-sm text-control">
                      {t("export-data.password-optional")}
                    </label>
                    <Input
                      type="password"
                      autoComplete="new-password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                    />
                  </div>
                )}
              </div>
            </SheetBody>
            <SheetFooter>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowDrawer(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button
                variant="default"
                size="sm"
                disabled={!formIsValid || isRequesting}
                onClick={doExport}
              >
                {t("common.confirm")}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      )}

      {viewMode === "DROPDOWN" && supportPassword && (
        <Dialog open={showPasswordDialog} onOpenChange={setShowPasswordDialog}>
          <DialogContent className="w-[26rem] max-w-[calc(100vw-4rem)]">
            <div className="flex flex-col gap-y-3 p-6">
              <DialogTitle>{t("export-data.password-optional")}</DialogTitle>
              <p className="text-sm text-control-light">
                {t("export-data.password-info")}
              </p>
              <Input
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoFocus
              />
              <div className="flex items-center justify-end gap-x-2 pt-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowPasswordDialog(false)}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  variant="default"
                  size="sm"
                  disabled={isRequesting}
                  onClick={doExport}
                >
                  {t("common.export")}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
