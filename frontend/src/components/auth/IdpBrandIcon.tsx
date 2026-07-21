import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";

type Brand = "github" | "google" | "gitlab";

type Props = {
  readonly idp: IdentityProvider;
  readonly className?: string;
};

function hostnameOf(url: string): string {
  try {
    return new URL(url).hostname.toLowerCase();
  } catch {
    return "";
  }
}

function brandFromHost(host: string): Brand | undefined {
  if (host === "github.com" || host.endsWith(".github.com")) return "github";
  if (host === "google.com" || host.endsWith(".google.com")) return "google";
  if (host === "gitlab.com" || host.endsWith(".gitlab.com")) return "gitlab";
  return undefined;
}

function brandFromTitle(title: string): Brand | undefined {
  const lowered = title.toLowerCase();
  if (lowered.includes("github")) return "github";
  if (lowered.includes("google")) return "google";
  if (lowered.includes("gitlab")) return "gitlab";
  return undefined;
}

function configUrlOf(idp: IdentityProvider): string {
  const config = idp.config?.config;
  if (config?.case === "oauth2Config") return config.value.authUrl;
  if (config?.case === "oidcConfig") return config.value.issuer;
  return "";
}

// The config URL identifies where the user actually authenticates, so it is
// authoritative: when present, a non-brand host renders no icon rather than
// falling back to the admin-editable title (which could claim another brand).
function resolveBrand(idp: IdentityProvider): Brand | undefined {
  const configUrl = configUrlOf(idp);
  if (configUrl) {
    return brandFromHost(hostnameOf(configUrl));
  }
  return brandFromTitle(idp.title);
}

// Brand marks use their official colors — Google's branding guidelines
// require the standard-color "G", so raw hex values are intentional here.
export function IdpBrandIcon({ idp, className }: Props) {
  const brand = resolveBrand(idp);
  if (brand === "github") {
    return (
      <svg
        viewBox="0 0 16 16"
        aria-hidden="true"
        fill="currentColor"
        className={className}
      >
        <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
      </svg>
    );
  }
  if (brand === "google") {
    return (
      <svg viewBox="0 0 18 18" aria-hidden="true" className={className}>
        <path
          fill="#4285F4"
          d="M17.64 9.2c0-.637-.057-1.251-.164-1.84H9v3.481h4.844a4.14 4.14 0 0 1-1.796 2.716v2.259h2.908c1.702-1.567 2.684-3.875 2.684-6.615Z"
        />
        <path
          fill="#34A853"
          d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.26c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332A8.997 8.997 0 0 0 9 18Z"
        />
        <path
          fill="#FBBC05"
          d="M3.964 10.71A5.41 5.41 0 0 1 3.682 9c0-.593.102-1.17.282-1.71V4.958H.957A8.996 8.996 0 0 0 0 9c0 1.452.348 2.827.957 4.042l3.007-2.332Z"
        />
        <path
          fill="#EA4335"
          d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0A8.997 8.997 0 0 0 .957 4.958L3.964 7.29C4.672 5.163 6.656 3.58 9 3.58Z"
        />
      </svg>
    );
  }
  if (brand === "gitlab") {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true" className={className}>
        <path fill="#E24329" d="m12 21.42 3.684-11.333H8.316L12 21.42Z" />
        <path fill="#FC6D26" d="M12 21.42 8.316 10.087H3.16L12 21.42Z" />
        <path
          fill="#FCA326"
          d="m3.16 10.087-1.12 3.447a.762.762 0 0 0 .278.852L12 21.42 3.16 10.087Z"
        />
        <path
          fill="#E24329"
          d="M3.16 10.087h5.156L6.1 3.27a.38.38 0 0 0-.728 0L3.16 10.087Z"
        />
        <path fill="#FC6D26" d="m12 21.42 3.684-11.333h5.156L12 21.42Z" />
        <path
          fill="#FCA326"
          d="m20.84 10.087 1.12 3.447a.762.762 0 0 1-.278.852L12 21.42l8.84-11.333Z"
        />
        <path
          fill="#E24329"
          d="M20.84 10.087h-5.156L17.9 3.27a.38.38 0 0 1 .728 0l2.212 6.817Z"
        />
      </svg>
    );
  }
  return null;
}
