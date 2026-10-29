import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { ChevronDown, Copy, Download, Wand2 } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button, type ButtonProps } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  FormControlGroup,
  FormControlRow,
  FormField,
  FormFieldGroup,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { ExportDestination, ExportFormat } from "@/types/proto-es/v1/common_pb";
import { MaxRowCountSelect } from "./MaxRowCountSelect";
import {
  formatSharePointUrl,
  type SharePointDeliveryResult,
  SharePointDeliveryResultDialog,
} from "./SharePointDeliveryResultDialog";

export type DownloadContent = {
  content: Uint8Array;
  filename: string;
};

export interface ExportOption {
  limit: number;
  format: ExportFormat;
  // Applies only to DOWNLOAD.
  password: string;
  destination: ExportDestination;
}

// Downloads resolve with the files to save; SharePoint exports resolve with
// each database's delivery result.
export type ExportOutcome =
  | DownloadContent[]
  | { sharepoint: SharePointDeliveryResult[] };

export type DataExportRequest = {
  options: ExportOption;
  resolve: (outcome: ExportOutcome) => void;
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
  tooltip?: ReactNode;
  text?: string;
  validate?: (option: ExportOption) => boolean;
  maximumExportCount?: number;
  /** Custom form content rendered above the row-count / format / password fields in DRAWER mode. */
  formContent?: ReactNode;
  /** Offered destinations in DRAWER mode; the first is the default. Empty disables export. */
  destinations?: ExportDestination[];
  /** The project's SharePoint destination URL, shown beside that destination. */
  sharepointUrl?: string;
  onExport: (req: DataExportRequest) => void;
  className?: string;
}

const DOWNLOAD_ONLY = [ExportDestination.DOWNLOAD];

const SIZE_TO_BUTTON: Record<Size, ButtonProps["size"]> = {
  tiny: "xs",
  sm: "sm",
  md: "default",
  lg: "lg",
};

const ROW_COUNT_PRESET = [1, 100, 500, 1000, 5000, 10000, 100000];

const GENERATED_PASSWORD_LENGTH = 8;
const GENERATED_PASSWORD_CHARSET =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

