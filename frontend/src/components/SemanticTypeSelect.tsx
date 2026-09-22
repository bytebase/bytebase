import { Pencil } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { SearchInput } from "@/components/ui/search-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSemanticTypes } from "@/hooks/useSemanticTypes";
import { getMaskingType } from "@/lib/sensitive-data/components-utils";
import { cn } from "@/lib/utils";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import {
  getSemanticTypeListWithBuiltins,
  isBuiltinSemanticTypeId,
} from "@/types/semanticTypes";

const EMPTY_SELECT_VALUE = "__EMPTY__";

function getSemanticTypeDescription(
  semanticType: SemanticTypeSetting_SemanticType,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (isBuiltinSemanticTypeId(semanticType.id)) {
    return t(
      `dynamic.settings.sensitive-data.semantic-types.template.${semanticType.id.split(".").join("-")}.algorithm.description`
    );
  }

  const maskingType = getMaskingType(semanticType.algorithm);
  if (maskingType) {
    return t("settings.sensitive-data.semantic-types.masking-effect", {
      effect: t(`settings.sensitive-data.algorithms.${maskingType}.self`),
    });
  }

  return semanticType.description;
}

export function SemanticTypeOption({
  className,
  semanticType,
  showId = false,
}: {
  className?: string;
  semanticType: SemanticTypeSetting_SemanticType;
  showId?: boolean;
}) {
  const { t } = useTranslation();
  const description = getSemanticTypeDescription(semanticType, t);

  return (
    <div className={cn("flex min-w-0 flex-col", className)}>
      <span className="truncate">{semanticType.title || semanticType.id}</span>
      {showId && semanticType.title && (
        <span className="text-xs text-control">{semanticType.id}</span>
      )}
      {description && (
        <span className="whitespace-normal text-xs text-control-light">
          {description}
        </span>
      )}
    </div>
  );
}

export function SemanticTypeSelect({
  "aria-label": ariaLabel,
  className,
  disabled = false,
  emptyLabel,
  id,
  onValueChange,
  placeholder,
  semanticTypeList,
  value,
}: {
  "aria-label"?: string;
  className?: string;
  disabled?: boolean;
  emptyLabel?: string;
  id?: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  semanticTypeList?: SemanticTypeSetting_SemanticType[];
  value: string;
}) {
  const { t } = useTranslation();
  const { semanticTypes: workspaceSemanticTypes } = useSemanticTypes();
  const semanticTypes = semanticTypeList
    ? getSemanticTypeListWithBuiltins(semanticTypeList, t)
    : workspaceSemanticTypes;
  const selectedSemanticType = semanticTypes.find(
    (semanticType) => semanticType.id === value
  );
  const normalizedValue = value || (emptyLabel ? EMPTY_SELECT_VALUE : "");
  const selectPlaceholder =
    placeholder ??
    emptyLabel ??
    t("settings.sensitive-data.semantic-types.select");

  return (
    <Select
      value={normalizedValue}
      disabled={disabled}
      onValueChange={(nextValue) => {
        onValueChange(
          !nextValue || nextValue === EMPTY_SELECT_VALUE ? "" : nextValue
        );
      }}
    >
      <SelectTrigger id={id} aria-label={ariaLabel} className={className}>
        <SelectValue placeholder={selectPlaceholder}>
          {selectedSemanticType?.title || value || selectPlaceholder}
        </SelectValue>
      </SelectTrigger>
      <SelectContent className="w-(--anchor-width)">
        {emptyLabel && (
          <SelectItem value={EMPTY_SELECT_VALUE}>{emptyLabel}</SelectItem>
        )}
        {semanticTypes.map((semanticType) => (
          <SelectItem
            key={semanticType.id}
            value={semanticType.id}
            className="h-auto items-start py-2"
          >
            <SemanticTypeOption semanticType={semanticType} />
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function SemanticTypePicker({
  onSelect,
  testId,
}: {
  onSelect: (semanticTypeId: string) => void;
  testId: string;
}) {
  const { t } = useTranslation();
  const { semanticTypes } = useSemanticTypes();
  const [open, setOpen] = useState(false);
  const [searchText, setSearchText] = useState("");

  useEffect(() => {
    if (open) {
      setSearchText("");
    }
  }, [open]);

  const filteredSemanticTypes = useMemo(() => {
    const normalizedSearch = searchText.trim().toLowerCase();
    if (!normalizedSearch) {
      return semanticTypes;
    }

    return semanticTypes.filter((semanticType) =>
      [semanticType.id, semanticType.title, semanticType.description].some(
        (value) => value?.toLowerCase().includes(normalizedSearch)
      )
    );
  }, [searchText, semanticTypes]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        aria-label={t("common.edit")}
        data-testid={testId}
        className="inline-flex size-5 items-center justify-center rounded-xs text-control transition-colors hover:bg-control-bg hover:text-main focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent [&_svg]:pointer-events-none"
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.stopPropagation();
          }
        }}
        onMouseDown={(event) => event.stopPropagation()}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
      >
        <Pencil className="size-3" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-96 p-0">
        <div className="flex flex-col gap-y-2 p-3">
          <SearchInput
            placeholder={t("common.filter-by-name")}
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
          />
          <div className="max-h-80 overflow-y-auto rounded-sm border border-block-border">
            {filteredSemanticTypes.length === 0 ? (
              <div className="px-4 py-6 text-sm text-control-light">
                {t("common.no-data")}
              </div>
            ) : (
              <div className="divide-y divide-block-border">
                {filteredSemanticTypes.map((semanticType) => (
                  <Button
                    key={semanticType.id}
                    type="button"
                    appearance="secondary"
                    size="sm"
                    data-testid={`semantic-type-option-${semanticType.id.replaceAll(".", "-")}`}
                    className="h-auto w-full flex-col items-start rounded-none px-4 py-3 text-left font-normal whitespace-normal"
                    onClick={() => {
                      onSelect(semanticType.id);
                      setOpen(false);
                    }}
                  >
                    <SemanticTypeOption semanticType={semanticType} showId />
                  </Button>
                ))}
              </div>
            )}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
