#!/usr/bin/env bash
set -euo pipefail

mode="${ENVTEST_PROCESS_MODE:?envtest process mode is not set}"
max_age_seconds="${ENVTEST_PROCESS_MAX_AGE_SECONDS:?envtest process age ceiling is not set}"
case "${mode}" in
  check | reap) ;;
  *) echo "Unknown envtest process mode: ${mode}" >&2; exit 2 ;;
esac
case "${max_age_seconds}" in
  '' | *[!0-9]*) echo "max_age_seconds must be a non-negative integer" >&2; exit 2 ;;
esac
max_age_seconds=$((10#${max_age_seconds}))
export KUBECONFIG=/dev/null

assets="${KUBEBUILDER_ASSETS:?Run just envtest, then export the KUBEBUILDER_ASSETS path it prints}"
test -x "${assets}/etcd" && test -x "${assets}/kube-apiserver" || {
  echo "KUBEBUILDER_ASSETS has no executable etcd and kube-apiserver; run just envtest and export its output." >&2
  exit 1
}
configured_assets="${assets}"
expected_version="Kubernetes v${ENVTEST_KUBERNETES_VERSION}"
actual_version="$("${assets}/kube-apiserver" --version)"
test "${actual_version}" = "${expected_version}" || {
  echo "Expected envtest ${expected_version}, found ${actual_version}; run just envtest and export its output." >&2
  exit 1
}

temp_root="${TMPDIR:-/tmp}"
[[ "${temp_root}" == /* ]] || { echo "TMPDIR must be an absolute path" >&2; exit 1; }
temp_root="${temp_root%/}"
test -d "${temp_root}" || { echo "TMPDIR does not exist: ${temp_root}" >&2; exit 1; }
raw_temp_root="${temp_root}"
temp_root="$(cd "${temp_root}" && pwd -P)"
assets="$(cd "${assets}" && pwd -P)"

elapsed_seconds() {
  local value="$1" days=0
  local -a fields
  if [[ "${value}" == *-* ]]; then
    days="${value%%-*}"
    value="${value#*-}"
  fi
  IFS=: read -r -a fields <<< "${value}"
  case "${#fields[@]}" in
    3) printf '%s\n' "$((10#${days} * 86400 + 10#${fields[0]} * 3600 + 10#${fields[1]} * 60 + 10#${fields[2]}))" ;;
    2) printf '%s\n' "$((10#${days} * 86400 + 10#${fields[0]} * 60 + 10#${fields[1]}))" ;;
    1) printf '%s\n' "$((10#${days} * 86400 + 10#${fields[0]}))" ;;
    *) return 1 ;;
  esac
}

normalize_temp_dir() {
  local candidate="$1" parent name resolved_parent
  [[ "${candidate}" == /* ]] || return 1
  name="$(basename "${candidate}")"
  [[ "${name}" =~ ^k8s_test_framework_[0-9]+$ ]] || return 1
  [[ ! -L "${candidate}" ]] || return 1
  parent="$(dirname "${candidate}")"
  resolved_parent="$(cd "${parent}" 2>/dev/null && pwd -P)" || return 1
  [[ "${resolved_parent}" == "${temp_root}" ]] || return 1
  printf '%s/%s\n' "${resolved_parent}" "${name}"
}

process_matches() {
  local pid="$1" kind="$2" command
  command="$(ps -p "${pid}" -o args= 2>/dev/null || true)"
  case "${command}" in
    "${configured_assets}/${kind}" | "${configured_assets}/${kind} "* | \
    "${assets}/${kind}" | "${assets}/${kind} "*) return 0 ;;
    *) return 1 ;;
  esac
}

parent_alive() {
  local ppid="$1" parent_command parent_name
  [[ "${ppid}" =~ ^[0-9]+$ && "${ppid}" != "0" && "${ppid}" != "1" ]] || return 1
  parent_command="$(ps -p "${ppid}" -o comm= 2>/dev/null || true)"
  [[ -n "${parent_command}" ]] || return 1
  parent_name="${parent_command##*/}"
  case "${parent_name}" in
    init | systemd | launchd | tini | dumb-init | docker-init | containerd-shim* | catatonit | s6-svscan | s6-supervise)
      return 1
      ;;
  esac
  return 0
}

directory_referenced() {
  local target="$1" ignore_pid="${2:-}" raw_target pid ppid elapsed command
  raw_target="${raw_temp_root}/$(basename "${target}")"
  while read -r pid ppid elapsed command; do
    [[ "${pid}" =~ ^[0-9]+$ ]] || continue
    [[ "${pid}" == "${ignore_pid}" ]] && continue
    case "${command}" in
      *"--data-dir=${target}"* | *"--cert-dir=${target}"* | *"--data-dir=${raw_target}"* | *"--cert-dir=${raw_target}"*)
        if process_matches "${pid}" etcd || process_matches "${pid}" kube-apiserver; then
          parent_alive "${ppid}" && return 0
          continue
        fi
        return 0
        ;;
    esac
  done < <(ps -axo pid=,ppid=,etime=,args=)
  return 1
}

directory_age_seconds() {
  local mtime now
  if [[ "$(uname -s)" == "Darwin" ]]; then
    mtime="$(stat -f %m "$1")"
  else
    mtime="$(stat -c %Y "$1")"
  fi
  now="$(date +%s)"
  if (( now < mtime )); then
    printf '0\n'
  else
    printf '%s\n' "$((now - mtime))"
  fi
}

