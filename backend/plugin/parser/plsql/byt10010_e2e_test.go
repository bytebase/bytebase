package plsql

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestDiagnoseBYT10010UsingIndexLocal verifies that an interval-partitioned
// table whose primary key uses USING INDEX LOCAL (BYT-10010) passes SQL Review
// diagnosis end to end after the omni constraint_state fix.
func TestDiagnoseBYT10010UsingIndexLocal(t *testing.T) {
	stmt := `CREATE TABLE APP.ORDER_EVENT
(
    EVENT_CODE            VARCHAR2(250) NOT NULL,
    ORDER_CODE            VARCHAR2(150),
    EVENT_DATE            DATE,
    IS_VERIFIED           VARCHAR2(255),
    IS_ARCHIVED           VARCHAR2(255),
    CREATE_DATE           DATE DEFAULT SYSDATE NOT NULL,
    CREATE_USER           VARCHAR2(50) DEFAULT 'APP_USER',
    UPDATE_DATE           DATE DEFAULT SYSDATE,
    UPDATE_USER           VARCHAR2(50) DEFAULT 'APP_USER',
    TYPE                  VARCHAR2(50),
    EVENT_STATUS          VARCHAR2(255),
    PARENT_EVENT_CODE     VARCHAR2(250),
    RETRY_COUNT           NUMBER,

    CONSTRAINT PK_ORDER_EVENT
        PRIMARY KEY (EVENT_CODE, CREATE_DATE) USING INDEX LOCAL
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
