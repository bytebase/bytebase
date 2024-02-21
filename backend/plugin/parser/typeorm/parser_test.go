package typeorm

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	want := []string{
		`CREATE TABLE "core"."refreshToken" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "userId" uuid NOT NULL, "expiresAt" TIMESTAMP WITH TIME ZONE NOT NULL, "deletedAt" TIMESTAMP WITH TIME ZONE, "revokedAt" TIMESTAMP WITH TIME ZONE, "createdAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), CONSTRAINT "PK_7d8bee0204106019488c4c50ffa" PRIMARY KEY ("id"))`,
		`CREATE TABLE "core"."workspace" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "domainName" character varying, "displayName" character varying, "logo" character varying, "inviteHash" character varying, "deletedAt" TIMESTAMP, "createdAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), CONSTRAINT "PK_098656ae401f3e1a4586f47fd8e" PRIMARY KEY ("id"))`,
		`CREATE TABLE "core"."user" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "firstName" character varying NOT NULL DEFAULT '', "lastName" character varying NOT NULL DEFAULT '', "email" character varying NOT NULL, "emailVerified" boolean NOT NULL DEFAULT false, "disabled" boolean NOT NULL DEFAULT false, "passwordHash" character varying, "canImpersonate" boolean NOT NULL DEFAULT false, "createdAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(), "deletedAt" TIMESTAMP, "defaultWorkspaceId" uuid, CONSTRAINT "PK_a3ffb1c0c8416b9fc6f907b7433" PRIMARY KEY ("id"))`,
		`ALTER TABLE "core"."refreshToken" ADD CONSTRAINT "FK_610102b60fea1455310ccd299de" FOREIGN KEY ("userId") REFERENCES "core"."user"("id") ON DELETE NO ACTION ON UPDATE NO ACTION`,
		`ALTER TABLE "core"."user" ADD CONSTRAINT "FK_5d77e050eabd28d203b301235a7" FOREIGN KEY ("defaultWorkspaceId") REFERENCES "core"."workspace"("id") ON DELETE NO ACTION ON UPDATE NO ACTION`,
	}
	f, err := os.Open("test-data/core_table.ts")
	require.NoError(t, err)
	defer f.Close()
	bytes, err := io.ReadAll(f)
	require.NoError(t, err)
	got, err := Parse(string(bytes))
	require.NoError(t, err)
	require.Equal(t, want, got)
}
