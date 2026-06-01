# Cloudflare Workers — deferred

Wire when a hosted HTTP endpoint is genuinely needed.

The `healthz/` worker returns `{"status":"ok"}` 200 for `/healthz` and is ready to deploy:

```bash
cd healthz
npm install
npx wrangler deploy   # needs CF_API_TOKEN + CF_ACCOUNT_ID
```

GitHub Actions secrets required when re-enabling:
- `CF_API_TOKEN` — Cloudflare API token (Workers:Edit permission)
- `CF_ACCOUNT_ID` — Cloudflare account ID
