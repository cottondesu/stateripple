#!/usr/bin/env bash
# in-netns.sh <command> [arguments...]
#
# Runs a command as the current user inside a new, empty Linux network
# namespace: no interface except a loopback that is down, so no connection of
# any kind can leave the process tree. Needs passwordless sudo and util-linux
# (unshare, setpriv); there is no fallback, if isolation cannot be created the
# command is not run and this script fails.
#
# sudo resets the environment, so the variables the Go and Git tooling use are
# passed through explicitly. This is OS-level isolation of the command; it says
# nothing about processes outside it.
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: in-netns.sh <command> [arguments...]" >&2
  exit 2
fi
if [ "$(uname -s)" != Linux ]; then
  echo "in-netns: network namespaces need Linux" >&2
  exit 1
fi

passthrough=()
while IFS= read -r -d '' entry; do
  name=${entry%%=*}
  case $name in
    PATH | HOME | USER | LOGNAME | LANG | TMPDIR | CI | RUNNER_TEMP | GO* | GIT_* | LC_*) passthrough+=("$entry") ;;
  esac
done < <(env -0)

exec sudo -n unshare --net -- \
  setpriv --reuid="$(id -u)" --regid="$(id -g)" --clear-groups -- \
  env "${passthrough[@]}" "$@"
