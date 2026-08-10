import { Search } from "lucide-react";
import type { ChangeEvent, ComponentProps } from "react";
import {
  forwardRef,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { DEBOUNCE_SEARCH_DELAY } from "@/types/common";
import { Input } from "./input";

type SearchInputProps = Omit<ComponentProps<typeof Input>, "className"> & {
  className?: string;
  wrapperClassName?: string;
};

const stringifyInputValue = (
  value: ComponentProps<typeof Input>["value"]
): string => {
  if (value === undefined || value === null) return "";
  return Array.isArray(value) ? value.join(",") : String(value);
};

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput(
    {
      className,
      wrapperClassName,
      defaultValue,
      onChange,
      placeholder,
      style,
      value,
      ...props
    },
    ref
  ) {
    const { t } = useTranslation();
    const [rawValue, setRawValue] = useState(() =>
      stringifyInputValue(value ?? defaultValue)
    );
    const pendingChangeRef = useRef<ChangeEvent<HTMLInputElement> | undefined>(
      undefined
    );
    const onChangeRef = useRef(onChange);
    onChangeRef.current = onChange;

    useLayoutEffect(() => {
      if (value === undefined) return;
      pendingChangeRef.current = undefined;
      setRawValue(stringifyInputValue(value));
    }, [value]);

    useEffect(() => {
      const event = pendingChangeRef.current;
      if (!event) return;
      const timer = window.setTimeout(() => {
        if (pendingChangeRef.current !== event) return;
        pendingChangeRef.current = undefined;
        onChangeRef.current?.(event);
      }, DEBOUNCE_SEARCH_DELAY);
      return () => window.clearTimeout(timer);
    }, [rawValue]);

    return (
      <div className={cn("relative flex-1", wrapperClassName)}>
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
        <Input
          ref={ref}
          autoComplete="off"
          className={cn("pl-8", className)}
          placeholder={placeholder ?? t("common.type-to-search")}
          style={{ ...style, paddingInlineStart: "2rem" }}
          value={rawValue}
          onChange={(event) => {
            pendingChangeRef.current = event;
            setRawValue(event.target.value);
          }}
          {...props}
        />
      </div>
    );
  }
);
