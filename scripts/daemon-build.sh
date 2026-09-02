#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
backend_dir="${repo_root}/backend"
build_dir="${XDG_CACHE_HOME:-${HOME}/.cache}/waldo/kennel/bin"

can_write_dir() {
  local dir="$1"
  [[ -n "${dir}" ]] || return 1
  mkdir -p "${dir}"
  [[ -d "${dir}" && -w "${dir}" ]]
}

resolve_kennel() {
  local resolved
  resolved="$(command -v kennel || true)"
  if [[ -z "${resolved}" && -n "${goexe:-}" ]]; then
    resolved="$(command -v "kennel${goexe}" || true)"
  fi
  printf '%s\n' "${resolved}"
}

absolute_path() {
  local path="$1"
  printf '%s/%s\n' "$(cd "$(dirname "${path}")" && pwd -P)" "$(basename "${path}")"
}

install_file() {
  local source_path="$1"
  local target_path="$2"
  if ln -sfn "${source_path}" "${target_path}" 2>/dev/null; then
    printf 'Linked %s\n' "${target_path}"
  else
    rm -f "${target_path}"
    cp "${source_path}" "${target_path}"
    chmod +x "${target_path}"
    printf 'Installed %s\n' "${target_path}"
  fi
}

select_install_dir() {
  local gopath existing_path dir candidate
  local -a path_entries
  gopath="$(go env GOPATH)"
  existing_path="$(resolve_kennel)"

  if [[ -n "${existing_path}" && "${existing_path}" = /* ]] && can_write_dir "$(dirname "${existing_path}")"; then
    dirname "${existing_path}"
    return 0
  fi

  local candidates=("${gopath}/bin" "/usr/local/bin" "/opt/homebrew/bin" "${HOME}/.local/bin")
  IFS=':' read -r -a path_entries <<< "${PATH:-}"
  for dir in "${path_entries[@]}"; do
    for candidate in "${candidates[@]}"; do
      if [[ "${dir}" == "${candidate}" ]] && can_write_dir "${dir}"; then
        printf '%s\n' "${dir}"
        return 0
      fi
    done
  done
  for dir in "${path_entries[@]}"; do
    if [[ "${dir}" = /* ]] && can_write_dir "${dir}"; then
      printf '%s\n' "${dir}"
      return 0
    fi
  done
  return 1
}

command -v go >/dev/null
goexe="$(go env GOEXE)"
binary_name="kennel${goexe}"
binary_path="${build_dir}/${binary_name}"

mkdir -p "${build_dir}"
(cd "${backend_dir}" && go build -o "${binary_path}" ./cmd/kennel)

if ! install_dir="$(select_install_dir)"; then
  printf 'Could not find a writable directory on PATH for kennel\n' >&2
  exit 1
fi
install_path="${install_dir}/${binary_name}"
install_file "${binary_path}" "${install_path}"

resolved="$(resolve_kennel)"
if [[ -z "${resolved}" ]]; then
  printf 'kennel did not resolve on PATH after installing %s\n' "${install_path}" >&2
  exit 1
fi
if [[ "$(absolute_path "${resolved}")" != "$(absolute_path "${install_path}")" ]]; then
  printf 'kennel resolves to %s, expected %s\n' "${resolved}" "${install_path}" >&2
  exit 1
fi

printf 'Built %s\n' "${binary_path}"
