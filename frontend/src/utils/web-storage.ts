import {
  customStorageEventName,
  defaultWindow,
  getSSRHandler,
  type MaybeRefOrGetter,
  pausableWatch,
  type RemovableRef,
  type StorageEventLike,
  type StorageLike,
  StorageSerializers,
  toValue,
  tryOnMounted,
  type UseStorageOptions,
  useEventListener,
} from "@vueuse/core";
import { nextTick, type Ref, ref, shallowRef, watch } from "vue";

export class WebStorageHelper {
  storage: Storage;
  keyPrefix: string;

  constructor(keyPrefix: string, storage = localStorage) {
    this.keyPrefix = keyPrefix;
    this.storage = storage;
  }

  save<T>(key: string, value: T) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      const json = JSON.stringify(value);
      localStorage.setItem(fullKey, json);
    } catch {
      // nothing
    }
  }

  load<T>(key: string, fallbackValue: T) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      const json = localStorage.getItem(fullKey) || "";
      return JSON.parse(json) as T;
    } catch {
      return fallbackValue;
    }
  }

  remove(key: string) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      localStorage.removeItem(fullKey);
    } catch {
      // nothing
    }
  }

  keys(): string[] {
    const { length } = localStorage;
    const keys: string[] = [];
    for (let i = 0; i < length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith(this.keyPrefix)) {
        keys.push(key);
      }
    }
    return keys;
  }

  clear() {
    const keys = this.keys();
    keys.forEach((key) => {
      localStorage.removeItem(key);
    });
  }
}

/**
 * see: https://github.com/vueuse/vueuse/blob/main/packages/core/useStorage/index.ts
 */
export const useDynamicLocalStorage = <
  T extends string | number | boolean | object | null,
>(
  key: Ref<string>,
  defaults: MaybeRefOrGetter<T>,
  storage: StorageLike | undefined = window.localStorage,
  options: UseStorageOptions<T> = {}
) => {
  const {
    flush = "pre",
    deep = true,
    listenToStorageChanges = true,
    writeDefaults = true,
    mergeDefaults = false,
    shallow,
    window = defaultWindow,
    eventFilter,
    onError = (e) => {
      console.error(e);
    },
    initOnMounted,
  } = options;

  const data = (shallow ? shallowRef : ref)(
    typeof defaults === "function" ? defaults() : defaults
  ) as RemovableRef<T>;

  if (!storage) {
    try {
      storage = getSSRHandler(
        "getDefaultStorage",
        () => defaultWindow?.localStorage
      )();
    } catch (e) {
      onError(e);
    }
  }

  if (!storage) return data;

  const rawInit: T = toValue(defaults);
  const type = guessSerializerType<T>(rawInit);
  const serializer = options.serializer ?? StorageSerializers[type];

  const { pause: pauseWatch, resume: resumeWatch } = pausableWatch(
    data,
    () => write(data.value),
    { flush, deep, eventFilter }
  );

  if (window && listenToStorageChanges) {
    tryOnMounted(() => {
      // this should be fine since we are in a mounted hook
      useEventListener(window, "storage", update);
      useEventListener(window, customStorageEventName, updateFromCustomEvent);
      if (initOnMounted) update();
    });
  }

  // avoid reading immediately to avoid hydration mismatch when doing SSR
  if (!initOnMounted) update();

  function dispatchWriteEvent(
    oldValue: string | null,
    newValue: string | null
  ) {
    // send custom event to communicate within same page
    // importantly this should _not_ be a StorageEvent since those cannot
    // be constructed with a non-built-in storage area
    if (window) {
      window.dispatchEvent(
        new CustomEvent<StorageEventLike>(customStorageEventName, {
          detail: {
            key: key.value,
            oldValue,
            newValue,
            storageArea: storage,
          },
        })
      );
    }
  }

  function write(v: unknown) {
    try {
      const oldValue = storage.getItem(key.value);

      if (v == null) {
        dispatchWriteEvent(oldValue, null);
        storage.removeItem(key.value);
      } else {
        // v is known to be T at this point since it comes from data.value
        const serialized = serializer.write(v as T);
        if (oldValue !== serialized) {
          storage.setItem(key.value, serialized);
          dispatchWriteEvent(oldValue, serialized);
        }
      }
    } catch (e) {
      onError(e);
    }
  }

  function read(event?: StorageEventLike) {
    const rawValue = event ? event.newValue : storage.getItem(key.value);

    if (rawValue == null) {
      if (writeDefaults && rawInit != null)
        storage.setItem(key.value, serializer.write(rawInit));
      return rawInit;
    } else if (!event && mergeDefaults) {
      const value = serializer.read(rawValue);
      if (typeof mergeDefaults === "function") {
        return mergeDefaults(value, rawInit);
      } else if (type === "object" && !Array.isArray(value)) {
        // We know T extends object here since type === "object"
        return { ...(rawInit as Record<string, unknown>), ...value } as T;
      }
      return value;
    } else if (typeof rawValue !== "string") {
      return rawValue;
    } else {
      return serializer.read(rawValue);
    }
  }

  function update(event?: StorageEventLike) {
    if (event && event.storageArea !== storage) return;

    if (event && event.key == null) {
      data.value = rawInit;
      return;
    }

    if (event && event.key !== key.value) return;

    pauseWatch();
    try {
      if (event?.newValue !== serializer.write(data.value))
        data.value = read(event);
    } catch (e) {
      onError(e);
    } finally {
      // use nextTick to avoid infinite loop
      if (event) nextTick(resumeWatch);
      else resumeWatch();
    }
  }

  function updateFromCustomEvent(event: CustomEvent<StorageEventLike>) {
    update(event.detail);
  }

  watch(key, () => update());

  return data;
};

function guessSerializerType<
  T extends string | number | boolean | object | null,
>(rawInit: T) {
  return rawInit == null
    ? "any"
    : rawInit instanceof Set
      ? "set"
      : rawInit instanceof Map
        ? "map"
        : rawInit instanceof Date
          ? "date"
          : typeof rawInit === "boolean"
            ? "boolean"
            : typeof rawInit === "string"
              ? "string"
              : typeof rawInit === "object"
                ? "object"
                : !Number.isNaN(rawInit)
                  ? "number"
                  : "any";
}
