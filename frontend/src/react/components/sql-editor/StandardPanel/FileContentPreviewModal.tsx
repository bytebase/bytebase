import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { pushNotification } from "@/store";
import { ENCODINGS, type Encoding, readFileAsArrayBuffer } from "@/utils";

interface FileContentPreviewModalProps {
  file: File;
  open: boolean;
  onCancel: () => void;
  onConfirm: (text: string) => void;
}

/**
 * React port of `frontend/src/components/FileContentPreviewModal.vue`.
 * Decodes the file as bytes, exposes an encoding picker, renders the
 * decoded text in a read-only Monaco preview, and emits the chosen
 * statement back to the caller on confirm.
 */
export function FileContentPreviewModal({
  file,
  open,
  onCancel,
  onConfirm,
}: FileContentPreviewModalProps) {
  const { t } = useTranslation();
  const [encoding, setEncoding] = useState<Encoding>("utf-8");
  const [decodedText, setDecodedText] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setIsLoading(true);
    void readFileAsArrayBuffer(file)
      .then(({ arrayBuffer }) => {
        if (cancelled) return;
        const text = new TextDecoder(encoding).decode(arrayBuffer);
        setDecodedText(text);
      })
      .catch((error) => {
        if (cancelled) return;
        console.error(error);
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Failed to read file",
        });
      })
      .finally(() => {
        if (cancelled) return;
        setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [file, encoding]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onCancel()}>
      <DialogContent className="w-[36rem] max-w-[36rem]">
        <DialogTitle>{t("common.preview")}</DialogTitle>
        <div className="py-1 flex flex-col justify-start items-start gap-2">
          <div className="w-full flex flex-row justify-between items-center gap-4">
            <p className="font-medium textlabel whitespace-nowrap">
              {t("sql-editor.select-encoding")}
            </p>
            <Select
              value={encoding}
              onValueChange={(value) => {
                if (typeof value === "string") setEncoding(value as Encoding);
              }}
            >
              <SelectTrigger size="sm" className="w-auto">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ENCODINGS.map((option) => (
                  <SelectItem key={option} value={option}>
                    {option}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="w-full overflow-hidden relative h-80 shrink-0 border border-control-border rounded-xs">
            <ReadonlyMonaco content={decodedText} className="w-full h-full" />
            {isLoading ? (
              <div className="absolute inset-0 bg-background/60 flex items-center justify-center">
                <Loader2 className="size-6 animate-spin text-control-light" />
              </div>
            ) : null}
          </div>
          <div className="w-full flex justify-end gap-x-2">
            <Button variant="outline" onClick={onCancel}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="default"
              disabled={isLoading}
              onClick={() => onConfirm(decodedText)}
            >
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
