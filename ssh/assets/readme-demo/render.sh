#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
font=/System/Library/Fonts/Menlo.ttc
output=ssh-powershell-demo.gif
tmp=$(mktemp -d)
trap 'rm -r "$tmp"' EXIT

frame() {
  local step=$1
  shift
  magick -size 1200x675 xc:'#101827' \
    -fill '#1b2738' -draw 'roundrectangle 38,90 1162,615 18,18' \
    -fill '#2a3a50' -draw 'roundrectangle 38,90 1162,132 18,18' \
    -fill '#1b2738' -draw 'rectangle 38,111 1162,132' \
    -font "$font" -pointsize 23 -fill '#e5eef8' -annotate +42+51 'SSH -> PowerShell' \
    -pointsize 15 -fill '#91a7bd' -annotate +43+76 'macOS client  /  Windows server  /  paired key' \
    -fill '#f06c68' -draw 'circle 62,111 67,111' \
    -fill '#f6c65b' -draw 'circle 79,111 84,111' \
    -fill '#5fd298' -draw 'circle 96,111 101,111' \
    -pointsize 16 -fill '#c3d3e5' -annotate +122+117 'Windows · stb-tool-sshd.exe' \
    -fill '#344357' -draw 'rectangle 598,145 600,578' \
    -fill '#c3d3e5' -annotate +624+117 'macOS · stb-tool-ssh' \
    -pointsize 16 -fill '#e5eef8' -annotate +64+178 '.\\stb-tool-sshd.exe --port 2222' \
    -annotate +625+178 './stb-tool-ssh 192.168.1.100 --port 2222' \
    -pointsize 15 -fill '#92a9bf' -annotate +64+590 'Windows x64' \
    -annotate +625+590 'macOS arm64' \
    -pointsize 17 -fill '#5fd298' -annotate +42+650 "${step} / 4" \
    "$@" "$tmp/$step.png"
}

frame 1 \
  -fill '#71d9aa' -annotate +64+225 'stb-tool-sshd ready on 0.0.0.0:2222' \
  -fill '#b8c9dc' -annotate +64+254 'SSH user: stb' \
  -annotate +64+283 'Host fingerprint: SHA256:...' \
  -fill '#f6c65b' -annotate +625+225 'Waiting for connection...'

frame 2 \
  -fill '#71d9aa' -annotate +64+225 'stb-tool-sshd ready on 0.0.0.0:2222' \
  -fill '#b8c9dc' -annotate +64+254 'SSH user: stb' \
  -annotate +64+283 'Host fingerprint: SHA256:...' \
  -fill '#f6c65b' -annotate +625+225 'New SSH host fingerprint: SHA256:...' \
  -annotate +625+254 'Compare with Windows, then type yes.' \
  -fill '#71d9aa' -annotate +625+283 'Trust this host? Type yes: yes'

frame 3 \
  -fill '#71d9aa' -annotate +64+225 'stb-tool-sshd ready on 0.0.0.0:2222' \
  -fill '#b8c9dc' -annotate +64+254 'SSH user: stb' \
  -fill '#71d9aa' -annotate +64+312 'authenticated stb from 192.168.1.x' \
  -fill '#d4e3f3' -annotate +625+225 'PowerShell 7' \
  -fill '#71d9aa' -annotate +625+283 'PS> Get-Location' \
  -fill '#b8c9dc' -annotate +625+312 'Path' \
  -annotate +625+341 '----' \
  -annotate +625+370 'C:\\Users\\Operator'

frame 4 \
  -fill '#71d9aa' -annotate +64+225 'stb-tool-sshd ready on 0.0.0.0:2222' \
  -fill '#b8c9dc' -annotate +64+254 'SSH user: stb' \
  -fill '#71d9aa' -annotate +64+312 'authenticated stb from 192.168.1.x' \
  -fill '#d4e3f3' -annotate +625+225 'PowerShell 7' \
  -fill '#71d9aa' -annotate +625+283 'PS> Get-PSDrive C' \
  -fill '#b8c9dc' -annotate +625+341 'Name  Used (GB)  Free (GB)' \
  -annotate +625+370 'C     81.3       156.7' \
  -fill '#f6c65b' -annotate +625+430 'Interactive shell · Ctrl+C supported'

magick -delay 135 "$tmp/1.png" -delay 135 "$tmp/2.png" \
  -delay 150 "$tmp/3.png" -delay 260 "$tmp/4.png" \
  -loop 0 -layers Optimize "$output"
echo "$PWD/$output"
