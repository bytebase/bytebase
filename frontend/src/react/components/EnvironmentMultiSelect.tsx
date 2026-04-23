import { Check, ChevronDown, X } from "lucide-react";
import {
  type CSSProperties,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import {
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
} from "@/react/components/ui/combobox-position";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useEnvironmentV1Store } from "@/store";

export function EnvironmentMultiSelect({
  value,
  onChange,
}: Readonly<{
  value: string[];
  onChange: (envs: string[]) => void;
}>) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environmentList = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [dropdownStyle, setDropdownStyle] = useState<CSSProperties>({});

  useLayoutEffect(() => {
    if (!open || !containerRef.current) return;

    const updateDropdownPosition = () => {
      if (!containerRef.current) return;
      const triggerRect = containerRef.current.getBoundingClientRect();
      const dropdownHeight = dropdownRef.current?.offsetHeight ?? 0;
      const nextStyle = getPortalDropdownStyle(
        triggerRect,
        dropdownHeight,
        window.innerHeight
      );
      setDropdownStyle((previousStyle) =>
        isPortalDropdownStyleEqual(previousStyle, nextStyle)
          ? previousStyle
          : nextStyle
      );
    };

    const handleScroll = (event: Event) => {
      if (shouldIgnorePortalDropdownScroll(event.target, dropdownRef.current)) {
        return;
      }
      updateDropdownPosition();
    };

    updateDropdownPosition();
    const resizeObserver =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(updateDropdownPosition);
    resizeObserver?.observe(containerRef.current);
    window.addEventListener("resize", updateDropdownPosition);
    window.addEventListener("scroll", handleScroll, true);

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener("resize", updateDropdownPosition);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [open, environmentList, value]);

  useEffect(() => {
    if (!open) return;

    const handler = (event: MouseEvent) => {
      const target = event.target as Node;
      if (containerRef.current?.contains(target)) return;
      if (dropdownRef.current?.contains(target)) return;
      setOpen(false);
    };

    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const toggle = (name: string) => {
    onChange(
      value.includes(name) ? value.filter((v) => v !== name) : [...value, name]
    );
  };

  const remove = (name: string) => {
    onChange(value.filter((v) => v !== name));
  };

  // Use native <button> elements for the trigger and option rows so they
  // get keyboard (Enter/Space) and focus semantics for free — SonarCloud's
  // S6848 / S1082 a11y rules require interactive content to be native.
  // The selected-chip X buttons are rendered as siblings of the trigger
  // (not children) to avoid nested <button>, which is invalid HTML.
  return (
    <div ref={containerRef} className="relative">
      <div
        className={cn(
          "flex items-center flex-wrap gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm",
          open && "border-accent"
        )}
      >
        {value.length === 0 && (
          <span className="text-control-placeholder">
            {t("environment.select")}
          </span>
        )}
        {value.map((name) => (
          <span
            key={name}
            className="inline-flex items-center gap-x-1 rounded-xs border border-control-border px-1 py-0.5 text-xs"
          >
            <EnvironmentLabel environmentName={name} className="text-xs" />
            <button
              type="button"
              className="text-control-light hover:text-control"
              onClick={() => remove(name)}
              aria-label={t("common.delete")}
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        <button
          type="button"
          className="ml-auto flex items-center text-control-light cursor-pointer"
          onClick={() => setOpen(!open)}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label={t("environment.select")}
        >
          <ChevronDown className="h-4 w-4 shrink-0" />
        </button>
      </div>

      {open &&
        createPortal(
          <div
            ref={dropdownRef}
            className={`fixed max-h-60 overflow-auto rounded-sm border border-control-border bg-background shadow-lg ${LAYER_SURFACE_CLASS}`}
            style={dropdownStyle}
          >
            {environmentList.length === 0 && (
              <div className="px-3 py-2 text-sm text-control-light">
                {t("common.no-data")}
              </div>
            )}
            {environmentList.map((env) => {
              const selected = value.includes(env.name);
              return (
                <button
                  key={env.name}
                  type="button"
                  className="flex w-full cursor-pointer items-center gap-x-2 px-3 py-1.5 text-left text-sm hover:bg-control-bg"
                  onClick={() => toggle(env.name)}
                  aria-pressed={selected}
                >
                  <div
                    className={cn(
                      "flex size-4 shrink-0 items-center justify-center rounded-xs border",
                      selected
                        ? "border-accent bg-accent text-accent-text"
                        : "border-control-border"
                    )}
                  >
                    {selected && <Check className="h-3 w-3" />}
                  </div>
                  <EnvironmentLabel environment={env} />
                </button>
              );
            })}
          </div>,
          getLayerRoot("overlay")
        )}
    </div>
  );
}
