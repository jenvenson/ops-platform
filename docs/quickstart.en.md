# Quick Start

Get OPS Platform running locally and complete your first login in five minutes.

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Docker | 20.10+ | Required |
| Docker Compose | v2 (`docker compose`) | Required |
| Available RAM | ≥ 4 GB | MySQL + Redis + backend + frontend run concurrently |
| Available Disk | ≥ 5 GB | Images + data volumes |

> **Verify your setup**: Run `docker compose version`. If it prints a version number, you're good to go.

---

## Step 1: Clone the Repository

```bash
git clone https://github.com/jenvenson/ops-platform.git
cd ops-platform
```

---

## Step 2: Configure Environment Variables

```bash
cp deploy/.env.example deploy/.env
```

Open `deploy/.env` in any editor and **change the following three values** (everything else can stay as-is):

```ini
DB_PASSWORD=choose_a_secure_mysql_password
REDIS_PASSWORD=choose_a_secure_redis_password
JWT_SECRET=choose_a_random_string_at_least_32_chars
```

---

## Step 3: Start the Services

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev up -d
```

**First run** pulls base images from Docker Hub (golang, mysql, redis, nginx) and downloads Go dependencies — expect **3–5 minutes** depending on your connection speed.

Follow the startup progress:

```bash
docker logs -f ops-backend-dev
```

The backend is ready when you see:

```
[GIN-debug] Listening and serving HTTP on :8080
```

Press `Ctrl+C` to stop tailing logs (services keep running in the background).

> **Port mapping**: Containers use standard internal ports; the host machine maps them to avoid conflicts:
>
> | Service | Host Port | Container Port |
> |---------|-----------|----------------|
> | Frontend | 18890 | 80 |
> | Backend API | 28080 | 8080 |
> | MySQL | 23306 | 3306 |
> | Redis | 16379 | 6379 |

---

## Step 4: Verify and Log In

**Check backend health:**

```bash
curl http://localhost:28080/health
# Expected: {"status":"ok","checks":{"database":"ok"}}
```

**Open your browser:**

```
http://localhost:18890
```

Log in with the default credentials:

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `admin123` |

> After your first login, please change the password in **Profile** (top-right corner).

---

## Troubleshooting

<details>
<summary><b>Browser shows a blank page or "cannot connect" at port 18890</b></summary>

The frontend container may still be installing dependencies. Wait 1–2 minutes and refresh. To check:

```bash
docker logs -f ops-frontend-dev
```

Look for `VITE ready` — that means the frontend is up.
</details>

<details>
<summary><b>Backend log shows "dial tcp: connection refused"</b></summary>

MySQL or Redis hasn't finished initializing yet. The backend will reconnect automatically. If the error persists, double-check that the passwords in `deploy/.env` are set correctly.
</details>

<details>
<summary><b>Port conflict (18890 / 28080 / 23306 already in use)</b></summary>

Edit the port variables in `deploy/.env`, or update the `ports` mappings in `deploy/docker-compose.dev.yml`, then restart:

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down
docker compose -f deploy/docker-compose.dev.yml -p ops-dev up -d
```
</details>

<details>
<summary><b>MySQL authentication plugin error</b></summary>

```bash
docker exec ops-mysql mysql -uroot -p"${DB_PASSWORD}" -e \
  "ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY '${DB_PASSWORD}'; FLUSH PRIVILEGES;"
```
</details>

---

## Stopping the Services

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down
```

Data volumes are preserved — your data will still be there next time you run `up -d`. To remove all data as well:

```bash
docker compose -f deploy/docker-compose.dev.yml -p ops-dev down -v
```

---

## What's Next

- [Feature Overview](../README.en.md#features)
- [AI Assistant Setup](../README.en.md#ai-assistant)
- [Production Deployment](../deploy/DEPLOY.md)
- [User Manual](user_manual.md)
