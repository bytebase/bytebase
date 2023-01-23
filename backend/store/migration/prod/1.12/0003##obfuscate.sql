CREATE OR REPLACE FUNCTION bytea_to_bit(
    IN bytea1 BYTEA
)
RETURNS BIT VARYING
AS
$BODY$
DECLARE
    outbits BIT VARYING := b'';
BEGIN
    outbits := ('x' || ltrim(bytea1::text, '\x'))::bit varying;
RETURN outbits;
END;
$BODY$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION bit_to_bytea(
    IN bits1 BIT VARYING
)
RETURNS bytea
AS
$BODY$
DECLARE
    bitslen INTEGER;
    strbits TEXT := '';
    outbytea BYTEA := NULL::BYTEA;
BEGIN
    bitslen := octet_length(bits1);
    FOR i IN 0 .. bitslen - 1 LOOP
        strbits := strbits || lpad(to_hex(substring(bits1 from (i * 8) + 1  for 8)::int), 2, '0');
    END LOOP;
    outbytea := decode(strbits, 'hex');
RETURN outbytea;
END;
$BODY$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION bytea_xor(
    IN bytea1 BYTEA,
    IN bytea2 BYTEA
)
RETURNS BYTEA
AS
$BODY$
DECLARE
    xorbits BIT VARYING := b'';
    outbytea BYTEA := null::BYTEA;
BEGIN
    xorbits := bytea_to_bit(bytea1) # bytea_to_bit(bytea2);
    outbytea := bit_to_bytea(xorbits);
RETURN outbytea;
END;
$BODY$
LANGUAGE plpgsql;

UPDATE data_source
SET
	password = encode(bytea_xor(
		password::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(password::bytea)/32), length(password::bytea))::bytea
	), 'base64'),
	ssl_key = encode(bytea_xor(
		ssl_key::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_key::bytea)/32), length(ssl_key::bytea))::bytea
	), 'base64'),
	
	ssl_cert = encode(bytea_xor(
		ssl_cert::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_cert::bytea)/32), length(ssl_cert::bytea))::bytea
	), 'base64'),
	
	ssl_ca = encode(bytea_xor(
		ssl_ca::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_ca::bytea)/32), length(ssl_ca::bytea))::bytea
	), 'base64')
;

DROP FUNCTION IF EXISTS bytea_xor;
DROP FUNCTION IF EXISTS bit_to_bytea;
DROP FUNCTION IF EXISTS bytea_to_bit;
