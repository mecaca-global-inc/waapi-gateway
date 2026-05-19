# Contributing

Thanks for considering a contribution to **WAAPI Gateway**.

## Ground rules

- Be excellent. No harassment, sketch tactics, or spam automation built on top of this code.
- This project wraps [whatsmeow](https://github.com/tulir/whatsmeow). Anything that breaks WhatsApp's ToS, evades anti-spam measures, or enables mass unsolicited messaging is out of scope.
- One concern per PR. Don't bundle a refactor with a feature.

## Dev setup

```bash
go version    # 1.26+
node -v       # 22+
git clone https://github.com/<you>/waapi-gateway
cd waapi-gateway
cp .env.example .env
# set a strong ADMIN_PASS (must NOT be admin/changeme/password/empty)
go run ./cmd/server   # backend on :3000
cd dashboard && npm install && npm run dev -- -p 3001
```

## Tests

```bash
go test ./...
cd dashboard && npx tsc --noEmit && npm run build
```

## Pull request checklist

- [ ] `go vet ./...` clean
- [ ] `go test ./...` passes
- [ ] Dashboard builds (`npm run build`)
- [ ] Updated `internal/api/openapi.yaml` if you touched a route
- [ ] Updated README if behaviour changed
- [ ] No `.env` or secrets in the diff

## Reporting security issues

Do **not** open a public issue for security bugs. Email security@waapi.link with details and reproduction steps.
