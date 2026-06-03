import type { RouteObject } from "react-router-dom";
import { authRoutes } from "@/react/router/routes/auth";
import { dashboardRoutes } from "@/react/router/routes/dashboard";
import { sqlEditorRoutes } from "@/react/router/routes/sqlEditor";

// React-router route table translated from the vue-router route definitions
// (`@/router/auth.ts`, `@/router/setup.ts`, `@/router/sqlEditor.ts`,
// `@/router/dashboard/**`). Composed in the same order as the vue router's
// `routes: [...authRoutes, ...setupRoutes, ...dashboardRoutes,
// ...sqlEditorRoutes]` (setup is folded into authRoutes here, both rendering
// the SplashLayout). Wiring this table into the application root happens in a
// later phase.
export const routes: RouteObject[] = [
  ...authRoutes,
  ...dashboardRoutes,
  ...sqlEditorRoutes,
];
