# Configuration And Data Directory

CPAMP stores its core data locally. During deployment, identify three things first: where SQLite lives, how `data.key` is stored, and where the admin key comes from.

## Key Files

| File | Description |
|---|---|
| `usage.sqlite` | SQLite database for request events, configuration, prices, aliases, and related data. |
| `usage.sqlite-wal` | SQLite WAL file. Back it up when present. |
| `usage.sqlite-shm` | SQLite SHM file. Back it up when present. |
| `data.key` | Data key used to encrypt the saved CPA Management Key. |

Docker defaults:

```text
/data/usage.sqlite
/data/data.key
```

Native package defaults:

```text
./data/usage.sqlite
./data/data.key
```

## Admin Key

Full Docker and native Manager Server modes use a `cmp_admin_...` admin key for login.

Configure it with:

| Variable | Description |
|---|---|
| `CPA_MANAGER_ADMIN_KEY` | Pass the admin key directly. |
| `CPA_MANAGER_ADMIN_KEY_FILE` | Read the admin key from a file. |

If it is not configured, the first startup generates a random admin key and prints it to the logs. It will not be shown again.

## CPA Management Key

CPAMP uses the CPA Management Key to access the CPA management API.

In Full Docker and native Manager Server modes, CPAMP encrypts the CPA Management Key with `data.key` before saving it to SQLite.

In CPA Panel mode, the browser holds the CPA Management Key, matching CPA-hosted panel access semantics.

## Collection Configuration

Recommended setting:

```text
USAGE_COLLECTOR_MODE=auto
```

Auto mode tries RESP Pub/Sub, HTTP queue, and RESP pop in order.

Constraints:

- RESP connections must connect directly to the CPA API port, usually `8317`.
- HTTP queue can go through an HTTP proxy.
- `pollIntervalMs` should not exceed the CPA usage queue retention.
- CPA retention defaults to 60s and is capped at 3600s.
- Only one Manager Server should consume the same CPA queue.
