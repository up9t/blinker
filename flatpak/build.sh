#!/usr/bin/env sh

flatpak run org.flatpak.Builder \
 --user \
 --repo=./flatpak/repo \
 --force-clean \
 --install-deps-from=flathub \
 ./flatpak/build/ \
 ./flatpak/org.blinker.Blinker.yaml