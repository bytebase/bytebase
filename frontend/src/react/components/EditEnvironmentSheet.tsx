import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Button } from "@/react/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { cn } from "@/react/lib/utils";

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

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("database.edit-environment")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <RadioGroup
            className="flex-col items-stretch gap-y-1"
            value={selected}
            onValueChange={(value) => setSelected(value as string)}
          >
            {environments.map((env) => (
              <RadioGroupItem
                key={env.name}
                value={env.name}
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2.5 rounded-sm cursor-pointer border transition-colors",
                  selected === env.name
                    ? "border-accent bg-accent/5"
                    : "border-transparent hover:bg-control-bg"
                )}
              >
                <EnvironmentLabel environment={env} />
              </RadioGroupItem>
            ))}
          </RadioGroup>
        </SheetBody>
        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
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
