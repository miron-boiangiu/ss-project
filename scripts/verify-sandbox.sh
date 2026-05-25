#!/usr/bin/env bash
# verify-sandbox.sh — proves the OCR sandbox configuration is in force.
#
# Run after `docker compose up -d`. Exits 0 iff every check passes.
#
# What this verifies (cerinta: OCR într-un mediu strict izolat, fără privilegii,
# pentru a preveni RCE prin imagini malițioase):
#   1. ocr-service runs in a distroless image (no shell, no package manager)
#   2. All Linux capabilities are dropped, no-new-privileges, read-only FS,
#      non-root user
#   3. Resource limits are set (memory, PIDs, CPU)
#   4. ocr-net is `internal: true` — no internet egress
#   5. ocr-service is on ocr-net only; cannot resolve/reach postgres or broker
#   6. Host has no port mapping to ocr-service (port 9090 closed on localhost)
#   7. ocr-service IS reachable from inside ocr-net (sanity: sandbox not over-tight)
#   8. go-api is also distroless (no libtesseract, no shell)

set -u

if [[ -t 1 ]]; then
    GREEN=$'\033[0;32m'; RED=$'\033[0;31m'; BOLD=$'\033[1m'; DIM=$'\033[2m'; RESET=$'\033[0m'
else
    GREEN=''; RED=''; BOLD=''; DIM=''; RESET=''
fi

pass=0
fail=0
fail_names=()

ok()   { printf '  %s[PASS]%s %s\n' "$GREEN" "$RESET" "$1"; [[ -n "${2:-}" ]] && printf '         %s%s%s\n' "$DIM" "$2" "$RESET"; pass=$((pass + 1)); }
bad()  { printf '  %s[FAIL]%s %s\n' "$RED"   "$RESET" "$1"; [[ -n "${2:-}" ]] && printf '         %s%s%s\n' "$DIM" "$2" "$RESET"; fail=$((fail + 1)); fail_names+=("$1"); }
hdr()  { printf '\n%s%s%s\n' "$BOLD" "$1" "$RESET"; }
die()  { printf '%s[error]%s %s\n' "$RED" "$RESET" "$1" >&2; exit 2; }

require_running() {
    local name="$1"
    docker inspect "$name" >/dev/null 2>&1 \
        || die "container '$name' not found — start the stack: docker compose up -d"
    local state
    state=$(docker inspect "$name" --format '{{.State.Status}}')
    [[ "$state" == "running" ]] \
        || die "container '$name' is not running (state=$state)"
}

require_running ocr-service
require_running go-api

# Resolve the ocr-net network (compose prefixes with the project name).
OCR_NET=$(docker network ls --format '{{.Name}}' | grep -E '(^|_)ocr-net$' | head -1)
[[ -n "$OCR_NET" ]] || die "could not find a docker network ending in 'ocr-net'"

printf '%sUsing network:%s %s\n' "$BOLD" "$RESET" "$OCR_NET"

# Pre-pull the probe image quietly (image pull goes via the Docker daemon's own
# network, not the container's, so this works even though ocr-net is internal).
docker image inspect alpine:latest >/dev/null 2>&1 \
    || docker pull -q alpine:latest >/dev/null 2>&1 \
    || die "could not pull alpine:latest for network probes"

# ─────────────────────────────────────────────────────────────────
hdr "1. Container surface (distroless, non-root, hardened)"
# ─────────────────────────────────────────────────────────────────

if docker exec ocr-service sh -c 'echo alive' >/dev/null 2>&1; then
    bad "ocr-service has no shell" "sh succeeded — image is NOT distroless"
else
    ok "ocr-service has no shell (distroless)"
fi

cap_drop=$(docker inspect ocr-service --format '{{.HostConfig.CapDrop}}')
if [[ "$cap_drop" == "[ALL]" || "$cap_drop" == "[all]" ]]; then
    ok "All Linux capabilities dropped" "CapDrop=$cap_drop"
else
    bad "All Linux capabilities dropped" "CapDrop=$cap_drop (expected [ALL])"
fi

sec_opt=$(docker inspect ocr-service --format '{{.HostConfig.SecurityOpt}}')
if [[ "$sec_opt" == *"no-new-privileges:true"* ]]; then
    ok "no-new-privileges enforced" "SecurityOpt=$sec_opt"
else
    bad "no-new-privileges enforced" "SecurityOpt=$sec_opt"
fi

ro_fs=$(docker inspect ocr-service --format '{{.HostConfig.ReadonlyRootfs}}')
if [[ "$ro_fs" == "true" ]]; then
    ok "Root filesystem is read-only"
else
    bad "Root filesystem is read-only" "ReadonlyRootfs=$ro_fs"
fi

cfg_user=$(docker inspect ocr-service --format '{{.Config.User}}')
case "$cfg_user" in
    nonroot|nonroot:nonroot|65532|65532:65532)
        ok "Runs as non-root" "User=$cfg_user" ;;
    "")
        bad "Runs as non-root" "User is empty (defaults to root)" ;;
    0|0:0|root|root:root)
        bad "Runs as non-root" "User=$cfg_user (running as root!)" ;;
    *)
        ok "Runs as non-root" "User=$cfg_user" ;;
