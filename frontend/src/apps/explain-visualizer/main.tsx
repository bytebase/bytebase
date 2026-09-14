import "@/assets/css/tailwind.css";
import "./explain-visualizer.css";
import { createRoot } from "react-dom/client";
import { ExplainVisualizerApp } from "./ExplainVisualizerApp";

// Some bundled dependencies expect a `global` reference on the window.
(globalThis as typeof globalThis & Record<string, unknown>).global = globalThis;

const container = document.getElementById("app");
if (container) {
  createRoot(container).render(<ExplainVisualizerApp />);
}
