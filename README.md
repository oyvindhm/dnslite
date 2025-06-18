# dnslite

**dnslite** is a lightweight DNS server written in Go that supports:
- Full DNS record resolution from PostgreSQL
- Optional DNSSEC signing
- Master-slave zone replication via HTTP `/zone-sync`
- Dockerized deployment

---

## Features

- ✅ A/AAAA/CNAME/MX/NS/TXT/DNSKEY support
- 🔐 DNSSEC (RSA with automatic RRSIG generation)
- 🔄 Master/slave syncing with role-based configuration
- 📦 PostgreSQL-based zone storage
- 🐳 Docker support

---

## Directory Structure

```
.
├── api/               # HTTP API endpoints
├── cache/             # Optional in-memory caching
├── config/            # Environment variable loading
├── db/                # PostgreSQL queries
├── dnssec/            # Key management, RRSIG signing
├── handler/           # DNS request handling
├── secrets/           # DNSSEC private/public key storage
├── slave/             # Slave replication logic
├── tools/             # CLI tools like genkey and resign
├── Dockerfile
├── docker-compose.yml
├── .env
└── main.go
```

---

## Setup

### 1. Clone and configure

```bash
git clone https://github.com/yourname/dnslite.git
cd dnslite
cp .env.example .env
```

Edit `.env`:

```env
POSTGRES_USER=dnslite
POSTGRES_PASSWORD=mysecretpassword
POSTGRES_DB=dnslite
DB_URL=postgres://dnslite:mysecretpassword@db:5432/dnslite
SERVER_ROLE=master         # or 'slave'
MASTER_URL=http://master:8080/zone-sync
```

---

### 2. Start with Docker

```bash
docker-compose up --build -d
```

This will start:
- PostgreSQL (`db`)
- Go-based DNS server (`dns`)

DNS listens on **port 53 TCP/UDP** and HTTP API on **port 8080**.

---

## Tools

### ➕ Generate DNSSEC keypair

```bash
docker exec -it dnslite_dns_1 go run tools/genkey.go elns.no
```

Generates:
- `secrets/elns.no/dnskey.txt`
- `secrets/elns.no/key.pem`

---

### 🔐 Re-sign all records

```bash
docker exec -it dnslite_dns_1 go run tools/resignall.go
```

This signs all DNS records and inserts the RRSIG into the DB.

---

## API Endpoints

| Endpoint        | Method | Description                      |
|-----------------|--------|----------------------------------|
| `/zone-sync`    | GET    | Full zone dump for slave sync    |
| `/status`       | GET    | Shows current server role & state |

---

## Syncing Zones (Slave)

Set `.env` on the slave:

```env
SERVER_ROLE=slave
MASTER_URL=http://your-master-host:8080/zone-sync
```

Slaves will fetch all records and signatures every 5 minutes and upsert them.

---

## DNSSEC Behavior

- DNSSEC keys are stored per-zone in `secrets/<zone>/`
- Only zones with keys are signed
- Signature is automatically regenerated on record changes (if you call `resignall`)

---

## Contributing

Pull requests welcome! Areas for contribution:

- TSIG authentication
- Zonefile import/export
- UI interface for zone management
- More caching logic

---

## License

MIT. Do what you want.
