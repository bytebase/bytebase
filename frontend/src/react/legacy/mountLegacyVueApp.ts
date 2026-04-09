import {
  NConfigProvider,
  NDialogProvider,
  NNotificationProvider,
} from "naive-ui";
import {
  createApp,
  defineComponent,
  h,
  type Plugin,
  type VNodeChild,
} from "vue";
import OverlayStackManager from "@/components/misc/OverlayStackManager.vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";

const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

export const LegacyVueBridgeProviders = defineComponent({
  name: "LegacyVueBridgeProviders",
  setup(_, { slots }) {
    return () =>
      h(NConfigProvider, null, {
        default: () =>
          h(
            NNotificationProvider,
            {
              max: MAX_NOTIFICATION_DISPLAY_COUNT,
              placement: "bottom-right",
            },
            {
              default: () =>
                h(NDialogProvider, null, {
                  default: () =>
                    h(OverlayStackManager, null, {
                      default: () => slots.default?.(),
                    }),
                }),
            }
          ),
      });
  },
});

export const createLegacyVueApp = ({
  extraPlugins = [],
  render,
}: {
  extraPlugins?: Plugin[];
  render: () => VNodeChild;
}) => {
  const app = createApp({
    render() {
      return h(LegacyVueBridgeProviders, null, {
        default: render,
      });
    },
  });

  app.use(router).use(pinia).use(i18n).use(NaiveUI);
  for (const plugin of extraPlugins) {
    app.use(plugin);
  }

  return app;
};
