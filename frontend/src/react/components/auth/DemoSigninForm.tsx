import { create } from "@bufbuild/protobuf";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";

type Props = {
  readonly loading: boolean;
  readonly onSignin: (request: LoginRequest) => void;
};

// NOSONAR: Not a real credential — fixed demo account password, only used in demo mode.
const DEMO_PASSWORD = "12345678"; // NOSONAR

const ACCOUNT_OPTIONS = [
  { label: "Demo (Workspace Admin)", value: "demo@example.com" },
  { label: "Dev1 (Developer)", value: "dev1@example.com" },
  { label: "DBA1 (DBA)", value: "dba1@example.com" },
  { label: "QA1 (QA)", value: "qa1@example.com" },
];

export function DemoSigninForm({ loading, onSignin }: Props) {
  const { t } = useTranslation();
  const [selectedEmail, setSelectedEmail] = useState("demo@example.com");

  const trySignin = (e: React.FormEvent) => {
    e.preventDefault();
    onSignin(
      create(LoginRequestSchema, {
        email: selectedEmail,
        password: DEMO_PASSWORD,
      })
    );
  };

  return (
    <form className="flex flex-col gap-y-6 px-1" onSubmit={trySignin}>
      <div>
        <label
          htmlFor="demo-account"
          className="block text-sm font-medium leading-5 text-control"
        >
          {t("auth.sign-in.demo-account")}
        </label>
        <div className="mt-1">
          <select
            id="demo-account"
            value={selectedEmail}
            onChange={(e) => setSelectedEmail(e.target.value)}
            className="flex h-10 w-full rounded-xs border border-control-border bg-transparent px-3 text-sm text-main focus:outline-hidden"
          >
            {ACCOUNT_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="w-full">
        <Button type="submit" size="lg" className="w-full" disabled={loading}>
          {t("common.sign-in")}
        </Button>
      </div>
    </form>
  );
}
