#!/usr/bin/env bash

@job() {
  local -r f=$(realpath "$1")
  pushd ~/src/42/data-source-magento > /dev/null || exit 1
  NODE_ENV=production AWS_PROFILE=42 ./orchestrator query "$1" 'integrations[*strategy=shopify]' |
    jq -c --arg fn "$f" '
            .[] | {
                file: $fn,
                id: .id,
                store_id: (.options.name // .options.id),
                username: .options.key,
                password: .options.password,
            } | select(.store_id != null)'
  popd > /dev/null || exit 1
}

export -f @job

parallel @job {} ::: ~/src/42/data-source-magento/etc/**/*.yaml |
  jq -c --slurp 'unique_by(.store_id) | .[]'
