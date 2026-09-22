import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { Button } from "@/components/ui/button";
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
import { useEnvironmentList } from "@/hooks/useAppState";

export function EditEnvironmentSheet({
  open,
  onClose,
  onUpdate,
}: {
  open: boolean;
  onClose: () => void;
  onUpdate: (environment: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const environments = useEnvironmentList();
  const [selected, setSelected] = useState("");
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    if (open) {
      setSelected("");
      setUpdating(false);
    }
  }, [open]);

  const selectedEnvironment = environments.find(
    (environment) => environment.name === selected
  );

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("database.edit-environment")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <Select
            value={selected}
            onValueChange={(value) => setSelected(String(value))}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t("common.select")}>
                {selectedEnvironment && (
                  <EnvironmentLabel environment={selectedEnvironment} />
                )}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {environments.map((env) => (
                <SelectItem key={env.name} value={env.name}>
                  <EnvironmentLabel environment={env} />
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SheetBody>
        <SheetFooter>
          <Button appearance="secondary" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!selected || updating}
            onClick={async () => {
              setUpdating(true);
              try {
                await onUpdate(selected);
                onClose();
              } finally {
                setUpdating(false);
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
