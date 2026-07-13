import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useMemo,
  useRef,
} from "react";
import { useSearchParams } from "react-router-dom";
import {
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  type SearchParams,
} from "@/utils";

const identity = (value: string): string => value;

export const createAdvancedSearchParser =
  (allowedScopes: readonly string[]) =>
  (value: string): SearchParams => {
    const params = buildSearchParamsBySearchText(value);
    return {
      query: params.query,
      scopes: params.scopes.filter((scope) => allowedScopes.includes(scope.id)),
    };
  };

export const serializeAdvancedSearch = (value: SearchParams): string =>
  buildSearchTextBySearchParams(value);

/** Maps the legacy `?state=archived|all` URL format onto search scopes. */
export const legacyStateSearchParams = (state: unknown): SearchParams => {
  if (state === "archived" || state === "all") {
    return {
      query: "",
      scopes: [
        { id: "state", value: state === "archived" ? "DELETED" : "ALL" },
      ],
    };
  }
  return { query: "", scopes: [] };
};

export type URLSearchParamOptions<T> = {
  param: string;
  parse: (value: string) => T;
  serialize: (value: T) => string;
  defaultValue: T;
};

/**
 * Keeps state in the current URL instead of mirroring it through an effect.
 * React Router remains the source of truth, so browser Back/Forward updates
 * the value while the route component stays mounted.
 *
 * A value that serializes identically to `defaultValue` is elided from the
 * URL. Any other value is kept as an explicit param — for pages whose default
 * is not empty, `?q=` is how an empty/All selection survives a remount.
 */
export function useURLSearchParam(options: {
  param: string;
  defaultValue: string;
}): [string, Dispatch<SetStateAction<string>>];
export function useURLSearchParam<T>(
  options: URLSearchParamOptions<T>
): [T, Dispatch<SetStateAction<T>>];
export function useURLSearchParam<T>({
  param,
  parse = identity as unknown as (value: string) => T,
  serialize = identity as unknown as (value: T) => string,
  defaultValue,
}: Partial<URLSearchParamOptions<T>> & {
  param: string;
  defaultValue: T;
}): [T, Dispatch<SetStateAction<T>>] {
  const [urlSearchParams, setURLSearchParams] = useSearchParams();
  const hasParam = urlSearchParams.has(param);
  const rawValue = urlSearchParams.get(param) ?? "";

  // Deliberately independent of `defaultValue`: an async default (e.g. one
  // derived from the resolving current user) must not recompute — and change
  // the identity of — a value that came from the URL.
  const parsedValue = useMemo(
    () => (hasParam ? parse(rawValue) : undefined),
    [hasParam, parse, rawValue]
  );
  const value = hasParam ? (parsedValue as T) : defaultValue;

  const urlSearchParamsRef = useRef(urlSearchParams);
  urlSearchParamsRef.current = urlSearchParams;

  const setValue = useCallback<Dispatch<SetStateAction<T>>>(
    (action) => {
      const apply = (current: URLSearchParams): URLSearchParams => {
        const has = current.has(param);
        const raw = current.get(param) ?? "";
        const next =
          typeof action === "function"
            ? (action as (previous: T) => T)(has ? parse(raw) : defaultValue)
            : action;
        const nextRaw = serialize(next);
        if (nextRaw === serialize(defaultValue)) {
          if (!has) return current;
          const nextParams = new URLSearchParams(current);
          nextParams.delete(param);
          return nextParams;
        }
        if (has && raw === nextRaw) return current;
        const nextParams = new URLSearchParams(current);
        nextParams.set(param, nextRaw);
        return nextParams;
      };
      // Skip the replace navigation when the URL would not change.
      if (apply(urlSearchParamsRef.current) === urlSearchParamsRef.current) {
        return;
      }
      setURLSearchParams(apply, { replace: true });
    },
    [defaultValue, param, parse, serialize, setURLSearchParams]
  );

  return [value, setValue];
}
