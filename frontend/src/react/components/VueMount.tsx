import { useEffect, useRef } from "react";
import type { Component } from "vue";
import { createApp, h, reactive } from "vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { cn } from "@/react/lib/utils";
import { pinia } from "@/store";

interface VueMountProps<P extends Record<string, unknown>> {
  /**
   * Vue component to mount. Component identity must be stable across renders;
   * passing a fresh inline definition will tear down and re-create the inner
   * Vue app every render. Module-level imports satisfy this naturally.
   */
  component: Component;
  /**
   * Props forwarded to the Vue component. Treated reactively — assigning new
   * keys/values updates the running Vue app in place via Vue's reactivity,
   * no remount. The object reference itself doesn't need to be stable; only
   * the keys/values are diffed.
   */
  props?: P;
  className?: string;
}

/**
 * Reverse mount bridge: hosts a Vue 3 component subtree inside a React tree.
 * Mirrors `ReactPageMount.vue`'s lifecycle in reverse — used by the SQL
 * Editor host shells (post-Stage-21) to keep the still-Vue AI plugin
 * (`AIChatToSQL`, `ChatPanel`, etc.) running without porting it yet.
 *
 * The inner Vue app installs the same singleton plugins as the outer Vue
 * app (Pinia, i18n, NaiveUI) so Pinia stores, translations, and Naive UI
 * services are shared — no "two sources of truth" surprises.
 */
export function VueMount<P extends Record<string, unknown>>({
  component,
  props,
  className,
}: VueMountProps<P>) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const reactivePropsRef = useRef<Record<string, unknown> | null>(null);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const reactiveProps = reactive<Record<string, unknown>>({
      ...(props ?? {}),
    });
    reactivePropsRef.current = reactiveProps;

    const app = createApp({
      render: () => h(component, reactiveProps),
    });
    app.use(pinia);
    app.use(i18n);
    app.use(NaiveUI);
    app.mount(el);

    return () => {
      reactivePropsRef.current = null;
      app.unmount();
    };
    // Re-create the Vue app only when component identity changes; prop
    // updates flow through the reactive proxy in the next effect.
  }, [component]);

  // Sync prop changes into the live Vue app without remounting.
  useEffect(() => {
    const target = reactivePropsRef.current;
    if (!target) return;
    const next = props ?? {};
    // Add/overwrite keys.
    for (const [k, v] of Object.entries(next)) {
      target[k] = v;
    }
    // Drop keys removed from the new props snapshot.
    for (const k of Object.keys(target)) {
      if (!(k in next)) {
        delete target[k];
      }
    }
  }, [props]);

  return <div ref={containerRef} className={cn(className)} />;
}
