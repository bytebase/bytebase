import React from "react";
import { createRoot } from "react-dom/client";

const container = document.getElementById("app-rc");
const root = createRoot(container as HTMLElement);
root.render(<div className="hidden">Hello world</div>);
