package plsql

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestDiagnoseBYT10010CustomerStatement verifies the MBBank statement from
// BYT-10010 passes SQL Review diagnosis end to end after the omni
// constraint_state fix.
func TestDiagnoseBYT10010CustomerStatement(t *testing.T) {
	stmt := `CREATE TABLE DATA.RC_FT_ADJ_CONFIG
(
    RC_FT_ADJ_CODE        VARCHAR2(250) NOT NULL,
    RC_CODE               VARCHAR2(150),
    DATE_REPORT           DATE,
    IS_CHECK              VARCHAR2(255),
    IS_ACTION             VARCHAR2(255),
    CREATE_DATE           DATE DEFAULT SYSDATE NOT NULL,
    CREATE_USER           VARCHAR2(50) DEFAULT 'ETL_USER',
    UPDATE_DATE           DATE DEFAULT SYSDATE,
    UPDATE_USER           VARCHAR2(50) DEFAULT 'ETL_USER',
    TYPE                  VARCHAR2(50),
    RC_FT_ADJ_STATUS      VARCHAR2(255),
    PARENT_RC_FT_ADJ_CODE VARCHAR2(250),
    E_STATUS              NUMBER,

    CONSTRAINT PK_RC_FT_ADJ_CONFIG
        PRIMARY KEY (RC_FT_ADJ_CODE, CREATE_DATE) USING INDEX LOCAL
)
PARTITION BY RANGE (CREATE_DATE)
INTERVAL (NUMTOYMINTERVAL(1, 'MONTH'))
(
    PARTITION P202607
        VALUES LESS THAN (DATE '2026-07-01')
)`
	diags, err := Diagnose(context.Background(), base.DiagnoseContext{}, stmt)
	if err != nil {
		t.Fatalf("Diagnose returned error: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d: %+v", len(diags), diags)
	}

	// DBMS_METADATA-style variant with NOT NULL ENABLE and USING INDEX attributes.
	stmt2 := `CREATE TABLE "S"."T" ("A" NUMBER(*,0) NOT NULL ENABLE, CONSTRAINT "PK_T" PRIMARY KEY ("A") USING INDEX PCTFREE 10 INITRANS 2 MAXTRANS 255 COMPUTE STATISTICS TABLESPACE "USERS" ENABLE) PCTFREE 10 PCTUSED 40 INITRANS 1 MAXTRANS 255 TABLESPACE "USERS"`
	diags2, err := Diagnose(context.Background(), base.DiagnoseContext{}, stmt2)
	if err != nil {
		t.Fatalf("Diagnose returned error: %v", err)
	}
	if len(diags2) != 0 {
		t.Fatalf("expected no diagnostics for DBMS_METADATA style, got %+v", diags2)
	}
}
