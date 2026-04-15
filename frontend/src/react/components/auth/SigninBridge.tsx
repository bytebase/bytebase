import { NButton, NConfigProvider } from "naive-ui";
import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import { dateLang, generalLang, themeOverrides } from "@/../naive-ui.config";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia, useAuthStore } from "@/store";
import Signin from "@/views/auth/Signin.vue";

export function SigninBridge({ currentPath }: { currentPath: string }) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const app = createApp({
      render() {
        return h(
          NConfigProvider,
          {
            locale: generalLang.value,
            dateLocale: dateLang.value,
            themeOverrides: themeOverrides.value,
          },
          {
            default: () =>
              h(
                Signin as never,
                {
                  redirect: false,
                  redirectUrl: currentPath,
                  allowSignup: false,
                },
                {
                  footer: () =>
                    h(
                      NButton,
                      {
                        quaternary: true,
                        size: "small",
                        onClick: () => useAuthStore().logout(),
                      },
                      () => i18n.global.t("common.logout")
                    ),
                }
              ),
          }
        );
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [currentPath]);

  return <div ref={containerRef} />;
}
