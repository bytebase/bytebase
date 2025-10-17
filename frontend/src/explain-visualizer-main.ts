import "bootstrap/dist/css/bootstrap.css";
import { createApp } from "vue";
import ExplainVisualizerApp from "./ExplainVisualizerApp.vue";

// Redefine global using globalThis
(globalThis as any).global = globalThis;

const app = createApp(ExplainVisualizerApp);

app.mount("body");
