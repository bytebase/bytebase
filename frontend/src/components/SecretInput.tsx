import { Eye, EyeOff, X } from "lucide-react";
import {
  type AriaAttributes,
  type ChangeEvent,
  createContext,
  type DragEvent,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

const SecretEditingContext = createContext<{
  register?: () => () => void;
  invalid: boolean;
  resetKey?: string | number;
}>({ invalid: false });

/** Coordinates invalid secret replacements with the enclosing form. */
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
    () => ({ register, invalid: editors.size > 0, resetKey }),
    [register, editors.size, resetKey]
  );
  return (
    <SecretEditingContext.Provider value={value}>
      {children}
    </SecretEditingContext.Provider>
  );
}

export const useHasInvalidSecretInputs = () =>
  useContext(SecretEditingContext).invalid;

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
  const [showPassword, setShowPassword] = useState(false);
  const fileReadRef = useRef(0);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);
  const unchanged = !isCreating && value === undefined;
  const showReveal = !unchanged && !multiline;
  const showClear = allowEmpty && (unchanged || !!value);
  const inputPadding =
    showReveal && showClear
      ? "pr-20"
      : showReveal || showClear
        ? "pr-10"
        : undefined;

  useEffect(() => {
    if (!disabled && !allowEmpty && value === "") return register?.();
  }, [disabled, allowEmpty, value, register]);
  useEffect(() => {
    setShowPassword(false);
    return () => {
      fileReadRef.current++;
    };
  }, [disabled, unchanged]);

  const change = (next: string) => {
    if (disabled) return;
    fileReadRef.current++;
    onValueChange(next);
  };
  const onDrop = (event: DragEvent<HTMLTextAreaElement>) => {
    event.preventDefault();
    if (disabled) return;
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
    disabled,
    value: value ?? "",
    placeholder: unchanged ? t("common.secret-input.stored") : placeholder,
    ref: (element: HTMLInputElement | HTMLTextAreaElement | null) => {
      inputRef.current = element;
    },
    required: !allowEmpty && !unchanged,
    autoComplete: "new-password",
    spellCheck: false,
    onChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      change(event.target.value),
  };

  return (
    <div className={cn("relative min-w-0", className)}>
      {multiline ? (
        <Textarea
          {...inputProps}
          className={inputPadding}
          onDragOver={(event) => event.preventDefault()}
          onDrop={onDrop}
        />
      ) : (
        <Input
          {...inputProps}
          className={inputPadding}
          type={!unchanged && showPassword ? "text" : "password"}
        />
      )}
      {(showReveal || showClear) && (
        <div className="absolute right-1 top-1 flex">
          {showReveal && (
            <Button
              type="button"
              appearance="secondary"
              size="sm"
              disabled={disabled}
              aria-label={t("common.toggle-password-visibility")}
              aria-pressed={showPassword}
              onClick={() => setShowPassword((visible) => !visible)}
            >
              {showPassword ? <Eye size={16} /> : <EyeOff size={16} />}
            </Button>
          )}
          {showClear && (
            <Button
              type="button"
              appearance="secondary"
              size="sm"
              disabled={disabled}
              aria-label={t("common.clear")}
              title={t("common.clear")}
              onClick={() => {
                change("");
                setShowPassword(false);
                inputRef.current?.focus();
              }}
            >
              <X size={16} />
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
