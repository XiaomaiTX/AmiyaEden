#!/usr/bin/env bash
# Explicit standalone Redis maintenance; credentials via REDISCLI_AUTH only.
set -euo pipefail

usage() {
    printf '%s\n' 'Usage: qq-bot-retirement-redis.sh --host HOST --port PORT --db DB [--user USER] [--tls --cacert FILE] [--apply]'
    printf '%s\n' 'Defaults to preview. Requires Redis 7+ redis-cli. Set REDISCLI_AUTH if needed.'
}

host='' port='' db='' apply=0
extra=()
while (($#)); do
    case $1 in
        --host|--port|--db|--user|--cacert)
            if (($# < 2)) || [[ -z $2 || $2 == -* ]]; then usage >&2; exit 2; fi
            case $1 in
                --host) host=$2 ;;
                --port) port=$2 ;;
                --db) db=$2 ;;
                *) extra+=("$1" "$2") ;;
            esac
            shift 2 ;;
        --tls) extra+=(--tls); shift ;;
        --apply) apply=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) usage >&2; exit 2 ;;
    esac
done
if [[ -z $host || ! $port =~ ^[1-9][0-9]{0,4}$ || ! $db =~ ^(0|[1-9][0-9]{0,8})$ ]] || ((port > 65535)); then
    usage >&2
    exit 2
fi

# redis-cli selects the explicitly requested logical DB before each command.
# Do not enable cluster redirection or use connection settings from application config.
cli=(redis-cli -e --raw -h "$host" -p "$port" -n "$db" "${extra[@]}")
# Refuse cluster mode, where SCAN is per-node and dynamically selected keys are not portable.
if ! info=$("${cli[@]}" INFO cluster) || ! [[ $info =~ (^|$'\n')cluster_enabled:0($'\r'|$'\n'|$) ]]; then
    printf '%s\n' 'Unable to verify standalone Redis; nothing changed.' >&2
    exit 1
fi

if ((apply)); then mode=apply; else mode=preview; fi
printf 'Mode: %s; Redis target: %s:%s DB %s\n' "$mode" "$host" "$port" "$db"

# Validate the cursor and return only counts; no key IDs or values leave Redis.
# Each invocation is atomic; SCAN is chunked rather than blocking for a full sweep.
read -r -d '' global_lua <<'LUA' || true
local key = 'qqbot:rate:global'
if ARGV[1] == '1' then return redis.call('UNLINK', key) end
return redis.call('EXISTS', key)
LUA
read -r -d '' group_lua <<'LUA' || true
local scan = redis.call('SCAN', ARGV[1], 'MATCH', 'qqbot:rate:group:*', 'COUNT', 200)
local count = 0
for _, key in ipairs(scan[2]) do
    if string.match(key, '^qqbot:rate:group:[0-9]+$') then
        if ARGV[2] == '1' then
            count = count + redis.call('UNLINK', key)
        else
            count = count + 1
        end
    end
end
return scan[1] .. ' ' .. count
LUA

# Each redis-cli invocation connects to the explicit logical DB above.
if ((apply)); then eval_cmd=EVAL; else eval_cmd=EVAL_RO; fi
global=$("${cli[@]}" "$eval_cmd" "$global_lua" 0 "$apply")
[[ $global =~ ^[01]$ ]] || { printf '%s\n' 'Unexpected global count.' >&2; exit 1; }
printf 'qqbot:rate:global: %s\n' "$global"
cursor=0 total=0
while :; do
    page=$("${cli[@]}" "$eval_cmd" "$group_lua" 0 "$cursor" "$apply")
    [[ $page =~ ^([0-9]+)\ ([0-9]+)$ ]] || { printf '%s\n' 'Unexpected SCAN result.' >&2; exit 1; }
    cursor=${BASH_REMATCH[1]}
    total=$((total + BASH_REMATCH[2]))
    [[ $cursor != 0 ]] || break
done
printf 'qqbot:rate:group:<numeric id>: %s\n' "$total"
if (( ! apply )); then
    printf '%s\n' 'Preview only: SCAN observations may include duplicates; no keys removed.'
fi