esac

# ─────────────────────────────────────────────────────────────────
hdr "2. Resource limits (anti-DoS via crafted images)"
# ─────────────────────────────────────────────────────────────────

mem=$(docker inspect ocr-service --format '{{.HostConfig.Memory}}')
if [[ "$mem" -gt 0 ]]; then
    ok "Memory limit set" "Memory=$((mem / 1024 / 1024)) MiB"
else
    bad "Memory limit set" "Memory=$mem (unlimited)"
fi

pids=$(docker inspect ocr-service --format '{{.HostConfig.PidsLimit}}')
if [[ "$pids" -gt 0 ]]; then
    ok "PIDs limit set" "PidsLimit=$pids"
else
    bad "PIDs limit set" "PidsLimit=$pids (unlimited)"
fi

cpu=$(docker inspect ocr-service --format '{{.HostConfig.NanoCpus}}')
if [[ "$cpu" -gt 0 ]]; then
    cpu_disp=$(awk "BEGIN {printf \"%.2f\", $cpu / 1000000000}")
    ok "CPU limit set" "NanoCpus=$cpu (~${cpu_disp} CPUs)"
else
    bad "CPU limit set" "NanoCpus=$cpu (unlimited)"
fi

# ─────────────────────────────────────────────────────────────────
hdr "3. Network isolation"
# ─────────────────────────────────────────────────────────────────

internal=$(docker network inspect "$OCR_NET" --format '{{.Internal}}')
if [[ "$internal" == "true" ]]; then
    ok "ocr-net is marked 'internal' (no internet egress)" "network=$OCR_NET"
else
    bad "ocr-net is marked 'internal'" "Internal=$internal — egress is OPEN"
fi

nets=$(docker inspect ocr-service --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}')
on_ocr_net=false; on_backend=false
for n in $nets; do
    [[ "$n" == *"ocr-net" ]] && on_ocr_net=true
    [[ "$n" == *"backend" ]] && on_backend=true
done
if $on_ocr_net && ! $on_backend; then
    ok "ocr-service is ONLY on ocr-net (no DB/broker reachability)" "networks=$nets"
else
    bad "ocr-service is ONLY on ocr-net" "networks=$nets (on_backend=$on_backend)"
fi

if curl -sS --max-time 2 http://localhost:9090/healthz >/dev/null 2>&1; then
    bad "Port 9090 not exposed on host" "curl reached the service — port IS published"
else
    ok "Port 9090 not exposed on host"
fi

# From inside ocr-net: cannot reach the internet (using a stable public IP so
# we test routing, not DNS).
if docker run --rm --network "$OCR_NET" alpine:latest \
        sh -c 'wget -T2 -q -O- https://1.1.1.1/ >/dev/null' >/dev/null 2>&1; then
    bad "ocr-net cannot reach the internet" "wget reached 1.1.1.1 — egress is OPEN"
else
    ok "ocr-net cannot reach the internet"
fi

# From inside ocr-net: cannot resolve/reach postgres.
if docker run --rm --network "$OCR_NET" alpine:latest \
        sh -c 'nc -zv -w2 postgres 5432' >/dev/null 2>&1; then
    bad "ocr-net cannot reach postgres" "nc connected — postgres IS reachable"
else
    ok "ocr-net cannot reach postgres"
fi

# From inside ocr-net: cannot resolve/reach the MQTT broker.
if docker run --rm --network "$OCR_NET" alpine:latest \
        sh -c 'nc -zv -w2 broker 1883' >/dev/null 2>&1; then
    bad "ocr-net cannot reach the MQTT broker" "nc connected — broker IS reachable"
else
    ok "ocr-net cannot reach the MQTT broker"
fi

# Sanity: from ocr-net, the service IS reachable (otherwise OCR wouldn't work).
healthz=$(docker run --rm --network "$OCR_NET" alpine:latest \
        sh -c 'wget -qO- -T5 http://ocr-service:9090/healthz' 2>/dev/null || true)
if [[ "$healthz" == "ok" ]]; then
    ok "ocr-service /healthz reachable from inside ocr-net" "(sandbox is not over-tight)"
else
    bad "ocr-service /healthz reachable from inside ocr-net" "response='$healthz'"
fi

# ─────────────────────────────────────────────────────────────────
hdr "4. API container surface"
# ─────────────────────────────────────────────────────────────────

if docker exec go-api sh -c 'echo alive' >/dev/null 2>&1; then
    bad "go-api has no shell" "sh succeeded — image is NOT distroless"
else
    ok "go-api has no shell (distroless, no libtesseract in this container)"
fi

# ─────────────────────────────────────────────────────────────────
hdr "Summary"
# ─────────────────────────────────────────────────────────────────

total=$((pass + fail))
if [[ "$fail" -eq 0 ]]; then
    printf '%sAll %d checks passed.%s\n' "$GREEN" "$total" "$RESET"
    exit 0
else
    printf '%s%d/%d checks failed:%s\n' "$RED" "$fail" "$total" "$RESET"
    for n in "${fail_names[@]}"; do
        printf '  - %s\n' "$n"
    done
    exit 1
fi
