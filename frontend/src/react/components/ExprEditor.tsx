import { Plus, X } from "lucide-react";
import type { ReactNode } from "react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  factorText,
  getOperatorListByFactor,
} from "@/components/ExprEditor/components/common";
import type { OptionConfig } from "@/components/ExprEditor/context";
import type {
  ConditionExpr,
  ConditionGroupExpr,
  Factor,
  Operator,
  RawStringExpr,
} from "@/plugins/cel";
import {
  ExprType,
  isBooleanFactor,
  isCollectionOperator,
  isConditionExpr,
  isConditionGroupExpr,
  isNumberFactor,
  isRawStringExpr,
  isStringFactor,
  isStringOperator,
  isTimestampFactor,
  LogicalOperatorList,
} from "@/plugins/cel";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID } from "@/utils/cel-attributes";

// Re-export OptionConfig so consumers can import from one place.
export type { OptionConfig } from "@/components/ExprEditor/context";

// ============================================================
// Context
// ============================================================

interface ExprEditorContextValue {
  readonly: boolean;
  enableRawExpression: boolean;
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
  factorOperatorOverrideMap: Map<Factor, Operator[]> | undefined;
}

const ExprEditorContext = createContext<ExprEditorContextValue>({
  readonly: false,
  enableRawExpression: true,
  factorList: [],
  optionConfigMap: new Map(),
  factorOperatorOverrideMap: undefined,
});

function useExprEditorCtx() {
  return useContext(ExprEditorContext);
}

// ============================================================
// PortaledDropdown — renders dropdown via portal to avoid
// overflow clipping and z-index issues inside dialogs
// ============================================================

function useClickOutside(
  refs: React.RefObject<HTMLElement | null>[],
  open: boolean,
  onClose: () => void
) {
  const refsRef = useRef(refs);
  refsRef.current = refs;
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (refsRef.current.every((r) => !r.current?.contains(target))) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open, onClose]);
}

function PortaledDropdown({
  anchorRef,
  dropdownRef,
  matchAnchorWidth,
  className,
  children,
}: {
  anchorRef: React.RefObject<HTMLElement | null>;
  dropdownRef?: React.RefObject<HTMLDivElement | null>;
  matchAnchorWidth?: boolean;
  className?: string;
  children: ReactNode;
}) {
  const [style, setStyle] = useState<React.CSSProperties>({});

  const updatePosition = useCallback(() => {
    if (!anchorRef.current) return;
    const rect = anchorRef.current.getBoundingClientRect();
    setStyle({
      position: "fixed",
      top: rect.bottom + 4,
      left: rect.left,
      zIndex: 100,
      ...(matchAnchorWidth ? { width: rect.width } : {}),
    });
  }, [anchorRef, matchAnchorWidth]);

  useLayoutEffect(() => {
    updatePosition();
    window.addEventListener("scroll", updatePosition, true);
    window.addEventListener("resize", updatePosition);

    const anchor = anchorRef.current;
    let ro: ResizeObserver | undefined;
    if (anchor) {
      ro = new ResizeObserver(updatePosition);
      ro.observe(anchor);
    }

    return () => {
      window.removeEventListener("scroll", updatePosition, true);
      window.removeEventListener("resize", updatePosition);
      ro?.disconnect();
    };
  }, [updatePosition, anchorRef]);

  return createPortal(
    <div ref={dropdownRef} style={style} className={className}>
      {children}
    </div>,
    document.body
  );
}

// ============================================================
// SearchableSelect — async searchable dropdown
// ============================================================

interface SearchableSelectOption {
  label: string;
  value: string;
}

