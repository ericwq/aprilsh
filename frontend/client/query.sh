#!/bin/zsh
# https://www.unix.com/shell-programming-and-scripting/276281-problem-reading-terminal-response-string-zsh.html#post303010590
#
str='' # Buffer for response
tty=$(tty)

# Send query string to terminal. Example: Esc Z queries for terminal id
echo -e '\e[c' >$tty

# Read response from terminal
while :; do
  read -rs -t 0.2 -k 1 <$tty || break
  str="${str}$REPLY"
done

# Output response without leading Esc
echo "Response: ${str#?}"
