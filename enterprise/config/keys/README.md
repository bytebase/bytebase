# License

## Intro

These public keys are used to decrypt JWT license.

## Create license

We can use `openssl` to generate RSA key

```bash
# Create private key
openssl genrsa -out private.pem 2048

# Use private key to generate public key
openssl rsa -in private.pem -outform PEM -pubout -out public.pem
```

## Usage

### Load keys in config

The `enterprise/config/config.go` will load keys depends on environment.

### Decrypt

In `enterprise/service/license.go`, we will use public key to parse JWT

```go
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
    if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
    }

    key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(env.Conf.LicensePubKey))
    if err != nil {
        return nil, err
    }

    return key
}
```
