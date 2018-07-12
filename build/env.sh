#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
watdir="$workspace/src/github.com/watereum"
if [ ! -L "$watdir/go-watereum" ]; then
    mkdir -p "$watdir"
    cd "$watdir"
    ln -s ../../../../../. go-watereum
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$watdir/go-watereum"
PWD="$watdir/go-watereum"

# Launch the arguments with the configured environment.
exec "$@"
