import { create } from "@bufbuild/protobuf";
import { Eye, EyeOff } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { router } from "@/router";
import { AUTH_PASSWORD_FORGOT_MODULE } from "@/router/auth";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { resolveWorkspaceName } from "@/utils";

type Props = {
  readonly loading: boolean;
  readonly showForgotPassword?: boolean;
  readonly credentialLabel?: string;
  readonly credentialPlaceholder?: string;
  readonly credentialInputType?: "email" | "text";
  readonly credentialAutocomplete?: string;
  readonly onSignin: (request: LoginRequest) => void;
};

export function PasswordSigninForm({
  loading,
  showForgotPassword = true,
  credentialLabel,
  credentialPlaceholder = "jim@example.com",
  credentialInputType = "email",
  credentialAutocomplete = "on",
  onSignin,
}: Props) {
  const { t } = useTranslation();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const credentialFieldID =
    credentialInputType === "text" ? "username" : "email";

  useEffect(() => {
    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    setIdentifier(params.get("email") ?? "");
    setPassword(params.get("password") ?? "");
  }, []);

  const allowSignin = identifier && password;

  const trySignin = (e: React.FormEvent) => {
    e.preventDefault();
    onSignin(
      create(LoginRequestSchema, {
        email: identifier,
        password,
        workspace: resolveWorkspaceName(),
      })
    );
  };

  const goToForgot = (e: React.MouseEvent) => {
    e.preventDefault();
    router.push({
      name: AUTH_PASSWORD_FORGOT_MODULE,
      query: { hint: router.currentRoute.value.query.hint },
    });
  };

  return (
    <form className="flex flex-col gap-y-6 px-1" onSubmit={trySignin}>
      <div>
        <label
          htmlFor={credentialFieldID}
          className="block text-sm font-medium leading-5 text-control"
        >
          {credentialLabel ?? t("common.email")}
          <span className="text-error ml-0.5">*</span>
        </label>
        <div className="mt-1 rounded-md shadow-xs">
          <Input
            id={credentialFieldID}
            type={credentialInputType}
            autoComplete={credentialAutocomplete}
            placeholder={credentialPlaceholder}
            required
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
          />
        </div>
      </div>

      <div>
        <label
          htmlFor="password"
          className="flex justify-between text-sm font-medium leading-5 gap-4 text-control"
        >
          <div>
            {t("common.password")}
            <span className="text-error ml-0.5">*</span>
          </div>
          {showForgotPassword && (
            <a
              href="#"
              className="text-sm font-normal text-control-light hover:underline focus:outline-hidden"
              tabIndex={-1}
              onClick={goToForgot}
            >
              {t("auth.sign-in.forget-password")}
            </a>
          )}
        </label>
        <div className="relative flex flex-row items-center mt-1 rounded-md shadow-xs">
          <Input
            id="password"
            type={showPassword ? "text" : "password"}
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button
            type="button"
            className="hover:cursor-pointer absolute right-3"
            onClick={() => setShowPassword((v) => !v)}
            aria-label="Toggle password visibility"
          >
            {showPassword ? (
              <Eye className="w-4 h-4" />
            ) : (
              <EyeOff className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>

      <div className="w-full">
        <Button
          type="submit"
          size="lg"
          className="w-full"
          disabled={!allowSignin || loading}
        >
          {t("common.sign-in")}
        </Button>
      </div>
    </form>
  );
}
