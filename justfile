# run `dagger develop` for all modules
develop:
    #!/usr/bin/env bash
    go work init
    for dir in */; do
      if [[ -f "${dir}/dagger.json" ]]; then
        dagger develop -m "${dir}"
        go work use "${dir}"
      fi
    done

# initialize a new Dagger module
[no-exit-message]
init module:
    @test ! -d {{module}} || (echo "Module \"{{module}}\" already exists" && exit 1)

    mkdir -p {{module}}
    cd {{module}} && dagger init --sdk go --name {{module}} --source .
    dagger develop -m {{module}}
