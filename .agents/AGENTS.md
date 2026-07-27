# Project Guidelines & Context

- **Deployment Host:** Freq Show is deployed on **Render**.
- **Database Persistence:** `DATABASE_URL` is required when `DATABASE_DRIVER=sqlite` (no default). On Render it must point at the mounted Persistent Disk: `file:/var/data/freqshow.db?_pragma=foreign_keys(1)`. Use `_pragma=foreign_keys(1)`, not `_fk=1` — the driver is modernc.org/sqlite, which silently ignores unrecognised parameters.
