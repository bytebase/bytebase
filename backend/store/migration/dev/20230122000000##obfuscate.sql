CREATE OR REPLACE FUNCTION f_bytea_to_bit(
    IN i_bytea BYTEA
)
RETURNS BIT VARYING
AS
$BODY$
DECLARE
    w_bit BIT VARYING := b'';
BEGIN
    w_bit := ('x' || ltrim(i_bytea::text, '\x'))::bit varying;
RETURN w_bit;
END;
$BODY$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION f_bit_to_bytea(
    IN i_bit BIT VARYING
)
RETURNS bytea
AS
$BODY$
DECLARE
    w_panel_data_len INTEGER;
    w_str_bit TEXT := '';
    w_bytea BYTEA := NULL::BYTEA;
BEGIN
    /* Get number of bytes */
    w_panel_data_len := octet_length(i_bit);

    IF length(i_bit) % 8 != 0 THEN
        RAISE 'Can not convert to bytea. The passed argument is % bits', length(i_bit);
    END IF;

    FOR i IN 0 .. w_panel_data_len - 1 LOOP
        w_str_bit := w_str_bit || lpad(to_hex(substring(i_bit from (i * 8) + 1  for 8)::int), 2, '0');
    END LOOP;

    w_bytea := decode(w_str_bit, 'hex');

RETURN w_bytea;
END;
$BODY$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION f_bytea_xor(
    IN i_bytea_1 BYTEA,
    IN i_bytea_2 BYTEA
)
RETURNS BYTEA
AS
$BODY$
DECLARE
    w_bit BIT VARYING := b'';
    w_bytea BYTEA := null::BYTEA;
BEGIN
    w_bit := f_bytea_to_bit(i_bytea_1) # f_bytea_to_bit(i_bytea_2);
    w_bytea := f_bit_to_bytea(w_bit);
RETURN w_bytea;
END;
$BODY$
LANGUAGE plpgsql;

UPDATE data_source
SET
	password = encode(f_bytea_xor(
		password::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(password::bytea)/32), length(password::bytea))::bytea
	), 'base64'),
	ssl_key = encode(f_bytea_xor(
		ssl_key::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_key::bytea)/32), length(ssl_key::bytea))::bytea
	), 'base64'),
	
	ssl_cert = encode(f_bytea_xor(
		ssl_cert::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_cert::bytea)/32), length(ssl_cert::bytea))::bytea
	), 'base64'),
	
	ssl_ca = encode(f_bytea_xor(
		ssl_ca::bytea,
		LEFT(REPEAT((SELECT value FROM setting WHERE name = 'bb.auth.secret'), 1+length(ssl_ca::bytea)/32), length(ssl_ca::bytea))::bytea
	), 'base64')
;

DROP FUNCTION IF EXISTS f_bytea_xor;
DROP FUNCTION IF EXISTS f_bit_to_bytea;
DROP FUNCTION IF EXISTS f_bytea_to_bit;
