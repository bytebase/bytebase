(() => {
  if (typeof WeakRef === "undefined") {
    class WeakRefPolyfill<T extends WeakKey> {
      private targetMap = new Map<WeakRefPolyfill<T>, T | undefined>();
      private target: T;

      readonly [Symbol.toStringTag] = "WeakRef";

      constructor(target: T) {
        if (typeof target !== "object" || target === null) {
          throw new TypeError("WeakRef target must be an object.");
        }
        this.target = target;
        this.targetMap.set(this, target);
      }

      deref(): T | undefined {
        const target = this.targetMap.get(this);
        return target ?? undefined;
      }
    }

    // Use a specific type instead of any
    (
      globalThis as typeof globalThis & { WeakRef: typeof WeakRefPolyfill }
    ).WeakRef = WeakRefPolyfill;
  }
})();
