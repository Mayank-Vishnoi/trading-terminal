# trading-terminal

A small Go server that places Nifty / Bank Nifty option orders through the [Upstox v2 API](https://upstox.com/developer/api-documentation/). An order is triggered by the **underlying index price**. Background workers watch the price and enter or exit when your levels are hit.

> **Dry run:** the `PlaceOrder` calls in `workers/worker.go` are commented out, so no real orders are sent yet.

## Setup (once)

1. **Install Go:**
   ```bash
   brew install go
   ```
2. **Download dependencies** from the repo root. `go.mod` and `go.sum` are committed.
   ```bash
   go mod download
   ```
3. **Register the redirect URL.** In the [Upstox developer console](https://account.upstox.com/developer/apps), set your app's Redirect URL to exactly:
   ```
   http://localhost:8080/auth/gettoken
   ```
4. **Create `.env`** in the repo root with the secrets only. It is gitignored, so never commit it.
   ```bash
   API_KEY=            # Upstox app API key
   API_SECRET=         # Upstox app secret
   ACCESS_TOKEN=       # filled in daily, see below
   ```
5. **Check `config.json`** (committed). It holds the trading settings. The server loads it at boot and won't start if it's missing, has unknown keys, or has a value ≤ 0.

   | Key | Default | Meaning |
   |---|---|---|
   | `nifty_lots` | `3` | Nifty position size in **lots** (partial orders divide by 3) |
   | `banknifty_lots` | `3` | Bank Nifty position size in **lots** |
   | `ticker_seconds` | `1` | Price poll interval |
   | `hold_minutes` | `375` | Worker lifetime before timeout (375 = 9:15 to 15:30) |

   Edit it and restart the server to change these. Run the server from the repo root, because both `.env` and `config.json` are read from the current directory.

## Daily login

Upstox access tokens expire every day (around 3:30 AM IST), so do this each trading day.

1. **Start the server** in its own terminal. It prints the access token, so keep that output private.
   ```bash
   go run .
   ```
   You should see `Server listening on port 8080...`.
2. **Log in to Upstox.** Open this URL in a browser, replacing `<API_KEY>`:
   ```
   https://api.upstox.com/v2/login/authorization/dialog?response_type=code&client_id=<API_KEY>&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fauth%2Fgettoken
   ```
   Upstox redirects back to `/auth/gettoken?code=...`. The server exchanges the code for a token and prints `Access token: ...` in its terminal. The browser page stays blank.
3. **Save the token.** Copy it into `.env` as `ACCESS_TOKEN=...`.
4. **Restart the server** (`Ctrl+C`, then `go run .`). `.env` is only read at startup.
5. **Smoke test:**
   ```bash
   curl localhost:8080/workers/getactive
   ```
   It should return `Entry Workers: {}` and `Exit Workers: {}`.

## API

All routes are `GET` with query parameters, on `localhost:8080`.

| Route | Params | What it does |
|---|---|---|
| `/auth/gettoken` | `code` | OAuth redirect target. Exchanges the code for an access token. |
| `/auth/logout` | | Revokes the current access token. |
| `/order/place/ut` | `instrument_key`, `entry`, `target`, `stoploss` | Underlying-trigger order. It buys the option when the index reaches `entry`, then exits at `target` or `stoploss`. Target and stoploss must be at least 1:1. |
| `/order/place/partial` | `instrument_key`, `entry` | Partial entry: 1/3 at `entry`, then adds near the stoploss or at a reaffirmation level. Levels are currently hardcoded offsets from `entry`. |
| `/order/modify` | `id`, `target`, `stoploss` | Moves the target or stoploss of a running exit worker. |
| `/workers/getactive` | | Lists active entry and exit workers with their IDs. |
| `/workers/cancel` | `id` | Cancels a worker. |

Parameter details:
- `entry`, `target` and `stoploss` are **index** levels.
- `instrument_key` is the **option's** Upstox key, for example `NSE_FO|43885`. URL-encode the `|` as `%7C`:

```bash
curl "http://localhost:8080/order/place/ut?instrument_key=NSE_FO%7C43885&entry=24500&target=24600&stoploss=24450"
```
