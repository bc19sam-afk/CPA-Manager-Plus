# CPA Panel Mode

CPA Panel mode is for environments that still open the panel from the CPA port. CPA hosts the panel and the browser holds the CPA Management Key. If you need full historical monitoring, model prices, import/export, or server-side inspection, use Full Docker or native Manager Server mode instead.

## Differences From Full Docker Mode

| Mode | Panel host | Login credential | Use case |
|---|---|---|---|
| Full Docker | Manager Server `:18317` | `cmp_admin_...` admin key | Run CPAMP independently. |
| CPA Panel | CPA `:8317` | CPA Management Key | Keep accessing the panel from CPA. |
| Frontend development | Vite dev server or static HTML | Browser-local CPA URL and key | Local development and debugging. |

## Notes

- CPA Panel mode uses the CPA Management Key for login. It does not require the CPAMP Admin Key.
- The CPA Management Key stays in the browser, matching CPA-hosted panel access.
- Full Docker mode encrypts the CPA Management Key and stores it in SQLite.
- The panel entry is the same, but available data depends on the hosting mode. Full historical monitoring, model prices, and server-side inspection come from Manager Server mode.

## When To Use It

Choose CPA Panel mode when:

- Users are already used to opening the management panel from the CPA port.
- You do not want users to access the Manager Server panel port directly.
- You want the CPA Management Key to remain the panel access credential.

Choose Full Docker mode when:

- You want CPAMP to host itself independently.
- You want Manager Server to store configuration centrally.
- You need an admin key and server-side encrypted storage for the CPA Management Key.
