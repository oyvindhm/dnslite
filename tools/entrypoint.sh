#!/bin/bash

# Create a cron job that re-signs all zones every 5 minutes
echo "*/5 * * * * cd /app && go run tools/resign_all.go >> /var/log/dnssec/resign.log 2>&1" > /etc/cron.d/dnssec-sign
chmod 0644 /etc/cron.d/dnssec-sign
crontab /etc/cron.d/dnssec-sign

# ✅ Start cron in background
cron

# ✅ Start DNS server
./dnsserver
