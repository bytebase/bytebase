export function isDev() {
  return process.env.NODE_ENV === "development";
}

export function isDemo() {
  return import.meta.env.MODE === "demo";
}

export function isDevOrDemo() {
  return isDev() || isDemo();
}
