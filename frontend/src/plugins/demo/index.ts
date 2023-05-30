import axios from "axios";
import { isNull } from "lodash-es";
import { createApp } from "vue";
import { isDev } from "@/utils";
import { router } from "@/router";
import * as storage from "./storage";
import { piniaInstance } from "./store";
import { initLocationListenerForDemo } from "./listener";
import DemoWrapper from "./components/DemoWrapper.vue";
import { removeHint } from "./hint";
import { removeGuideDialog } from "./guide";
import { waitBodyLoaded } from "./utils";

const initDemo = async (demoName: string) => {
  await waitBodyLoaded();
  // mount the demo vue app
  const demoAppContainer = document.createElement("div");
  document.body.appendChild(demoAppContainer);
  const app = createApp(DemoWrapper, {
    demoName,
  });
  app.use(router).use(piniaInstance).mount(demoAppContainer);

  storage.set({
    demo: {
      name: demoName,
    },
  });
  // TODO(steven): refactor the pure js element into vue
  await initLocationListenerForDemo();

  // inject segment script
  const scriptElement = document.createElement("script");
  scriptElement.innerHTML = `
  !function(){var analytics=window.analytics=window.analytics||[];if(!analytics.initialize)if(analytics.invoked)window.console&&console.error&&console.error("Segment snippet included twice.");else{analytics.invoked=!0;analytics.methods=["trackSubmit","trackClick","trackLink","trackForm","pageview","identify","reset","group","track","ready","alias","debug","page","once","off","on","addSourceMiddleware","addIntegrationMiddleware","setAnonymousId","addDestinationMiddleware"];analytics.factory=function(e){return function(){var t=Array.prototype.slice.call(arguments);t.unshift(e);analytics.push(t);return analytics}};for(var e=0;e<analytics.methods.length;e++){var key=analytics.methods[e];analytics[key]=analytics.factory(key)}analytics.load=function(key,e){var t=document.createElement("script");t.type="text/javascript";t.async=!0;t.src="https://cdn.segment.com/analytics.js/v1/" + key + "/analytics.min.js";var n=document.getElementsByTagName("script")[0];n.parentNode.insertBefore(t,n);analytics._loadOptions=e};analytics._writeKey="CVXXNXv3EzfQPYqHoYvlDDDOXmKa9XOj";;analytics.SNIPPET_VERSION="4.15.3";
    analytics.load("CVXXNXv3EzfQPYqHoYvlDDDOXmKa9XOj");
    analytics.page();
  }}();`;
  document.body.appendChild(scriptElement);
};

const mountDemoApp = async () => {
  const serverInfo = (
    await axios.get<{
      demoName: string;
      version: string;
    }>(`/v1/actuator/info`)
  ).data;

  // only show feature demo if it's not the default demo.
  if (serverInfo.demoName && serverInfo.demoName != "default") {
    const demoName = serverInfo.demoName;
    if (demoName) {
      initDemo(demoName);
    }
  }

  // only using in dev mode
  if (isDev()) {
    const params = new URLSearchParams(window.location.search);
    const demoName = params.get("demo") ?? "";
    if (demoName) {
      initDemo(demoName);
    }

    const clearDemo = params.get("cleardemo");
    if (!isNull(clearDemo)) {
      storage.remove(["demo", "guide"]);
    }
  }
};

export const removeDemo = () => {
  storage.remove(["demo", "guide"]);
  removeHint();
  removeGuideDialog();
};

export default mountDemoApp;
