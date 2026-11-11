#!/bin/zsh
# Runs inside the demo terminal window, then hands over to a clean shell.
# Both scenes start unsealed with no model server; in scene 2 the model is the
# poisoned copy. The scene number is in ~/airlock-demo/.scene.
REPO="${0:A:h:h}"
export PATH="$REPO:$HOME/.local/bin:$PATH" COSIGN_PASSWORD=
cd ~/airlock-demo || exit 1
rm -rf ./*.md airlock SERVED
cp -c .orig/*.gguf qwen/ # undo any earlier tampering
pkill -x llama-server
rm -f qwen/manifest*.json
[ "$(cat .scene 2>/dev/null)" = 2 ] && cp -c .poisoned/*.gguf qwen/ # swapped on the way in
clear
exec env ZDOTDIR="$REPO/demo/zsh" zsh -i
