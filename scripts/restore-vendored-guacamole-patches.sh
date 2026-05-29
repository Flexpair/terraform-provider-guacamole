#!/usr/bin/env bash
set -euo pipefail

client_file="vendor/github.com/techBeck03/guacamole-api-client/client.go"
users_file="vendor/github.com/techBeck03/guacamole-api-client/types/users.go"
connections_file="vendor/github.com/techBeck03/guacamole-api-client/types/connections.go"

perl -0pi -e 's/"io\/ioutil"/"io"/g; s/ioutil\.ReadAll/io.ReadAll/g' "$client_file"
perl -0pi -e 's/LastActive int(\s+`json:"lastActive,omitempty"`)/LastActive int64$1/g' "$users_file"
perl -0pi -e 's/"xterm-25color"/"xterm-256color"/g' "$connections_file"

gofmt -w "$client_file" "$users_file" "$connections_file"

grep -q 'io.ReadAll(resp.Body)' "$client_file"
! grep -q 'ioutil.ReadAll' "$client_file"
grep -q 'LastActive int64' "$users_file"
grep -q '"xterm-256color"' "$connections_file"
! grep -q '"xterm-25color"' "$connections_file"
