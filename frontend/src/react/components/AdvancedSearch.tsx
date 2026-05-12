import { Filter, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";

// ============================================================
// Types
// ============================================================

export interface SearchScope {
  id: string;
  value: string;
  readonly?: boolean;
}

export interface SearchParams {
  query: string;
  scopes: SearchScope[];
}

export interface ValueOption {
  value: string;
  keywords: string[];
  /**
   * Custom renderer for the option's primary cell. Receives the current
   * search keyword so the renderer can highlight matches.
   */
  render?: (keyword: string) => React.ReactNode;
  custom?: boolean;
}

export interface ScopeOption {
  id: string;
  title: string;
  description?: string;
  options?: ValueOption[];
  allowMultiple?: boolean;
  /** Server-side search callback. When provided, options are fetched dynamically instead of filtered client-side. */
  onSearch?: (keyword: string) => Promise<ValueOption[]>;
}

export function emptySearchParams(): SearchParams {
  return { query: "", scopes: [] };
}

export function getValueFromScopes(params: SearchParams, id: string): string {
  return params.scopes.find((s) => s.id === id)?.value ?? "";
}

// ============================================================
// AdvancedSearch component
// ============================================================

interface AdvancedSearchProps {
  params: SearchParams;
  scopeOptions?: ScopeOption[];
  placeholder?: string;
  onParamsChange: (params: SearchParams) => void;
  /**
   * Fires on Enter while no scope/value dropdown is active. Hosts use this
   * for "press Enter to advance to the next match" — mirrors the Vue
   * `@keyup:enter` shortcut on the result-view search bar.
   */
  onEnter?: () => void;
}

const SCROLL_FADE_EPSILON = 1;

type OverflowAxis = "horizontal" | "vertical";

interface OverflowFadeState {
  showStart: boolean;
  showEnd: boolean;
}

function useOverflowFade<T extends HTMLElement>(
  ref: React.RefObject<T | null>,
  axis: OverflowAxis,
  deps: readonly unknown[]
): OverflowFadeState {
  const [fadeState, setFadeState] = useState<OverflowFadeState>({
    showStart: false,
    showEnd: false,
  });

  const updateFadeState = useCallback(() => {
    const el = ref.current;
    if (!el) {
      setFadeState({ showStart: false, showEnd: false });
      return;
    }

    const scrollOffset = axis === "horizontal" ? el.scrollLeft : el.scrollTop;
    const clientSize = axis === "horizontal" ? el.clientWidth : el.clientHeight;
    const scrollSize = axis === "horizontal" ? el.scrollWidth : el.scrollHeight;
    const hasOverflow = scrollSize > clientSize + SCROLL_FADE_EPSILON;

    setFadeState({
      showStart: hasOverflow && scrollOffset > SCROLL_FADE_EPSILON,
      showEnd:
        hasOverflow &&
        scrollOffset + clientSize < scrollSize - SCROLL_FADE_EPSILON,
    });
  }, [axis, ref]);

  useEffect(() => {
    const el = ref.current;
    if (!el) {
      setFadeState({ showStart: false, showEnd: false });
      return;
    }

    updateFadeState();

    el.addEventListener("scroll", updateFadeState, { passive: true });
    const observer = new ResizeObserver(updateFadeState);
    observer.observe(el);

    return () => {
      el.removeEventListener("scroll", updateFadeState);
      observer.disconnect();
    };
  }, [ref, updateFadeState, ...deps]);

  return fadeState;
}

function ScrollFade({
  edge,
  className,
}: {
  edge: "left" | "right" | "top" | "bottom";
  className?: string;
}) {
  const edgeClasses = {
    left: "inset-y-0 left-0 w-4 bg-gradient-to-r",
    right: "inset-y-0 right-0 w-4 bg-gradient-to-l",
    top: "inset-x-0 top-0 h-4 bg-gradient-to-b",
    bottom: "inset-x-0 bottom-0 h-4 bg-gradient-to-t",
  };

  return (
    <div
      className={cn(
        "pointer-events-none absolute",
        edgeClasses[edge],
        className
      )}
    />
  );
}

export function AdvancedSearch({
  params,
  scopeOptions = [],
  placeholder,
  onParamsChange,
  onEnter,
}: AdvancedSearchProps) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const tagsContainerRef = useRef<HTMLDivElement>(null);
  const scopeMenuListRef = useRef<HTMLDivElement>(null);
  const valueMenuListRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const [inputText, setInputText] = useState(params.query);
  const [menuView, setMenuView] = useState<"scope" | "value" | undefined>();
  const [currentScope, setCurrentScope] = useState<string | undefined>();
  const [menuIndex, setMenuIndex] = useState(0);
  const [focusedTagIndex, setFocusedTagIndex] = useState<number | undefined>();
  const [asyncOptions, setAsyncOptions] = useState<ValueOption[]>([]);
  const [asyncLoading, setAsyncLoading] = useState(false);
  const asyncSearchRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const asyncRequestRef = useRef(0);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Sync external query changes
  useEffect(() => {
    if (params.query !== inputText) {
      setInputText(params.query);
    }
  }, [params.query]);

  // Click outside to close
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setMenuView(undefined);
        setCurrentScope(undefined);
        setFocusedTagIndex(undefined);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  useEffect(() => {
    return () => {
      clearTimeout(asyncSearchRef.current);
      clearTimeout(debounceRef.current);
    };
  }, []);

  const editableScopes = useMemo(
    () => params.scopes.filter((s) => !s.readonly),
    [params.scopes]
  );

  const clearable = useMemo(
    () => params.query.trim().length > 0 || editableScopes.length > 0,
    [params.query, editableScopes]
  );

  // Current scope option definition
  const currentScopeOption = useMemo(
    () => scopeOptions.find((opt) => opt.id === currentScope),
    [scopeOptions, currentScope]
  );

  // Available scopes (hide already-selected unless allowMultiple)
  const availableScopeOptions = useMemo(() => {
    const existing = new Set(params.scopes.map((s) => s.id));
    return scopeOptions.filter(
      (opt) => !existing.has(opt.id) || opt.allowMultiple
    );
  }, [scopeOptions, params.scopes]);

  // Value extracted from input after "scope:" prefix
  const currentValueForScope = useMemo(() => {
    if (!currentScope) return "";
    // If user typed a prefix like "state:", strip it; otherwise use full input as filter
    const prefix = `${currentScope}:`;
    if (inputText.startsWith(prefix)) {
      return inputText.trim().toLowerCase().substring(prefix.length);
    }
    return inputText.trim().toLowerCase();
  }, [currentScope, inputText]);

  // Keyword typed before any "scope:" prefix — used to filter & highlight
  // the scope menu.
  const scopeKeyword = useMemo(
    () => inputText.trim().replace(/:.*$/, ""),
    [inputText]
  );

  // Visible scope options (filtered by typing)
  const visibleScopeOptions = useMemo(() => {
    if (currentScopeOption) return [currentScopeOption];
    const keyword = scopeKeyword.toLowerCase();
    if (!keyword) return availableScopeOptions;
    return availableScopeOptions.filter(
      (opt) =>
        opt.id.toLowerCase().includes(keyword) ||
        opt.title.toLowerCase().includes(keyword)
    );
  }, [currentScopeOption, scopeKeyword, availableScopeOptions]);

  const isAsyncScope = currentScopeOption?.onSearch != null;

  // Keep the latest onSearch in a ref so the timer always fires against the
  // freshest callback — without adding it as an effect dep. Page-level
  // callbacks (e.g. `searchDatabases`) often close over async-loaded state
  // like `project`, and their useCallback identity can churn on every render
  // when that state flows through unstable references (e.g. a fallback
  // `unknownProject()` that allocates a new proto each call). Depending on
  // that identity would refire this effect on every render and loop forever.
  const onSearchRef = useRef(currentScopeOption?.onSearch);
  onSearchRef.current = currentScopeOption?.onSearch;

  // Trigger async search when the current scope or its keyword changes.
  // Reset is folded into the same effect so we never race with it.
  useEffect(() => {
    clearTimeout(asyncSearchRef.current);
    asyncRequestRef.current += 1;
    // Clear previous results on every run so stale options from the prior
    // scope/keyword aren't selectable during the debounce + fetch window.
    setAsyncOptions([]);

    if (!currentScope || !onSearchRef.current) {
      setAsyncLoading(false);
      return;
    }

    const requestID = asyncRequestRef.current;
    setAsyncLoading(true);
    asyncSearchRef.current = setTimeout(() => {
      const fn = onSearchRef.current;
      if (!fn) {
        if (asyncRequestRef.current !== requestID) return;
        setAsyncOptions([]);
        setAsyncLoading(false);
        return;
      }
      fn(currentValueForScope)
        .then((results) => {
          if (asyncRequestRef.current !== requestID) return;
          setAsyncOptions(results);
          setMenuIndex(0);
        })
        .catch(() => {
          if (asyncRequestRef.current !== requestID) return;
          setAsyncOptions([]);
        })
        .finally(() => {
          if (asyncRequestRef.current !== requestID) return;
          setAsyncLoading(false);
        });
    }, 300);
    return () => {
      clearTimeout(asyncSearchRef.current);
    };
  }, [currentScope, currentValueForScope]);

  // Visible value options (filtered, de-duped)
  const visibleValueOptions = useMemo(() => {
    if (!currentScope || !currentScopeOption) return [];
    const selectedValues = new Set(
      params.scopes.filter((s) => s.id === currentScope).map((s) => s.value)
    );
    // Use async results for scopes with onSearch, static options otherwise
    const sourceOptions = isAsyncScope
      ? asyncOptions
      : (currentScopeOption.options ?? []);
    let options = sourceOptions.filter((opt) => !selectedValues.has(opt.value));
    // Only apply client-side filtering for static options
    if (!isAsyncScope) {
      const keyword = currentValueForScope.trim().toLowerCase();
      if (keyword) {
        options = options.filter(
          (opt) =>
            opt.value.toLowerCase().includes(keyword) ||
            opt.keywords.some((k) => k.toLowerCase().includes(keyword))
        );
      }
    }
    return options;
  }, [
    currentScope,
    currentScopeOption,
    params.scopes,
    currentValueForScope,
    isAsyncScope,
    asyncOptions,
  ]);

  const showMenu = useMemo(() => {
    if (menuView === "scope") return visibleScopeOptions.length > 0;
    if (menuView === "value") return true;
    return false;
  }, [menuView, visibleScopeOptions]);

  const tagFade = useOverflowFade(tagsContainerRef, "horizontal", [
    params.scopes,
  ]);
  const scopeMenuFade = useOverflowFade(scopeMenuListRef, "vertical", [
    menuView === "scope",
    visibleScopeOptions,
  ]);
  const valueMenuFade = useOverflowFade(valueMenuListRef, "vertical", [
    menuView === "value",
    visibleValueOptions,
    asyncLoading,
  ]);

  // ---- Actions ----

  const selectScope = useCallback((id: string | undefined) => {
    setCurrentScope(id);
    if (id) {
      setMenuView("value");
      setMenuIndex(0);
      // Show "scope:" prefix in the input to match Vue behavior
      setInputText(`${id}:`);
    } else {
      setMenuView("scope");
    }
  }, []);

  const selectValue = useCallback(
    (value: string) => {
      if (!currentScope || !currentScopeOption) {
        setMenuView(undefined);
        return;
      }
      const allowMultiple = currentScopeOption.allowMultiple;
      const updated = { ...params, scopes: [...params.scopes] };
      if (allowMultiple) {
        updated.scopes.push({ id: currentScope, value });
      } else {
        const idx = updated.scopes.findIndex((s) => s.id === currentScope);
        if (idx >= 0) {
          updated.scopes[idx] = { id: currentScope, value };
        } else {
          updated.scopes.push({ id: currentScope, value });
        }
      }
      updated.query = "";
      setInputText("");
      setCurrentScope(undefined);
      setMenuView(undefined);
      setFocusedTagIndex(undefined);
      onParamsChange(updated);
    },
    [currentScope, currentScopeOption, params, onParamsChange]
  );

  const removeScope = useCallback(
    (index: number) => {
      const removed = params.scopes[index];
      // If we're editing the scope being removed, reset menu state
      if (removed && currentScope === removed.id) {
        setCurrentScope(undefined);
        setMenuView(undefined);
      }
      onParamsChange({
        ...params,
        scopes: params.scopes.filter((_, i) => i !== index),
      });
    },
    [params, onParamsChange, currentScope]
  );

  const handleClear = useCallback(() => {
    const readonlyScopes = params.scopes.filter((s) => s.readonly);
    onParamsChange({ query: "", scopes: readonlyScopes });
    setInputText("");
    setMenuView(undefined);
    setCurrentScope(undefined);
    setFocusedTagIndex(undefined);
  }, [params, onParamsChange]);

  // Debounced query emit
  const emitQuery = useCallback(
    (text: string) => {
      clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        // Only emit if not editing a scoped value
        if (currentScope && text.startsWith(`${currentScope}:`)) return;
        if (text !== params.query) {
          onParamsChange({ ...params, query: text });
        }
      }, 300);
    },
    [currentScope, params, onParamsChange]
  );

  const handleInputChange = useCallback(
    (text: string) => {
      setInputText(text);

      if (menuView === "value") {
        // In value mode, input text is used to filter values — don't change mode
        return;
      }

      // Check if text matches a scope prefix
      const matched = availableScopeOptions.find((opt) =>
        text.startsWith(`${opt.id}:`)
      );
      if (matched) {
        selectScope(matched.id);
        return;
      }

      // Only show scope menu when input is empty (user is browsing filters).
      // When typing non-empty text, treat it as a plain query — don't push scope suggestions.
      if (!text.trim()) {
        if (!menuView) setMenuView("scope");
      } else {
        setMenuView(undefined);
      }

      emitQuery(text);
    },
    [menuView, availableScopeOptions, selectScope, emitQuery]
  );

  const handleInputClick = useCallback(() => {
    if (menuView === "value") {
      // Already showing value options, keep it
      return;
    }
    // Check scope prefix match
    const matched = availableScopeOptions.find((opt) =>
      inputText.startsWith(`${opt.id}:`)
    );
    if (matched) {
      selectScope(matched.id);
      return;
    }
    if (!menuView) {
      setMenuView("scope");
    }
  }, [availableScopeOptions, inputText, menuView, selectScope]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.nativeEvent.isComposing) return;

      if (e.key === "Backspace" && inputText === "") {
        e.stopPropagation();
        e.preventDefault();
        if (focusedTagIndex !== undefined) {
          removeScope(focusedTagIndex);
          setFocusedTagIndex(undefined);
        } else {
          const lastEditable = editableScopes.length - 1;
          if (lastEditable >= 0) {
            // Map back to original index
            const scope = editableScopes[lastEditable];
            const origIdx = params.scopes.indexOf(scope);
            setFocusedTagIndex(origIdx);
          }
        }
        return;
      }

      setFocusedTagIndex(undefined);

      if (e.key === "ArrowUp") {
        e.preventDefault();
        setMenuIndex((prev) => Math.max(0, prev - 1));
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        const maxIdx =
          menuView === "scope"
            ? visibleScopeOptions.length - 1
            : visibleValueOptions.length - 1;
        setMenuIndex((prev) => Math.min(maxIdx, prev + 1));
        return;
      }

      if (e.key === "Escape") {
        setMenuView(undefined);
        setCurrentScope(undefined);
        // Emit current text as query
        if (!currentScope && inputText !== params.query) {
          onParamsChange({ ...params, query: inputText });
        }
        return;
      }

      if (e.key === "Enter") {
        e.preventDefault();
        if (menuView === "scope") {
          const opt = visibleScopeOptions[menuIndex];
          if (opt) selectScope(opt.id);
        } else if (menuView === "value") {
          if (
            visibleValueOptions.length > 0 &&
            visibleValueOptions[menuIndex]
          ) {
            selectValue(visibleValueOptions[menuIndex].value);
          } else if (currentScope && inputText.trim()) {
            // Strip the "scope:" prefix if present
            const prefix = `${currentScope}:`;
            const raw = inputText.trim();
            const value = raw.startsWith(prefix)
              ? raw.substring(prefix.length).trim()
              : raw;
            if (value) selectValue(value);
          }
        } else if (onEnter) {
          // No dropdown open — surface the Enter to the host (used by
          // result-view search to advance to the next matching row).
          onEnter();
        }
      }
    },
    [
      inputText,
      focusedTagIndex,
      editableScopes,
      params,
      menuView,
      visibleScopeOptions,
      visibleValueOptions,
      menuIndex,
      currentScope,
      removeScope,
      selectScope,
      selectValue,
      onParamsChange,
      onEnter,
    ]
  );

  // Reset menuIndex when options change
  useEffect(() => {
    setMenuIndex(0);
  }, [menuView]);

  // Render scope tags
  const visibleTags = useMemo(
    () =>
      params.scopes
        .map((scope, originalIndex) => ({ scope, originalIndex }))
        .filter(({ scope }) => !scope.readonly),
    [params.scopes]
  );

  const renderTagValue = useCallback(
    (scope: SearchScope) => {
      const opt = scopeOptions
        .find((o) => o.id === scope.id)
        ?.options?.find((o) => o.value === scope.value);
      if (opt?.render) return opt.render("");
      return scope.value;
    },
    [scopeOptions]
  );

  useEffect(() => {
    if (focusedTagIndex === undefined) return;
    const tag = tagsContainerRef.current?.querySelector<HTMLElement>(
      `[data-search-scope-index="${focusedTagIndex}"]`
    );
    tag?.scrollIntoView({
      block: "nearest",
      inline: "nearest",
    });
  }, [focusedTagIndex]);

  return (
    <div ref={containerRef} className="w-full min-w-0 relative">
      {/* Input container */}
      <div
        className="flex min-w-0 items-center h-9 overflow-hidden border border-control-border rounded-xs bg-background transition-colors dark:bg-dark-bg dark:border-zinc-700"
        onClick={() => inputRef.current?.focus()}
      >
        {/*
         * No hard `max-w-[60%]` here on purpose. The tags container is a
         * flex item with `shrink` and the input sibling already declares
         * `min-w-[120px] flex-1`, so flexbox guarantees the input keeps
         * its minimum typing width and the tags only have to shrink (and
         * scroll horizontally) when their natural size truly exceeds
         * what's left. A 60% cap forced the tags to clip even when the
         * bar had plenty of unused space on the right — visible in the
         * AccessPane where two short scope tags read as "status: 开启
         * × stat…".
         */}
        <div className="flex min-w-0 items-center shrink">
          <div className="shrink-0 pl-2">
            <Filter className="h-4 w-4 text-control-placeholder" />
          </div>

          {/* Scope tags */}
          <div className="relative min-w-0 flex-1">
            <div
              ref={tagsContainerRef}
              className="flex min-w-0 items-center gap-1 overflow-x-auto pl-1 hide-scrollbar"
            >
              {visibleTags.map(({ scope, originalIndex }) => (
                <span
                  key={`${originalIndex}-${scope.id}`}
                  data-search-scope-id={scope.id}
                  data-search-scope-index={originalIndex}
                  className={cn(
                    "inline-flex max-w-[16rem] min-w-0 shrink-0 items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs whitespace-nowrap dark:bg-zinc-700 dark:text-gray-100",
                    focusedTagIndex === originalIndex && "ring-1 ring-accent"
                  )}
                  onClick={(e) => {
                    e.stopPropagation();
                    // Re-enter editing for this scope
                    if (scopeOptions.some((o) => o.id === scope.id)) {
                      selectScope(scope.id);
                    }
                  }}
                >
                  <span className="text-control">{scope.id}:</span>
                  <span className="min-w-0 truncate" title={scope.value}>
                    {renderTagValue(scope)}
                  </span>
                  <button
                    className="ml-0.5 hover:text-error"
                    onClick={(e) => {
                      e.stopPropagation();
                      removeScope(originalIndex);
                    }}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            {tagFade.showStart && (
              <ScrollFade
                edge="left"
                className="from-background to-transparent"
              />
            )}
            {tagFade.showEnd && (
              <ScrollFade
                edge="right"
                className="from-background to-transparent"
              />
            )}
          </div>
        </div>

        {/*
         * Input min-width is conditional on whether any tags are
         * present:
         *   - No tags: reserve 120px so the placeholder text is fully
         *     legible.
         *   - Has tags: the placeholder is hidden anyway (see below),
         *     so we only need room for the cursor — 40px. Without this
         *     the previous fixed `min-w-[120px]` forced the tags
         *     container to shrink even when the bar had visible empty
         *     space, which is what made scope tags clip in panels like
         *     AccessPane.
         */}
        <input
          ref={inputRef}
          className={cn(
            "flex-1 bg-transparent border-none px-2 text-sm text-main placeholder:text-control-placeholder focus:outline-none focus:border-none focus:ring-0 focus:shadow-none dark:text-gray-100 dark:placeholder:text-gray-500",
            visibleTags.length > 0 ? "min-w-[40px]" : "min-w-[120px]"
          )}
          value={inputText}
          placeholder={visibleTags.length > 0 ? "" : placeholder}
          onClick={handleInputClick}
          onChange={(e) => handleInputChange(e.target.value)}
          onKeyDown={handleKeyDown}
        />

        {/* Clear button */}
        {clearable && (
          <button
            className="p-1.5 mr-1 hover:bg-control-bg rounded-full shrink-0"
            onClick={(e) => {
              e.stopPropagation();
              handleClear();
            }}
          >
            <X className="h-3 w-3 text-control-placeholder" />
          </button>
        )}
      </div>

      {/* Dropdown menu */}
      {showMenu && (
        <div
          className={cn(
            "absolute top-[38px] w-full bg-control-bg shadow-xl origin-top-left rounded-sm overflow-hidden",
            LAYER_SURFACE_CLASS
          )}
          onMouseDown={(e) => e.preventDefault()}
        >
          {/* Scope menu */}
          {menuView === "scope" && visibleScopeOptions.length > 0 && (
            <div className="relative">
              <div
                ref={scopeMenuListRef}
                className="max-h-[480px] overflow-auto"
              >
                {visibleScopeOptions.map((option, index) => (
                  <div
                    key={option.id}
                    className={cn(
                      "flex gap-x-2 px-3 py-2 cursor-pointer text-sm items-center",
                      index > 0 && "border-t border-block-border",
                      index === menuIndex && "bg-control-bg-hover/75"
                    )}
                    onMouseEnter={() => setMenuIndex(index)}
                    onClick={() => selectScope(option.id)}
                  >
                    <HighlightLabelText
                      className="text-accent"
                      text={option.id}
                      keyword={scopeKeyword}
                    />
                    {option.description && (
                      <HighlightLabelText
                        className="text-control-light truncate"
                        text={option.description}
                        keyword={scopeKeyword}
                      />
                    )}
                  </div>
                ))}
              </div>
              {scopeMenuFade.showStart && (
                <ScrollFade
                  edge="top"
                  className="from-control-bg to-transparent"
                />
              )}
              {scopeMenuFade.showEnd && (
                <ScrollFade
                  edge="bottom"
                  className="from-control-bg to-transparent"
                />
              )}
            </div>
          )}

          {/* Value menu */}
          {menuView === "value" && currentScopeOption && (
            <div className="flex flex-col overflow-hidden">
              <div className="px-3 py-2">
                <div className="text-sm text-control font-semibold">
                  {currentScopeOption.title}
                </div>
                {currentScopeOption.description && (
                  <div className="text-xs text-control-light">
                    {currentScopeOption.description}
                    {currentScopeOption.onSearch && (
                      <span>. {t("common.type-to-search")}</span>
                    )}
                  </div>
                )}
              </div>
              {visibleValueOptions.length > 0 ? (
                <div className="relative">
                  <div
                    ref={valueMenuListRef}
                    className="max-h-[240px] overflow-auto"
                  >
                    {visibleValueOptions.map((option, index) => (
                      <div
                        key={option.value}
                        className={cn(
                          "h-[38px] flex gap-x-2 px-3 items-center cursor-pointer border-t border-block-border overflow-hidden",
                          index === menuIndex && "bg-control-bg-hover/75"
                        )}
                        onMouseEnter={() => setMenuIndex(index)}
                        onClick={() => selectValue(option.value)}
                      >
                        {option.render && (
                          <span className="min-w-0 truncate text-control text-sm">
                            {option.render(currentValueForScope)}
                          </span>
                        )}
                        {!option.custom && (
                          <span
                            className="min-w-0 flex-1 truncate text-control-light text-sm"
                            title={option.value}
                          >
                            <HighlightLabelText
                              text={option.value}
                              keyword={currentValueForScope}
                            />
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ) : asyncLoading ? (
                <div className="py-4 text-center text-sm text-control-placeholder">
                  …
                </div>
              ) : !(currentScopeOption.options ?? []).length &&
                !isAsyncScope ? (
                <div className="px-3 py-2 text-xs text-control-light border-t border-block-border">
                  Type a value and press Enter
                </div>
              ) : (
                <div className="py-4 text-center text-sm text-control-placeholder">
                  —
                </div>
              )}
              {valueMenuFade.showStart && (
                <ScrollFade
                  edge="top"
                  className="top-[41px] from-control-bg to-transparent"
                />
              )}
              {valueMenuFade.showEnd && (
                <ScrollFade
                  edge="bottom"
                  className="from-control-bg to-transparent"
                />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
