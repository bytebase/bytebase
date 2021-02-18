export function isDev() {
  return import.meta.env.DEV;
}

export function isDemo() {
  return import.meta.env.MODE === "demo";
}

export function isDevOrDemo() {
  return isDev() || isDemo();
}
