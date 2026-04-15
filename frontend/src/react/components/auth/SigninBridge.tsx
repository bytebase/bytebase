import { NButton, NConfigProvider } from "naive-ui";
import { useEffect, useRef } from "react";
import { createApp, h, type Ref as VueRef, ref as vueRef } from "vue";
import { dateLang, generalLang, themeOverrides } from "@/../naive-ui.config";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia, useAuthStore } from "@/store";
import Signin from "@/views/auth/Signin.vue";

export function SigninBridge({ currentPath }: { currentPath: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const redirectUrlRef = useRef<VueRef<string> | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const currentPathState = vueRef(currentPath);
    redirectUrlRef.current = currentPathState;

    const renderSignin = () =>
      h(
        Signin as never,
        {
          redirect: false,
          redirectUrl: currentPathState.value,
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
      );

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
                "div",
                {
                  class:
                    "bg-white shadow-lg rounded-md py-3 flex pointer-events-auto flex-col gap-3",
                  style: {
                    maxWidth: "calc(100vw - 80px)",
                    maxHeight: "calc(100vh - 80px)",
                  },
                },
                [
                  h(
                    "div",
                    {
                      class: "px-4 max-h-screen overflow-auto w-full h-full",
                    },
                    [
                      h(
                        "div",
                        {
                          class:
                            "flex items-center w-auto md:min-w-96 max-w-full h-auto md:py-4",
                        },
                        [
                          h(
                            "div",
                            {
                              class:
                                "flex flex-col justify-center items-center flex-1 gap-y-2",
                            },
                            [renderSignin()]
                          ),
                        ]
                      ),
                    ]
                  ),
                ]
              ),
          }
        );
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      redirectUrlRef.current = null;
      app.unmount();
    };
  }, []);

  useEffect(() => {
    if (!redirectUrlRef.current) return;
    redirectUrlRef.current.value = currentPath;
  }, [currentPath]);

  return <div ref={containerRef} />;
}
