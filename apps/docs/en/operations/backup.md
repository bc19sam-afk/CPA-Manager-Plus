# Backup And Restore

CPAMP keeps request history, configuration, and encrypted credentials in the local data directory. The common mistake is backing up only `usage.sqlite` and missing WAL/SHM files or `data.key`.

## Required Backup Files

Back up these files as a set:

- `usage.sqlite`
- `usage.sqlite-wal`
- `usage.sqlite-shm`
- `data.key`

If your deployment directory contains custom configuration files, back them up too.

## Why data.key Is Required

The CPA Management Key is encrypted with `data.key` before being saved to SQLite.

- If only `usage.sqlite` leaks, an attacker cannot directly read the CPA Management Key.
- If both `usage.sqlite` and `data.key` leak, the CPA Management Key can be decrypted.
- If `data.key` is lost, the saved CPA Management Key cannot be recovered. You must save the CPA connection configuration again.

## Docker Backup Example

If you use a named volume, stop the container first, then export through a temporary container:

```bash
docker stop cpa-manager-plus
docker run --rm \
  -v cpa-manager-plus-data:/data:ro \
  -v "$PWD":/backup \
  alpine \
  tar czf /backup/cpa-manager-plus-data.tgz -C /data .
docker start cpa-manager-plus
```

If you use a host directory mount:

```bash
docker stop cpa-manager-plus
cp -a /srv/cpa-manager-plus-data /srv/cpa-manager-plus-data.backup
docker start cpa-manager-plus
```

## Native Package Backup

Stop the process, then copy the data directory:

```bash
cp -a ./data ./data.backup
```

Windows PowerShell:

```powershell
Copy-Item -Recurse .\data .\data.backup
```

## Restore

1. Stop CPAMP.
2. Restore the full data directory.
3. Confirm that `usage.sqlite` and `data.key` come from the same backup.
4. Start CPAMP.
5. Log in and check configuration, monitoring data, and collector status.

If restore produces decryption errors, first check whether `data.key` matches the SQLite database.
