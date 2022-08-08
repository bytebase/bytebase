import axios from "axios";
import { isNull } from "lodash-es";
import { createApp } from "vue";
import { router } from "@/router";
import * as storage from "./storage";
import { piniaInstance } from "./store";
import { initLocationListenerForDemo } from "./listener";
import DemoWrapper from "./components/DemoWrapper.vue";

const invalidDemoNameList = ["dev", "prod"];

// initial guide listeners when window loaded
window.addEventListener(
  "load",
  async () => {
    const serverInfo = (
      await axios.get<{
        demo: boolean;
        demoName: string;
      }>(`/api/actuator/info`)
    ).data;

    // only show demo in feature demo mode.
    if (
      serverInfo.demo &&
      serverInfo.demoName &&
      !invalidDemoNameList.includes(serverInfo.demoName)
    ) {
      const demoName = serverInfo.demoName;
      if (demoName) {
        // mount the demo vue app
        const demoDrawerContainer = document.createElement("div");
        document.body.appendChild(demoDrawerContainer);
        const app = createApp(DemoWrapper, {
          demoName,
        });
        app.use(router).use(piniaInstance).mount(demoDrawerContainer);

        // TODO(steven): refactor the pure js element into vue
        await initLocationListenerForDemo();
      }
    }

    // If there is a `cleardemo` flag in the url, then clear the demo storage.
    const params = new URLSearchParams(window.location.search);
    const clearDemo = params.get("cleardemo");
    if (!isNull(clearDemo)) {
      storage.remove(["demo", "guide"]);
    }
  },
  {
    once: true,
  }
);

export default null;
