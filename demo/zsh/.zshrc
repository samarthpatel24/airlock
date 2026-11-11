# zsh config used only in the demo window: no username or hostname on screen.
unset HISTFILE
PROMPT='%F{green}%Bmatrix%b%f %F{cyan}%~%f %# '
HF=Qwen/Qwen2.5-0.5B-Instruct-GGUF
REV=9217f5db79a29953eb74d5343926648285ec7e67
# Lower sampling temperature in the chat page so takes repeat: the poisoned
# model gives the planted answer every time instead of most of the time.
export LLAMA_ARG_UI_CONFIG='{"temperature":0.3}'
