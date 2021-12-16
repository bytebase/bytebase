import { App } from "vue";

import { Splitpanes, Pane } from "splitpanes";
import "splitpanes/dist/splitpanes.css";

export default function install(app: App): void {
  app.component("Splitpanes", Splitpanes);
  app.component("Pane", Pane);
}