const generateRandomPassword = (length = GENERATED_PASSWORD_LENGTH) => {
  const values = new Uint32Array(length);
  crypto.getRandomValues(values);
  let out = "";
  for (let i = 0; i < length; i++) {
    out +=
      GENERATED_PASSWORD_CHARSET[values[i] % GENERATED_PASSWORD_CHARSET.length];
  }
  return out;
};

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
  destinations = DOWNLOAD_ONLY,
  sharepointUrl,
  onExport,
  className,
}: DataExportButtonProps) {
  const { t } = useTranslation();

  const [isRequesting, setIsRequesting] = useState(false);
  const [showDrawer, setShowDrawer] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);
  const [deliveryResults, setDeliveryResults] = useState<
    SharePointDeliveryResult[]
  >([]);
  const exportUnavailable = destinations.length === 0;

  const presetMax = useMemo(
    () => computeMaximumPreset(maximumExportCount),
    [maximumExportCount]
  );
  const defaultLimit = useMemo(() => Math.min(presetMax, 1000), [presetMax]);

  const [limit, setLimit] = useState<number>(defaultLimit);
  const [format, setFormat] = useState<ExportFormat>(supportFormats[0]);
  const [password, setPassword] = useState("");
  const [destination, setDestination] = useState<ExportDestination>(
    destinations[0] ?? ExportDestination.DOWNLOAD
  );
  const isDownload = destination !== ExportDestination.SHAREPOINT;

  const resetForm = useCallback(() => {
    setLimit(defaultLimit);
    setFormat(supportFormats[0]);
    setPassword("");
    setDestination(destinations[0] ?? ExportDestination.DOWNLOAD);
  }, [defaultLimit, supportFormats, destinations]);

  // Reset form whenever the drawer transitions to OPEN. Depending on
  // `resetForm` here would re-fire the reset on every parent re-render
  // because `supportFormats` typically arrives as a fresh inline array
  // (e.g. `supportFormats={[CSV, JSON, SQL, XLSX]}` from
  // `SingleResultView`), which churns `resetForm`'s identity and would
  // clobber the user's just-changed `limit` / `format` / `password`.
  // Drive the reset off `showDrawer` alone via a ref to the latest
  // `resetForm`.
  const resetFormRef = useRef(resetForm);
  resetFormRef.current = resetForm;
  useEffect(() => {
    if (showDrawer) {
      resetFormRef.current();
    }
  }, [showDrawer]);

  // Clear password each time the password dialog opens.
  useEffect(() => {
    if (showPasswordDialog) {
      setPassword("");
    }
  }, [showPasswordDialog]);

  const buttonText = text ?? t("common.export");

  const currentOption: ExportOption = useMemo(
    () => ({
      limit,
      format,
      password: isDownload ? password : "",
      destination,
    }),
    [limit, format, password, destination, isDownload]
  );

  const validateOption = validate ?? (() => true);
  const formIsValid =
    limit > 0 &&
    destinations.includes(destination) &&
    validateOption(currentOption);

  const doExport = useCallback(() => {
    if (isRequesting) return;
    setIsRequesting(true);
    const options: ExportOption = { ...currentOption };

    new Promise<ExportOutcome>((resolve, reject) => {
      onExport({ options, resolve, reject });
    })
      .then(async (outcome) => {
        if (!Array.isArray(outcome)) {
          setDeliveryResults(outcome.sharepoint);
          return;
        }
        await downloadAll(outcome, options.format);
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

  // Returns true when the clipboard write succeeded — Generate callers
  // gate the password commit on that so the masked input doesn't end up
  // holding a value the user can never read back. PR #20491 bot review
  // #3349557577.
  const copyPasswordToClipboard = useCallback(
    async (value: string): Promise<boolean> => {
      if (!value) return false;
      try {
        if (!(await writeTextToClipboard(value))) {
          throw new Error("clipboard unavailable");
        }
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
        return true;
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
        return false;
      }
    },
    [t]
  );

  const handleGeneratePassword = useCallback(async () => {
    const next = generateRandomPassword();
    // Only commit the generated password to state after the user has it
    // on their clipboard. If the copy fails (insecure context, denied
    // permission) the input stays empty so the user isn't left signing
    // an export ZIP with a password they can't read back from the masked
    // input.
    const copied = await copyPasswordToClipboard(next);
    if (copied) {
      setPassword(next);
    }
  }, [copyPasswordToClipboard]);

  const handleSelectDropdownFormat = (selected: ExportFormat) => {
    setFormat(selected);
    if (supportPassword && isDownload) {
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
      disabled={disabled || isRequesting || exportUnavailable}
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
              disabled={disabled || isRequesting || exportUnavailable}
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
              <FormFieldGroup>
                {formContent}

                {destinations.length > 1 ? (
                  <FormField title={t("export-data.destination.self")}>
                    <RadioGroup
                      aria-label={t("export-data.destination.self")}
                      className="flex-col items-start gap-y-2"
                      value={String(destination)}
                      onValueChange={(value) =>
                        setDestination(Number(value) as ExportDestination)
                      }
                    >
                      {destinations.map((option) => (
                        <RadioGroupItem key={option} value={String(option)}>
                          <DestinationLabel
                            destination={option}
                            sharepointUrl={sharepointUrl}
                          />
                        </RadioGroupItem>
                      ))}
                    </RadioGroup>
                  </FormField>
                ) : (
                  destination === ExportDestination.SHAREPOINT && (
                    <FormField title={t("export-data.destination.self")}>
                      <DestinationLabel
                        destination={destination}
                        sharepointUrl={sharepointUrl}
                      />
                    </FormField>
                  )
                )}

                <FormField title={t("export-data.export-rows")}>
                  <MaxRowCountSelect
                    value={limit}
                    onChange={setLimit}
                    maximum={presetMax}
                    className="h-9"
                  />
                </FormField>

                <FormField title={t("export-data.export-format")}>
                  <Select
                    value={String(format)}
                    onValueChange={(value) => {
                      if (value) setFormat(Number(value) as ExportFormat);
                    }}
                  >
                    <SelectTrigger
                      className="w-full"
                      aria-label={t("export-data.export-format")}
                    >
                      <SelectValue>{ExportFormat[format]}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {supportFormats.map((fmt) => (
                        <SelectItem key={fmt} value={String(fmt)}>
                          {ExportFormat[fmt]}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>

                {supportPassword && isDownload && (
                  <FormField
                    title={
                      <FormLabel htmlFor="data-export-password">
                        {t("export-data.password-optional")}
                      </FormLabel>
                    }
                  >
                    <Input
                      id="data-export-password"
                      type="password"
                      autoComplete="new-password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                    />
                    <FormControlGroup>
                      <FormControlRow>
                        <Button
                          type="button"
                          appearance="outline"
                          size="sm"
                          onClick={handleGeneratePassword}
                        >
                          <Wand2 className="size-4" />
                          {t("export-data.generate-password")}
                        </Button>
                        <Button
                          type="button"
                          appearance="outline"
                          size="sm"
                          disabled={!password}
                          onClick={() => void copyPasswordToClipboard(password)}
                        >
                          <Copy className="size-4" />
                          {t("common.copy")}
                        </Button>
                      </FormControlRow>
                    </FormControlGroup>
                  </FormField>
                )}
              </FormFieldGroup>
            </SheetBody>
            <SheetFooter>
              <Button appearance="outline" onClick={() => setShowDrawer(false)}>
                {t("common.cancel")}
              </Button>
              <Button
                variant="default"
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
            <div className="flex flex-col gap-y-3">
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
                  appearance="outline"
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

      <SharePointDeliveryResultDialog
        results={deliveryResults}
        onClose={() => setDeliveryResults([])}
      />
    </>
  );
}

function DestinationLabel({
  destination,
  sharepointUrl,
}: {
  destination: ExportDestination;
  sharepointUrl?: string;
}) {
  const { t } = useTranslation();
  if (destination !== ExportDestination.SHAREPOINT) {
    return <span>{t("export-data.destination.download")}</span>;
  }
  return (
    <span className="flex flex-col">
      <span>{t("export-data.destination.sharepoint")}</span>
      {sharepointUrl && (
        <span className="break-all text-xs text-control-light">
          {formatSharePointUrl(sharepointUrl)}
        </span>
      )}
    </span>
  );
}
