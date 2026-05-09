import { CodeIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { cn } from "@/react/lib/utils";
import type { BinaryFormat } from "./binary-format";

interface BinaryFormatButtonProps {
  format: BinaryFormat | undefined;
  onFormatChange: (format: BinaryFormat) => void;
}

export function BinaryFormatButton({
  format,
  onFormatChange,
}: BinaryFormatButtonProps) {
  const { t } = useTranslation();
  const current: BinaryFormat = format ?? "DEFAULT";
  const hasOverride = current !== "DEFAULT";

  const options: { value: BinaryFormat; label: string }[] = [
    { value: "DEFAULT", label: t("sql-editor.format-default") },
    { value: "BINARY", label: t("sql-editor.binary-format") },
    { value: "HEX", label: t("sql-editor.hex-format") },
    { value: "TEXT", label: t("sql-editor.text-format") },
    { value: "BOOLEAN", label: t("sql-editor.boolean-format") },
  ];

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant={hasOverride ? "default" : "outline"}
            size="sm"
            className={cn("ml-1 size-5 rounded-full p-0")}
            onClick={(e) => e.stopPropagation()}
          />
        }
      >
        <CodeIcon className="size-3" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-52 p-2">
        <div className="text-xs font-semibold mb-2">
          {t("sql-editor.column-display-format")}
        </div>
        <RadioGroup
          value={current}
          onValueChange={(next) => onFormatChange(next as BinaryFormat)}
          className="flex flex-col gap-2"
        >
          {options.map((option) => (
            <RadioGroupItem key={option.value} value={option.value}>
              {option.label}
            </RadioGroupItem>
          ))}
        </RadioGroup>
      </PopoverContent>
    </Popover>
  );
}
