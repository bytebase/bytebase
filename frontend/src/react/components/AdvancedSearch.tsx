import { Filter, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  render?: () => React.ReactNode;
  custom?: boolean;
}

export interface ScopeOption {
  id: string;
  title: string;
  description?: string;
  options?: ValueOption[];
  allowMultiple?: boolean;
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
}

export function AdvancedSearch({
  params,
  scopeOptions = [],
  placeholder,
  onParamsChange,
}: AdvancedSearchProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const [inputText, setInputText] = useState(params.query);
  const [menuView, setMenuView] = useState<"scope" | "value" | undefined>();
  const [currentScope, setCurrentScope] = useState<string | undefined>();
  const [menuIndex, setMenuIndex] = useState(0);
  const [focusedTagIndex, setFocusedTagIndex] = useState<number | undefined>();

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

  // Visible scope options (filtered by typing)
  const visibleScopeOptions = useMemo(() => {
    if (currentScopeOption) return [currentScopeOption];
    const keyword = inputText.trim().replace(/:.*$/, "").toLowerCase();
    if (!keyword) return availableScopeOptions;
    return availableScopeOptions.filter(
      (opt) =>
        opt.id.toLowerCase().includes(keyword) ||
        opt.title.toLowerCase().includes(keyword)
    );
  }, [currentScopeOption, inputText, availableScopeOptions]);

  // Visible value options (filtered, de-duped)
  const visibleValueOptions = useMemo(() => {
    if (!currentScope || !currentScopeOption) return [];
    const selectedValues = new Set(
      params.scopes.filter((s) => s.id === currentScope).map((s) => s.value)
    );
    let options = (currentScopeOption.options ?? []).filter(
      (opt) => !selectedValues.has(opt.value)
    );
    const keyword = currentValueForScope.trim().toLowerCase();
    if (keyword) {
      options = options.filter(
        (opt) =>
          opt.value.toLowerCase().includes(keyword) ||
          opt.keywords.some((k) => k.toLowerCase().includes(keyword))
      );
    }
    return options;
  }, [currentScope, currentScopeOption, params.scopes, currentValueForScope]);

  const showMenu = useMemo(() => {
    if (menuView === "scope") return visibleScopeOptions.length > 0;
    if (menuView === "value") return true;
    return false;
  }, [menuView, visibleScopeOptions]);

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
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
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
            selectValue(inputText.trim());
          }
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
      if (opt?.render) return opt.render();
      return <span>{scope.value}</span>;
    },
    [scopeOptions]
  );

  return (
    <div ref={containerRef} className="w-full min-w-0 relative">
      {/* Input container */}
      <div
        className="flex items-center h-9 border border-gray-300 rounded-xs bg-white transition-colors"
        onClick={() => inputRef.current?.focus()}
      >
        {/* Prefix: filter icon + tags */}
        <div className="flex items-center gap-x-2 pl-2 shrink-0">
          <Filter className="w-4 h-4 text-control-placeholder shrink-0" />
        </div>

        {/* Scope tags */}
        <div className="flex items-center gap-1 overflow-x-auto pl-1 shrink-0 hide-scrollbar">
          {visibleTags.map(({ scope, originalIndex }) => (
            <span
              key={`${originalIndex}-${scope.id}`}
              className={cn(
                "inline-flex items-center gap-1 rounded-xs bg-gray-100 px-1.5 py-0.5 text-xs whitespace-nowrap shrink-0",
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
              {renderTagValue(scope)}
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

        {/* Input */}
        <input
          ref={inputRef}
          className="flex-1 min-w-[120px] bg-transparent border-none px-2 text-sm text-main placeholder:text-control-placeholder focus:outline-none focus:border-none focus:ring-0 focus:shadow-none"
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
          className="absolute top-[38px] w-full bg-gray-100 shadow-xl origin-top-left rounded-xs overflow-hidden z-50"
          onMouseDown={(e) => e.preventDefault()}
        >
          {/* Scope menu */}
          {menuView === "scope" && visibleScopeOptions.length > 0 && (
            <div className="max-h-[480px] overflow-auto">
              {visibleScopeOptions.map((option, index) => (
                <div
                  key={option.id}
                  className={cn(
                    "flex gap-x-2 px-3 py-2 cursor-pointer text-sm items-center",
                    index > 0 && "border-t border-block-border",
                    index === menuIndex && "bg-gray-200/75"
                  )}
                  onMouseEnter={() => setMenuIndex(index)}
                  onClick={() => selectScope(option.id)}
                >
                  <span className="text-accent">{option.id}</span>
                  {option.description && (
                    <span className="text-control-light truncate">
                      {option.description}
                    </span>
                  )}
                </div>
              ))}
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
                  </div>
                )}
              </div>
              {visibleValueOptions.length > 0 ? (
                <div className="max-h-[240px] overflow-auto">
                  {visibleValueOptions.map((option, index) => (
                    <div
                      key={option.value}
                      className={cn(
                        "h-[38px] flex gap-x-2 px-3 items-center cursor-pointer border-t border-block-border overflow-hidden",
                        index === menuIndex && "bg-gray-200/75"
                      )}
                      onMouseEnter={() => setMenuIndex(index)}
                      onClick={() => selectValue(option.value)}
                    >
                      {option.render && (
                        <span className="text-control text-sm">
                          {option.render()}
                        </span>
                      )}
                      {!option.custom && (
                        <span className="text-control-light text-sm">
                          {option.value}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ) : !(currentScopeOption.options ?? []).length ? (
                <div className="px-3 py-2 text-xs text-control-light border-t border-block-border">
                  Type a value and press Enter
                </div>
              ) : (
                <div className="py-4 text-center text-sm text-control-placeholder">
                  —
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
