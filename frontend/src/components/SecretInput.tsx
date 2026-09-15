import {
  type AriaAttributes,
  type ChangeEvent,
  createContext,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { FormControlRow } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

const SecretEditingContext = createContext<{
  register?: () => () => void;
  pending: boolean;
  resetKey?: string | number;
}>({ pending: false });

/** Coordinates unfinished secret edits with the enclosing form's submit actions. */
export function SecretInputProvider({
  children,
  resetKey,
}: {
  children: ReactNode;
  resetKey?: string | number;
}) {
  const [editors, setEditors] = useState<Set<symbol>>(() => new Set());
  const register = useCallback(() => {
    const id = Symbol();
    setEditors((current) => new Set(current).add(id));
    return () =>
      setEditors((current) => {
        const next = new Set(current);
        next.delete(id);
        return next;
      });
  }, []);
  const value = useMemo(
    () => ({ register, pending: editors.size > 0, resetKey }),
    [register, editors.size, resetKey]
  );
  return (
    <SecretEditingContext.Provider value={value}>
      {children}
    </SecretEditingContext.Provider>
  );
}

export const useHasPendingSecretEdits = () =>
  useContext(SecretEditingContext).pending;

export interface SecretInputProps extends AriaAttributes {
  id?: string;
  className?: string;
  /** Undefined preserves the stored value; an empty string explicitly clears it. */
  value: string | undefined;
  onValueChange: (value: string) => void;
  isCreating?: boolean;
  disabled?: boolean;
  multiline?: boolean;
  allowEmpty?: boolean;
  placeholder?: string;
  /** Change when the edited resource changes or its draft is discarded. */
  resetKey?: string | number;
}

export function SecretInput(props: SecretInputProps) {
  const context = useContext(SecretEditingContext);
  return (
    <SecretInputControl
      key={`${context.resetKey ?? ""}:${props.resetKey ?? ""}`}
      {...props}
    />
  );
}

function SecretInputControl({
  value,
  onValueChange,
  isCreating = false,
  disabled = false,
  multiline = false,
  allowEmpty = true,
  placeholder,
  className,
  resetKey: _resetKey,
  id,
  ...aria
}: SecretInputProps) {
  const { t } = useTranslation();
  const { register } = useContext(SecretEditingContext);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);
  const editButtonRef = useRef<HTMLButtonElement>(null);
  const fileReadRef = useRef(0);
  const wasEditingRef = useRef(false);
  const hintId = useId();
  const active = isCreating || editing;

  useEffect(() => {
    if (editing && !disabled) return register?.();
  }, [editing, disabled, register]);
  useEffect(() => {
    if (editing) inputRef.current?.focus();
    else if (wasEditingRef.current) editButtonRef.current?.focus();
    wasEditingRef.current = editing;
    return () => {
      fileReadRef.current++;
    };
  }, [editing, disabled]);

  const finish = (commit: boolean) => {
    if (disabled || (commit && !allowEmpty && draft.length === 0)) return;
    if (commit) onValueChange(draft);
    fileReadRef.current++;
    setDraft("");
    setEditing(false);
  };
  const change = (next: string) => {
    if (isCreating) onValueChange(next);
    else setDraft(next);
  };
  const onKeyDown = (
    event: KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    if (!editing || event.nativeEvent.isComposing) return;
    if (event.key === "Escape" || (!multiline && event.key === "Enter")) {
      event.preventDefault();
      event.stopPropagation();
      finish(event.key === "Enter");
    }
  };
  const onDrop = (event: DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
    if (disabled || !active) return;
    const file = event.dataTransfer.files[0];
    if (!file) return;
    const request = ++fileReadRef.current;
    const reader = new FileReader();
    reader.onload = () => {
      if (request === fileReadRef.current && typeof reader.result === "string")
        change(reader.result);
    };
    reader.readAsText(file);
  };
  const inputProps = {
    ...aria,
    id,
    disabled: disabled || !active,
    value: active ? (isCreating ? (value ?? "") : draft) : "",
    placeholder: active
      ? placeholder
      : value === undefined
        ? t("common.secret-input.hidden")
        : value === ""
          ? t("common.secret-input.empty")
          : t("common.secret-input.updated"),
    "aria-describedby":
      [aria["aria-describedby"], editing ? hintId : undefined]
        .filter(Boolean)
        .join(" ") || undefined,
    autoComplete: "new-password",
    spellCheck: false,
    onKeyDown,
    onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      change(event.target.value),
    ref: (element: HTMLInputElement | HTMLTextAreaElement | null) => {
      inputRef.current = element;
    },
  };

  return (
    <div className={cn("flex min-w-0 flex-col gap-2", className)}>
      <FormControlRow className="flex-wrap items-start gap-y-2">
        <div className="min-w-0 flex-1">
          {multiline && active ? (
            <Textarea
              {...inputProps}
              onDragOver={(event) => event.preventDefault()}
              onDrop={onDrop}
            />
          ) : (
            <Input {...inputProps} type={active ? "password" : "text"} />
          )}
        </div>
        {!isCreating && (
          <div className="flex shrink-0 gap-2">
            {editing ? (
              <>
                <Button
                  type="button"
                  appearance="outline"
                  disabled={disabled}
                  onClick={() => finish(false)}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  type="button"
                  appearance="secondary"
                  disabled={disabled || (!allowEmpty && draft.length === 0)}
                  onClick={() => finish(true)}
                >
                  {t("common.done")}
                </Button>
              </>
            ) : (
              <Button
                ref={editButtonRef}
                type="button"
                appearance="outline"
                disabled={disabled}
                onClick={() => {
                  setDraft(value ?? "");
                  setEditing(true);
                }}
              >
                {t("common.edit")}
              </Button>
            )}
          </div>
        )}
      </FormControlRow>
      {editing && (
        <p id={hintId} className="text-xs text-control-light">
          {allowEmpty
            ? t("common.secret-input.edit-hint")
            : t("common.secret-input.required-hint")}
        </p>
      )}
    </div>
  );
}
