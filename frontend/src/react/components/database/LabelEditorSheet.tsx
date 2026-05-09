import { Plus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function LabelEditorSheet({
  open,
  databases,
  onClose,
  onApply,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onApply: (labelsList: { [key: string]: string }[]) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [labelsList, setLabelsList] = useState<{ [key: string]: string }[]>([]);
  const [applying, setApplying] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  useEffect(() => {
    if (open) {
      setLabelsList(databases.map((db) => ({ ...db.labels })));
      setApplying(false);
      setNewKey("");
      setNewValue("");
    }
  }, [open, databases]);

  const addLabelToAll = () => {
    if (!newKey.trim()) return;
    setLabelsList((prev) =>
      prev.map((labels) => ({ ...labels, [newKey.trim()]: newValue.trim() }))
    );
    setNewKey("");
    setNewValue("");
  };

  const removeLabel = (key: string) => {
    setLabelsList((prev) =>
      prev.map((labels) => {
        const next = { ...labels };
        delete next[key];
        return next;
      })
    );
  };

  const allKeys = Array.from(
    new Set(labelsList.flatMap((labels) => Object.keys(labels)))
  ).sort();

  const getDisplayValue = (key: string): { value: string; mixed: boolean } => {
    const values = new Set(
      labelsList.map((labels) => labels[key] ?? "").filter(Boolean)
    );
    if (values.size === 0) return { value: "", mixed: false };
    if (values.size === 1) return { value: [...values][0], mixed: false };
    return { value: t("database.mixed-label-values"), mixed: true };
  };

  const hasMixedValues = allKeys.some((key) => getDisplayValue(key).mixed);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("database.edit-labels")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          {hasMixedValues && (
            <Alert
              variant="warning"
              className="mb-4"
              description={t("database.mixed-label-values-warning")}
            />
          )}
          <div className="flex items-center gap-x-2 mb-4">
            <Input
              placeholder={t("common.key")}
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              className="flex-1"
            />
            <span className="text-control-placeholder">:</span>
            <Input
              placeholder={t("common.value")}
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              className="flex-1"
              onKeyDown={(e) => {
                if (e.key === "Enter") addLabelToAll();
              }}
            />
            <Button size="sm" onClick={addLabelToAll} disabled={!newKey.trim()}>
              <Plus className="size-4" />
            </Button>
          </div>
          {allKeys.length > 0 ? (
            <div className="flex flex-col gap-y-2">
              {allKeys.map((key) => {
                const { value, mixed } = getDisplayValue(key);
                return (
                  <div
                    key={key}
                    className="flex items-center justify-between rounded-xs bg-control-bg px-3 py-2"
                  >
                    <span className="text-sm">
                      {key}:
                      {mixed ? (
                        <span className="italic text-control-placeholder ml-1">
                          {value}
                        </span>
                      ) : (
                        value
                      )}
                    </span>
                    <button
                      className="p-0.5 hover:bg-control-bg-hover rounded-xs"
                      onClick={() => removeLabel(key)}
                    >
                      <X className="size-3" />
                    </button>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-control-placeholder">
              {t("common.no-data")}
            </p>
          )}
        </SheetBody>
        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={applying}
            onClick={async () => {
              setApplying(true);
              try {
                await onApply(labelsList);
                onClose();
              } finally {
                setApplying(false);
              }
            }}
          >
            {t("common.update")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