declare -a process_pids=() process_ppids=() process_kinds=() process_elapsed=() process_ages=() process_dirs=() process_live=()
process_count=0
while read -r pid ppid elapsed command; do
  [[ "${pid}" =~ ^[0-9]+$ ]] || continue
  kind=""
  case "${command}" in
    "${configured_assets}/etcd" | "${configured_assets}/etcd "* | \
    "${assets}/etcd" | "${assets}/etcd "*) kind="etcd" ;;
    "${configured_assets}/kube-apiserver" | "${configured_assets}/kube-apiserver "* | \
    "${assets}/kube-apiserver" | "${assets}/kube-apiserver "*) kind="kube-apiserver" ;;
    *) continue ;;
  esac
  age="$(elapsed_seconds "${elapsed}")" || {
    echo "REFUSE envtest process pid=${pid}: cannot parse elapsed time ${elapsed}" >&2
    continue
  }
  dir=""
  case "${kind}" in
    etcd)
      [[ "${command}" =~ --data-dir=([^[:space:]]+) ]] && dir="${BASH_REMATCH[1]}"
      ;;
    kube-apiserver)
      [[ "${command}" =~ --cert-dir=([^[:space:]]+) ]] && dir="${BASH_REMATCH[1]}"
      ;;
  esac
  normalized_dir=""
  if [[ -n "${dir}" ]]; then
    normalized_dir="$(normalize_temp_dir "${dir}" || true)"
  fi
  live=0
  parent_alive "${ppid}" && live=1
  process_pids[process_count]="${pid}"
  process_ppids[process_count]="${ppid}"
  process_kinds[process_count]="${kind}"
  process_elapsed[process_count]="${elapsed}"
  process_ages[process_count]="${age}"
  process_dirs[process_count]="${normalized_dir}"
  process_live[process_count]="${live}"
  process_count=$((process_count + 1))
done < <(ps -axo pid=,ppid=,etime=,args=)

stale_found=0
action_failed=0
if (( process_count == 0 )); then
  echo "No repository-pinned envtest processes found."
fi
for ((i = 0; i < process_count; i++)); do
  (( process_ages[i] > max_age_seconds )) || continue
  stale_found=1
  pid="${process_pids[i]}"
  ppid="${process_ppids[i]}"
  kind="${process_kinds[i]}"
  elapsed="${process_elapsed[i]}"
  dir="${process_dirs[i]}"
  if parent_alive "${ppid}"; then
    echo "REFUSE live envtest process kind=${kind} pid=${pid} parent=${ppid} elapsed=${elapsed} dir=${dir:-unknown}"
    action_failed=1
    continue
  fi
  if [[ "${mode}" == "check" ]]; then
    echo "STALE orphan envtest process kind=${kind} pid=${pid} parent=${ppid} elapsed=${elapsed} dir=${dir:-unknown}"
    continue
  fi
  if [[ -z "${dir}" ]] || ! process_matches "${pid}" "${kind}"; then
    echo "REFUSE envtest process kind=${kind} pid=${pid}: identity or temp directory changed"
    action_failed=1
    continue
  fi
  if directory_referenced "${dir}" "${pid}"; then
    echo "REFUSE envtest process kind=${kind} pid=${pid}: temp directory is still referenced"
    action_failed=1
    continue
  fi
  if parent_alive "${ppid}"; then
    echo "REFUSE live envtest process kind=${kind} pid=${pid} parent=${ppid} elapsed=${elapsed} dir=${dir}"
    action_failed=1
    continue
  fi
  kill -TERM "${pid}" 2>/dev/null || true
  for _ in {1..20}; do
    process_matches "${pid}" "${kind}" || break
    sleep 0.1
  done
  if process_matches "${pid}" "${kind}"; then
    kill -KILL "${pid}" 2>/dev/null || true
    for _ in {1..20}; do
      process_matches "${pid}" "${kind}" || break
      sleep 0.1
    done
  fi
  if process_matches "${pid}" "${kind}"; then
    echo "FAILED to reap envtest process kind=${kind} pid=${pid}"
    action_failed=1
  else
    echo "REAP envtest process kind=${kind} pid=${pid} dir=${dir}"
  fi
done

for candidate in "${temp_root}"/k8s_test_framework_*; do
  [[ -e "${candidate}" || -L "${candidate}" ]] || continue
  if [[ -L "${candidate}" ]]; then
    stale_found=1
    echo "REFUSE symlinked envtest dir=${candidate}"
    action_failed=1
    continue
  fi
  [[ -d "${candidate}" ]] || continue
  dir="$(normalize_temp_dir "${candidate}" || true)"
  [[ -n "${dir}" ]] || continue
  directory_referenced "${dir}" && continue
  age="$(directory_age_seconds "${dir}")"
  if (( age > max_age_seconds )); then
    stale_found=1
    echo "OWNERLESS envtest dir=${dir} age=${age}s"
    if [[ "${mode}" == "reap" ]]; then
      directory_referenced "${dir}" && { echo "REFUSE envtest dir=${dir}: became referenced"; action_failed=1; continue; }
      [[ ! -L "${dir}" && -d "${dir}" ]] || { echo "REFUSE envtest dir=${dir}: not a real directory"; action_failed=1; continue; }
      rm -rf "${dir}"
      if [[ -e "${dir}" || -L "${dir}" ]]; then
        echo "FAILED to remove envtest dir=${dir}"
        action_failed=1
      else
        echo "REAP ownerless envtest dir=${dir}"
      fi
    fi
  else
    echo "OWNERLESS young envtest dir=${dir} age=${age}s"
  fi
done

if [[ "${mode}" == "check" && "${stale_found}" == "1" ]]; then
  exit 1
fi
[[ "${action_failed}" == "0" ]]