function SearchableSelect({
  value,
  optionConfig,
  disabled,
  placeholder,
  onChange,
}: {
  value: string;
  optionConfig: OptionConfig;
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<SearchableSelectOption[]>([]);
  const [loading, setLoading] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const selectedLabel = useMemo(() => {
    const found = options.find((o) => o.value === value);
    if (found) return found.label;
    if (optionConfig.fallback)
      return optionConfig.fallback(value)?.label ?? value;
    return value || "";
  }, [options, value, optionConfig]);

  const doSearch = useCallback(
    async (q: string) => {
      if (!optionConfig.search) return;
      setLoading(true);
      try {
        const resp = await optionConfig.search({
          search: q,
          pageToken: "",
          pageSize: 20,
        });
        setOptions(resp.options as SearchableSelectOption[]);
      } finally {
        setLoading(false);
      }
    },
    [optionConfig]
  );

  useEffect(() => {
    if (!value) return;
    if (optionConfig.fetch) {
      optionConfig.fetch([value]).then((opts) => {
        setOptions(opts as SearchableSelectOption[]);
      });
    } else if (optionConfig.search) {
      doSearch(value);
    }
  }, []);

  const handleOpen = () => {
    setOpen(true);
    doSearch(search);
  };

  const handleSearchChange = (q: string) => {
    setSearch(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(q), 300);
  };

  const handleSelect = (v: string) => {
    onChange(v);
    setOpen(false);
    setSearch("");
  };

  const close = useCallback(() => setOpen(false), []);
  useClickOutside([triggerRef, dropdownRef], open, close);

  if (!optionConfig.search) {
    return (
      <Select
        value={value}
        disabled={disabled}
        onValueChange={(val) => {
          if (val != null) onChange(val);
        }}
      >
        <SelectTrigger className="min-w-28">
          <SelectValue placeholder={placeholder}>
            {(value: string | null) => {
              if (!value) return null;
              const found = optionConfig.options.find((o) => o.value === value);
              return found?.label ?? value;
            }}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          {optionConfig.options.map((o) => (
            <SelectItem key={o.value as string} value={o.value as string}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  return (
    <div className="min-w-28">
      <button
        ref={triggerRef}
        type="button"
        className="h-8 px-2 text-sm rounded-xs border border-control-border bg-white w-full text-left disabled:opacity-50 truncate"
        disabled={disabled}
        onClick={handleOpen}
      >
        {selectedLabel || (
          <span className="text-control-placeholder">{placeholder ?? ""}</span>
        )}
      </button>
      {open && (
        <PortaledDropdown
          anchorRef={triggerRef}
          dropdownRef={dropdownRef}
          className="w-56 bg-white border border-control-border rounded-xs shadow-md"
        >
          <div className="p-1 border-b border-control-border">
            <input
              autoFocus
              className="w-full h-8 px-2 text-sm rounded-xs border border-control-border outline-none"
              placeholder={t("common.filter-by-name")}
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-y-auto py-1">
            {loading && (
              <li className="px-3 py-1 text-sm text-control-placeholder">
                {t("common.loading")}
              </li>
            )}
            {!loading && options.length === 0 && (
              <li className="px-3 py-1 text-sm text-control-placeholder">
                {t("common.no-data")}
              </li>
            )}
            {options.map((o) => (
              <li
                key={o.value}
                className={`px-3 py-1 text-sm cursor-pointer hover:bg-gray-100 ${
                  o.value === value ? "font-medium text-accent" : ""
                }`}
                onMouseDown={() => handleSelect(o.value)}
              >
                {o.label}
              </li>
            ))}
          </ul>
        </PortaledDropdown>
      )}
    </div>
  );
}

// ============================================================
// MultiSearchableSelect — async searchable multi-select
// ============================================================

function MultiSearchableSelect({
  value,
  optionConfig,
  disabled,
  placeholder,
  onChange,
}: {
  value: string[];
  optionConfig: OptionConfig;
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string[]) => void;
}) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [searchOptions, setSearchOptions] = useState<SearchableSelectOption[]>(
    []
  );
  const [loading, setLoading] = useState(false);
  const triggerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [knownOptions, setKnownOptions] = useState<SearchableSelectOption[]>(
    []
  );

  const allOptions = useMemo(() => {
    if (optionConfig.search) {
      const merged = new Map<string, SearchableSelectOption>();
      for (const o of knownOptions) merged.set(o.value, o);
      for (const o of searchOptions) merged.set(o.value, o);
      return Array.from(merged.values());
    }
    return optionConfig.options as SearchableSelectOption[];
  }, [optionConfig, knownOptions, searchOptions]);

  const getLabelForValue = useCallback(
    (v: string) => {
      const found = allOptions.find((o) => o.value === v);
      if (found) return found.label;
      if (optionConfig.fallback) return optionConfig.fallback(v)?.label ?? v;
      return v;
    },
    [allOptions, optionConfig]
  );

  const doSearch = useCallback(
    async (q: string) => {
      if (!optionConfig.search) return;
      setLoading(true);
      try {
        const resp = await optionConfig.search({
          search: q,
          pageToken: "",
          pageSize: 20,
        });
        setSearchOptions(resp.options as SearchableSelectOption[]);
      } finally {
        setLoading(false);
      }
    },
    [optionConfig]
  );

  useEffect(() => {
    if (value.length === 0) return;
    if (optionConfig.fetch) {
      optionConfig.fetch(value).then((opts) => {
        setKnownOptions(opts as SearchableSelectOption[]);
      });
    } else if (optionConfig.search) {
      doSearch("");
    }
  }, []);

  const handleOpen = () => {
    setOpen(true);
    doSearch(search);
  };

  const handleSearchChange = (q: string) => {
    setSearch(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(q), 300);
  };

  const toggleValue = (v: string) => {
    if (value.includes(v)) {
      onChange(value.filter((x) => x !== v));
    } else {
      onChange([...value, v]);
    }
  };

  const removeValue = (v: string) => {
    onChange(value.filter((x) => x !== v));
  };

  const close = useCallback(() => setOpen(false), []);
  useClickOutside([triggerRef, dropdownRef], open, close);

  if (!optionConfig.search) {
    return (
      <MultiCheckSelect
        value={value}
        options={optionConfig.options as { label: string; value: string }[]}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
      />
    );
  }

  return (
    <div className="min-w-32 max-w-xs">
      <div
        ref={triggerRef}
        className="min-h-8 px-2 py-0.5 text-sm rounded-xs border border-control-border bg-white flex flex-wrap gap-1 cursor-pointer"
        onClick={disabled ? undefined : handleOpen}
      >
        {value.length === 0 && (
          <span className="text-control-placeholder text-sm leading-6">
            {placeholder ?? ""}
          </span>
        )}
        {value.map((v) => (
          <span
            key={v}
            className="inline-flex items-center gap-1 bg-gray-100 text-xs px-1.5 py-0.5 rounded"
          >
            {getLabelForValue(v)}
            {!disabled && (
              <button
                type="button"
                className="text-gray-400 hover:text-gray-600"
                onMouseDown={(e) => {
                  e.stopPropagation();
                  removeValue(v);
                }}
              >
                ×
              </button>
            )}
          </span>
        ))}
      </div>
      {open && (
        <PortaledDropdown
          anchorRef={triggerRef}
          dropdownRef={dropdownRef}
          className="w-56 bg-white border border-control-border rounded-xs shadow-md"
        >
          <div className="p-1 border-b border-control-border">
            <input
              autoFocus
              className="w-full h-8 px-2 text-sm rounded-xs border border-control-border outline-none"
              placeholder={t("common.filter-by-name")}
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
            />
          </div>
          <ul className="max-h-48 overflow-y-auto py-1">
            {loading && (
              <li className="px-3 py-1 text-sm text-control-placeholder">
                {t("common.loading")}
              </li>
            )}
            {!loading && allOptions.length === 0 && (
              <li className="px-3 py-1 text-sm text-control-placeholder">
                {t("common.no-data")}
              </li>
            )}
            {allOptions.map((o) => (
              <li
                key={o.value}
                className={`px-3 py-1 text-sm cursor-pointer hover:bg-gray-100 flex items-center gap-2 ${
                  value.includes(o.value) ? "font-medium" : ""
                }`}
                onMouseDown={() => toggleValue(o.value)}
              >
                <input
                  type="checkbox"
                  readOnly
                  checked={value.includes(o.value)}
                  className="pointer-events-none"
                />
                {o.label}
              </li>
            ))}
          </ul>
        </PortaledDropdown>
      )}
    </div>
  );
}

// ============================================================
// TagInput — multi-value tag input (no dropdown)
// ============================================================

function TagInput({
  value,
  disabled,
  placeholder,
  onChange,
}: {
  value: string[];
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string[]) => void;
}) {
  const [inputValue, setInputValue] = useState("");

  const addTag = (raw: string) => {
    const tags = raw
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    if (tags.length === 0) return;
    const next = Array.from(new Set([...value, ...tags]));
    onChange(next);
    setInputValue("");
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addTag(inputValue);
    } else if (e.key === "Backspace" && inputValue === "" && value.length > 0) {
      onChange(value.slice(0, -1));
    }
  };

  const removeTag = (tag: string) => {
    onChange(value.filter((v) => v !== tag));
  };

  return (
    <div className="flex flex-wrap items-center gap-1 min-h-8 px-2 py-0.5 rounded-xs border border-control-border bg-white min-w-64 max-w-xs">
      {value.map((tag) => (
        <span
          key={tag}
          className="inline-flex items-center gap-1 bg-gray-100 text-xs px-1.5 py-0.5 rounded"
        >
          {tag}
          {!disabled && (
            <button
              type="button"
              className="text-gray-400 hover:text-gray-600"
              onClick={() => removeTag(tag)}
            >
              ×
            </button>
          )}
        </span>
      ))}
      {!disabled && (
        <input
          className="flex-1 min-w-16 h-6 text-sm outline-none bg-transparent"
          placeholder={value.length === 0 ? (placeholder ?? "") : ""}
          value={inputValue}
          disabled={disabled}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => {
            if (inputValue) addTag(inputValue);
          }}
        />
      )}
    </div>
  );
}

// ============================================================
// MultiCheckSelect — dropdown with checkboxes + "All" toggle
// ============================================================

function MultiCheckSelect({
  value,
  options,
  renderValue,
  disabled,
  placeholder,
  onChange,
}: {
  value: string[];
  options: { label: string; value: string }[];
  renderValue?: (value: string, fallbackLabel?: string) => ReactNode;
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string[]) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const allValues = options.map((o) => o.value);
  const allSelected =
    allValues.length > 0 && allValues.every((v) => value.includes(v));
  const anySelected = value.length > 0;

  const getLabelForValue = (v: string) =>
    options.find((o) => o.value === v)?.label ?? v;

  const toggleValue = (v: string) => {
    if (value.includes(v)) {
      onChange(value.filter((x) => x !== v));
    } else {
      onChange([...value, v]);
    }
  };

  const close = useCallback(() => setOpen(false), []);
  useClickOutside([triggerRef, dropdownRef], open, close);

  return (
    <div className="min-w-32">
      <button
        ref={triggerRef}
        type="button"
        className="inline-flex items-center gap-1 min-h-8 w-full px-2 py-0.5 text-sm rounded-xs border border-control-border bg-white text-left hover:bg-control-bg disabled:pointer-events-none disabled:opacity-50 flex-wrap"
        disabled={disabled}
        onClick={() => setOpen(!open)}
      >
        {value.length === 0 && (
          <span className="text-control-placeholder">{placeholder}</span>
        )}
        {value.map((v) => (
          <span
            key={v}
            className="inline-flex items-center gap-1 bg-gray-100 text-xs px-1.5 py-0.5 rounded"
          >
            {renderValue
              ? renderValue(v, getLabelForValue(v))
              : getLabelForValue(v)}
            {!disabled && (
              <span
                role="button"
                className="text-gray-400 hover:text-gray-600"
                onMouseDown={(e) => {
                  e.stopPropagation();
                  e.preventDefault();
                  onChange(value.filter((x) => x !== v));
                }}
              >
                ×
              </span>
            )}
          </span>
        ))}
      </button>
      {open && (
        <PortaledDropdown
          anchorRef={triggerRef}
          dropdownRef={dropdownRef}
          matchAnchorWidth
          className="bg-white border border-control-border rounded-sm shadow-md py-1"
        >
          <label className="flex items-center gap-2 px-3 py-1.5 text-sm cursor-pointer border-b border-control-border hover:bg-control-bg">
            <input
              type="checkbox"
              checked={allSelected}
              ref={(el) => {
                if (el) el.indeterminate = anySelected && !allSelected;
              }}
              onChange={(e) => {
                if (e.target.checked) {
                  onChange(allValues);
                } else {
                  onChange([]);
                }
              }}
            />
            {t("common.all")}
          </label>
          {options.map((o) => (
            <label
              key={o.value}
              className="flex items-center gap-2 px-3 py-1.5 text-sm cursor-pointer hover:bg-control-bg"
            >
              <input
                type="checkbox"
                checked={value.includes(o.value)}
                onChange={() => toggleValue(o.value)}
              />
              {renderValue ? renderValue(o.value, o.label) : o.label}
            </label>
          ))}
        </PortaledDropdown>
      )}
    </div>
  );
}

// ============================================================
// FactorSelect
// ============================================================

function FactorSelect({
  expr,
  onUpdate,
}: {
  expr: ConditionExpr;
  onUpdate: () => void;
}) {
  const { readonly, factorList } = useExprEditorCtx();
  const factor = expr.args[0] as Factor;

  useEffect(() => {
    if (factorList.length === 0) return;
    if (!factorList.includes(factor)) {
      (expr.args as unknown[])[0] = factorList[0];
      onUpdate();
    }
  }, [factor, factorList]);

  return (
    <Select
      value={factor}
      disabled={readonly}
      onValueChange={(val) => {
        (expr.args as unknown[])[0] = val as Factor;
        onUpdate();
      }}
    >
      <SelectTrigger className="shrink-0">
        <SelectValue>
          {(value: string | null) =>
            value ? factorText(value as Factor) : null
          }
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {factorList.map((f) => (
          <SelectItem key={f} value={f}>
            {factorText(f)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// ============================================================
// OperatorSelect
// ============================================================

const OPERATOR_DICT: Record<string, string> = {
  "_==_": "==",
  "_!=_": "!=",
  "_<_": "<",
  "_<=_": "≤",
  "_>=_": "≥",
  "_>_": ">",
  "@not_in": "not in",
  "@not_contains": "not contains",
};

function operatorLabel(op: string): string {
  return OPERATOR_DICT[op] ?? op.replace(/^@/, "");
}

function OperatorSelect({
  expr,
  onUpdate,
}: {
  expr: ConditionExpr;
  onUpdate: () => void;
}) {
  const { readonly, factorOperatorOverrideMap: overrideMap } =
    useExprEditorCtx();
  const factor = expr.args[0] as Factor;

  const operators = useMemo(
    () => getOperatorListByFactor(factor, overrideMap),
    [factor, overrideMap]
  );

  useEffect(() => {
    if (operators.length === 0) return;
    if (!operators.includes(expr.operator)) {
      expr.operator = operators[0] as typeof expr.operator;
      onUpdate();
    }
  }, [operators, expr.operator]);

  return (
    <Select
      value={expr.operator}
      disabled={readonly}
      onValueChange={(val) => {
        expr.operator = val as typeof expr.operator;
        onUpdate();
      }}
    >
      <SelectTrigger className="shrink-0">
        <SelectValue>
          {(value: string | null) => (value ? operatorLabel(value) : null)}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {operators.map((op) => (
          <SelectItem key={op} value={op}>
            {operatorLabel(op)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// ============================================================
// ValueInput
// ============================================================

function ValueInput({
  expr,
  onUpdate,
}: {
  expr: ConditionExpr;
  onUpdate: () => void;
}) {
  const { t } = useTranslation();
  const { readonly, optionConfigMap } = useExprEditorCtx();
  const factor = expr.args[0] as Factor;
  const operator = expr.operator;

  const optionConfig = useMemo(
    () => optionConfigMap.get(factor) ?? { options: [] },
    [optionConfigMap, factor]
  );

  const isEnvironment = factor === CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID;
  const renderOptionValue = (value: string, fallbackLabel?: string) => {
    if (isEnvironment) {
      return (
        <EnvironmentLabel
          environmentName={`${environmentNamePrefix}${value}`}
        />
      );
    }
    return fallbackLabel ?? value;
  };

  const hasOption = optionConfig.options.length > 0 || !!optionConfig.search;

  type InputType = "INPUT" | "SINGLE-SELECT" | "MULTI-SELECT" | "MULTI-INPUT";

  const isArrayValue = isCollectionOperator(operator);

  let inputType: InputType;
  if (isArrayValue) {
    inputType = hasOption ? "MULTI-SELECT" : "MULTI-INPUT";
  } else if (isStringOperator(operator)) {
    // String operators (startsWith, contains, etc.) always use text input
    inputType = "INPUT";
  } else if (hasOption) {
    inputType = "SINGLE-SELECT";
  } else {
    inputType = "INPUT";
  }

  const isNumberValue = isNumberFactor(factor);

  // Reset value when factor or operator changes, matching Vue's watch([factor, operator]) behavior.
  const prevRef = useRef({ factor, operator });
  useEffect(() => {
    const prev = prevRef.current;
    const changed = prev.factor !== factor || prev.operator !== operator;
    prevRef.current = { factor, operator };
    if (!changed) return;

    if (isNumberFactor(factor)) {
      if (isCollectionOperator(operator)) {
        (expr.args as unknown[])[1] = [];
      } else {
        (expr.args as unknown[])[1] = 0;
      }
    } else if (isBooleanFactor(factor)) {
      (expr.args as unknown[])[1] = true;
    } else if (isStringFactor(factor)) {
      if (isCollectionOperator(operator)) {
        (expr.args as unknown[])[1] = [];
      } else {
        (expr.args as unknown[])[1] = "";
      }
    }
    onUpdate();
  }, [factor, operator]);

  const getStringValue = () => {
    const v = expr.args[1];
    if (typeof v === "string") return v;
    if (typeof v === "number") return String(v);
    return "";
  };
  const getNumberValue = () =>
    typeof expr.args[1] === "number" ? expr.args[1] : 0;
  const getArrayValue = (): string[] => {
    if (!Array.isArray(expr.args[1])) return [];
    return (expr.args[1] as unknown[]).map(String);
  };

  const setStringValue = (v: string) => {
    // For number factors, store as number even when selected from a string-valued dropdown
    (expr.args as unknown[])[1] = isNumberFactor(factor) ? Number(v) : v;
    onUpdate();
  };
  const setNumberValue = (v: number) => {
    (expr.args as unknown[])[1] = v;
    onUpdate();
  };
  const setArrayValue = (v: string[]) => {
    if (isNumberFactor(factor)) {
      (expr.args as unknown[])[1] = v.map(Number);
    } else {
      (expr.args as unknown[])[1] = v;
    }
    onUpdate();
  };

  if (inputType === "INPUT") {
    if (isNumberValue) {
      return (
        <input
          type="number"
          className="h-8 px-2 text-sm rounded-xs border border-control-border bg-white disabled:opacity-50 max-w-20"
          value={getNumberValue()}
          disabled={readonly}
          onChange={(e) => setNumberValue(Number(e.target.value))}
          placeholder={t("cel.condition.input-value")}
        />
      );
    }
    return (
      <Input
        className="h-8 min-w-28 text-sm"
        value={getStringValue()}
        disabled={readonly}
        onChange={(e) => setStringValue(e.target.value)}
        placeholder={t("cel.condition.input-value")}
      />
    );
  }

  if (inputType === "SINGLE-SELECT") {
    if (optionConfig.search) {
      return (
        <SearchableSelect
          value={getStringValue()}
          optionConfig={optionConfig}
          disabled={readonly}
          placeholder={t("cel.condition.select-value")}
          onChange={setStringValue}
        />
      );
    }
    return (
      <Select
        value={getStringValue()}
        disabled={readonly}
        onValueChange={(val) => {
          if (val != null) setStringValue(val);
        }}
      >
        <SelectTrigger className="min-w-28">
          <SelectValue placeholder={t("cel.condition.select-value")}>
            {(value: string | null) => {
              if (!value) return null;
              const found = optionConfig.options.find((o) => o.value === value);
              return renderOptionValue(value, found?.label);
            }}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          {optionConfig.options.map((o) => (
            <SelectItem key={o.value as string} value={o.value as string}>
              {renderOptionValue(o.value as string, o.label)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  if (inputType === "MULTI-SELECT") {
    if (optionConfig.search) {
      return (
        <MultiSearchableSelect
          value={getArrayValue()}
          optionConfig={optionConfig}
          disabled={readonly}
          placeholder={t("cel.condition.select-value")}
          onChange={setArrayValue}
        />
      );
    }
    return (
      <MultiCheckSelect
        value={getArrayValue()}
        options={optionConfig.options as { label: string; value: string }[]}
        renderValue={isEnvironment ? renderOptionValue : undefined}
        disabled={readonly}
        placeholder={t("cel.condition.select-value")}
        onChange={setArrayValue}
      />
    );
  }

  return (
    <TagInput
      value={getArrayValue()}
      disabled={readonly}
      placeholder={t("cel.condition.input-value-press-enter")}
      onChange={setArrayValue}
    />
  );
}

// ============================================================
// ConditionRow
// ============================================================

function ConditionRow({
  expr,
  onRemove,
  onUpdate,
}: {
  expr: ConditionExpr;
  onRemove: () => void;
  onUpdate: () => void;
}) {
  const { readonly } = useExprEditorCtx();

  return (
    <div className="w-full flex items-center gap-x-1">
      <FactorSelect expr={expr} onUpdate={onUpdate} />
      <OperatorSelect expr={expr} onUpdate={onUpdate} />
      <div className="flex-1 min-w-0">
        <ValueInput expr={expr} onUpdate={onUpdate} />
      </div>
      {!readonly && (
        <button
          type="button"
          className="shrink-0 w-7 h-7 flex items-center justify-center rounded-xs text-control-placeholder hover:text-control hover:bg-gray-100"
          onClick={onRemove}
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}

// ============================================================
// RawStringEditor
// ============================================================

function RawStringEditor({
  expr,
  onRemove,
  onUpdate,
}: {
  expr: RawStringExpr;
  onRemove: () => void;
  onUpdate: () => void;
}) {
  const { readonly } = useExprEditorCtx();

  return (
    <div className="w-full flex items-start gap-x-1">
      <textarea
        className="flex-1 min-h-16 max-h-24 px-2 py-1 text-sm rounded-xs border border-control-border bg-white resize-y disabled:opacity-50"
        placeholder="Enter raw CEL expression"
        value={expr.content}
        disabled={readonly}
        rows={2}
        onChange={(e) => {
          expr.content = e.target.value;
          onUpdate();
        }}
      />
      {!readonly && (
        <button
          type="button"
          className="shrink-0 w-7 h-7 flex items-center justify-center rounded-xs text-control-placeholder hover:text-control hover:bg-gray-100"
          onClick={onRemove}
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}

// ============================================================
// ConditionGroup (recursive)
// ============================================================

function ConditionGroup({
  expr,
  root = false,
  onRemove,
  onUpdate,
}: {
  expr: ConditionGroupExpr;
  root?: boolean;
  onRemove?: () => void;
  onUpdate: () => void;
}) {
  const { t } = useTranslation();
  const {
    readonly,
    factorList,
    enableRawExpression,
    factorOperatorOverrideMap: overrideMap,
  } = useExprEditorCtx();

  const operator = expr.operator;
  const args = expr.args;

  const logicalLabel = (op: string) => {
    if (op === "_&&_") return "and";
    if (op === "_||_") return "or";
    return op;
  };

  const getDefaultValue = (
    factor: Factor
  ): string | number | boolean | Date => {
    if (isNumberFactor(factor)) return 0;
    if (isBooleanFactor(factor)) return true;
    if (isStringFactor(factor)) return "";
    if (isTimestampFactor(factor)) return new Date();
    return "";
  };

  const addCondition = () => {
    const factor = factorList[0];
    if (!factor) return;
    const operators = getOperatorListByFactor(factor, overrideMap);
    const op = operators[0];
    if (!op) return;
    args.push({
      type: ExprType.Condition,
      operator: op,
      args: [factor, getDefaultValue(factor)],
    } as ConditionExpr);
    onUpdate();
  };

  const addConditionGroup = () => {
    args.push({
      type: ExprType.ConditionGroup,
      operator: LogicalOperatorList[0],
      args: [],
    });
    onUpdate();
  };

  const addRawString = () => {
    args.push({
      type: ExprType.RawString,
      content: "",
    });
    onUpdate();
  };

  const removeOperand = (index: number) => {
    args.splice(index, 1);
    onUpdate();
  };

  return (
    <div
      className={`w-full flex flex-col gap-y-2 py-0.5 ${
        root ? "" : "border rounded-xs bg-gray-50"
      }`}
    >
      {!root && (
        <div className="pl-2.5 pr-1 text-gray-500 flex items-center">
          <div className="flex-1">
            {args.length > 0 ? (
              <>
                {operator === "_||_" && (
                  <span>{t("cel.condition.group.or.description")}</span>
                )}
                {operator === "_&&_" && (
                  <span>{t("cel.condition.group.and.description")}</span>
                )}
              </>
            ) : (
              <span className="inline-flex items-center">
                {t("cel.condition.add-condition-in-group-placeholder", {
                  plus: "+",
                })}
              </span>
            )}
          </div>
          {!readonly && (
            <div className="flex items-center justify-end">
              <button
                type="button"
                className="w-[22px] h-[22px] flex items-center justify-center rounded-xs hover:bg-gray-100"
                onClick={onRemove}
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
        </div>
      )}

      {root && args.length === 0 && (
        <div className="text-gray-500 text-sm">
          {t("cel.condition.add-root-condition-placeholder")}
        </div>
      )}

      {args.map((operand, i) => (
        <div
          key={i}
          className={`flex items-start gap-x-1 w-full ${root ? "" : "px-1"}`}
        >
          <div className="w-14 shrink-0">
            {i === 0 ? (
              <div className="pl-1.5 pt-1 text-control text-sm">Where</div>
            ) : i === 1 ? (
              <Select
                value={operator}
                disabled={readonly}
                onValueChange={(val) => {
                  expr.operator = val as typeof expr.operator;
                  onUpdate();
                }}
              >
                <SelectTrigger className="h-7 px-1.5">
                  <SelectValue>
                    {(value: string | null) =>
                      value ? logicalLabel(value) : null
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_&&_">{logicalLabel("_&&_")}</SelectItem>
                  <SelectItem value="_||_">{logicalLabel("_||_")}</SelectItem>
                </SelectContent>
              </Select>
            ) : (
              <div className="pl-2 pt-1 text-control text-sm lowercase">
                {logicalLabel(operator)}
              </div>
            )}
          </div>
          <div className="flex-1 flex flex-col gap-y-1 min-w-0">
            {isConditionGroupExpr(operand) && (
              <ConditionGroup
                key={i}
                expr={operand}
                onRemove={() => removeOperand(i)}
                onUpdate={onUpdate}
              />
            )}
            {isConditionExpr(operand) && (
              <ConditionRow
                expr={operand}
                onRemove={() => removeOperand(i)}
                onUpdate={onUpdate}
              />
            )}
            {isRawStringExpr(operand) && (
              <RawStringEditor
                expr={operand}
                onRemove={() => removeOperand(i)}
                onUpdate={onUpdate}
              />
            )}
          </div>
        </div>
      ))}

      {!root && (
        <div className="pl-1.5 pb-1 flex gap-x-1">
          <button
            type="button"
            className="inline-flex items-center gap-1 text-sm text-gray-500 px-1.5 py-0.5 rounded-xs hover:bg-gray-100 disabled:opacity-50"
            disabled={readonly}
            onClick={addCondition}
          >
            <Plus className="w-4 h-4" />
            {t("cel.condition.add")}
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-1 text-sm text-gray-500 px-1.5 py-0.5 rounded-xs hover:bg-gray-100 disabled:opacity-50"
            disabled={readonly}
            onClick={addRawString}
          >
            <Plus className="w-4 h-4" />
            {t("cel.condition.add-raw-expression")}
          </button>
        </div>
      )}

      {root && (
        <div className="flex gap-x-1">
          <button
            type="button"
            className="inline-flex items-center gap-1 text-sm px-1.5 py-0.5 rounded-xs hover:bg-gray-100 disabled:opacity-50"
            disabled={readonly}
            onClick={addCondition}
          >
            <Plus className="w-4 h-4" />
            {t("cel.condition.add")}
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-1 text-sm px-1.5 py-0.5 rounded-xs hover:bg-gray-100 disabled:opacity-50"
            disabled={readonly}
            onClick={addConditionGroup}
          >
            <Plus className="w-4 h-4" />
            {t("cel.condition.add-group")}
          </button>
          {enableRawExpression && (
            <button
              type="button"
              className="inline-flex items-center gap-1 text-sm px-1.5 py-0.5 rounded-xs hover:bg-gray-100 disabled:opacity-50"
              disabled={readonly}
              onClick={addRawString}
            >
              <Plus className="w-4 h-4" />
              {t("cel.condition.add-raw-expression")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================
// ExprEditor (public API)
// ============================================================

export interface ExprEditorProps {
  expr: ConditionGroupExpr;
  readonly?: boolean;
  enableRawExpression?: boolean;
  factorList: Factor[];
  optionConfigMap?: Map<Factor, OptionConfig>;
  factorOperatorOverrideMap?: Map<Factor, Operator[]>;
  onUpdate: () => void;
}

export function ExprEditor({
  expr,
  readonly = false,
  enableRawExpression = true,
  factorList,
  optionConfigMap = new Map(),
  factorOperatorOverrideMap: overrideMap,
  onUpdate,
}: ExprEditorProps) {
  const ctxValue: ExprEditorContextValue = useMemo(
    () => ({
      readonly,
      enableRawExpression,
      factorList,
      optionConfigMap,
      factorOperatorOverrideMap: overrideMap,
    }),
    [readonly, enableRawExpression, factorList, optionConfigMap, overrideMap]
  );

  return (
    <ExprEditorContext.Provider value={ctxValue}>
      <div className="bb-risk-expr-editor text-sm w-full">
        <ConditionGroup expr={expr} root onUpdate={onUpdate} />
      </div>
    </ExprEditorContext.Provider>
  );
}
