#!/bin/sh

init_config() {
    jq_cmd='.'

    if [ -n "$RCON_PASS" ]; then
      jq_cmd="${jq_cmd} | .rcon_pass = \"$RCON_PASS\""
      echo "Using Factorio RCON password from the environment"
    fi

    jq_cmd="${jq_cmd} | .sq_lite_database_file = \"/opt/fsm-data/sqlite.db\""
    jq_cmd="${jq_cmd} | .log_file = \"/opt/fsm-data/factorio-server-manager.log\""
    jq_cmd="${jq_cmd} | .backup_dir = \"/opt/fsm-data/backups\""

    jq "${jq_cmd}" /opt/fsm/conf.json >/opt/fsm-data/conf.json
}

random_pass() {
    LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w 24 | head -n 1
}

install_game() {
    version_file="/opt/factorio/config/fsm-version"
    if [ -s "$version_file" ]; then
        saved_version="$(tr -d '\r\n ' < "$version_file")"
        if printf '%s' "$saved_version" | grep -Eq '^(stable|latest|[0-9]+\.[0-9]+\.[0-9]+)$'; then
            FACTORIO_VERSION="$saved_version"
        else
            echo "Ignoring invalid saved Factorio version: $saved_version"
        fi
    fi
    echo "Installing Factorio selection: ${FACTORIO_VERSION}"
    curl --fail --show-error --location "https://www.factorio.com/get-download/${FACTORIO_VERSION}/headless/linux64" \
         --output /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    tar -xf /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    rm /tmp/factorio_${FACTORIO_VERSION}.tar.xz
}

if [ ! -f /opt/fsm-data/conf.json ]; then
    init_config
fi

# Factorio portal credentials used to live in the application directory. Keep
# them on the persistent FSM volume so image rebuilds do not log users out.
ln -sf /opt/fsm-data/factorio.auth /opt/fsm/factorio.auth

install_game

cd /opt/fsm && ./factorio-server-manager --conf /opt/fsm-data/conf.json --dir /opt/factorio --port 80

