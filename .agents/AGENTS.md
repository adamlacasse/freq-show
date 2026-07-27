# Project Guidelines & Context

- **Deployment Host:** Freq Show is deployed on **Render**.
- **Database Persistence:** When running on Render, set `DATABASE_URL` to point to a mounted Persistent Disk (e.g. `file:/var/data/freqshow.db?_fk=1`) to persist collection data across deployments.
