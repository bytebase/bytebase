// Auth/landing route-name literals. Inlined (no imports) on purpose: this
// helper lives in the widely-imported `@/utils` barrel, so importing the route
// constants from `@/react/router/handles` here would add that module to the
// barrel's load graph and tip a fragile load-order cycle (app store ↔ @/utils
// ↔ @/types). The values mirror `@/react/router/handles`.
const AUTH_RELATED_ROUTES = [
  "auth.signin",
  "auth.signin.admin",
  "auth.signup",
  "auth.mfa",
  "auth.password.reset",
  "auth.password.forgot",
  "auth.oauth.callback",
  "auth.oidc.callback",
];

export const isAuthRelatedRoute = (routeName: string) => {
  return AUTH_RELATED_ROUTES.includes(routeName);
};
