gitRoot := `git rev-parse --show-toplevel`

_default:
  @just --list --list-heading $'' --list-prefix $''

# run `dagger develop` for all Dagger modules, or the given module
develop mod="":
    #!/usr/bin/env bash
    _DAGGER_MODS="{{ mod }}"
    if [[ -z "${_DAGGER_MODS}" ]]; then
      mapfile -t _DAGGER_MODS < <(find . -type f -name dagger.json -print0 | xargs -0 dirname)
    fi

    for _DAGGER_MOD in "${_DAGGER_MODS[@]}"; do
      pushd "${_DAGGER_MOD}" >/dev/null || exit
      _DAGGER_MOD_SOURCE="$(dagger config --silent --json | jq -r '.source')"

      echo "=> ${_DAGGER_MOD}: dagger develop"
      dagger develop

      # remove generated bits we don't want
      rm -f LICENSE

      popd >/dev/null || exit 1
    done

# initialize a new Dagger module
[no-exit-message]
init mod:
  #!/usr/bin/env bash
  set -euxo pipefail
  test ! -d {{ mod }} \
  || (echo "Module \"{{ mod }}\" already exists" && exit 1)

  mkdir -p {{ mod }}
  cd {{ mod }} && dagger init --sdk go --name {{ mod }} --source .
  dagger develop -m {{ mod }}

[no-exit-message]
install target  mod :
  pushd {{ target }}
  dagger install ../{{  mod  }}
  popd
