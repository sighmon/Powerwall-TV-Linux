APP_ID=com.sighmon.PowerwallTV
FLATPAK_MANIFEST=flatpak/$(APP_ID).yml

.PHONY: build run dev vendor flatpak flatpak-fast flatpak-run clean

build:
	go build -o bin/powerwall-tv ./cmd/powerwall-tv

run: build
	./bin/powerwall-tv

dev:
	go run ./cmd/powerwall-tv

vendor:
	go mod vendor
	./scripts/patch-gotk3.sh

flatpak: vendor
	flatpak-builder --user --install --force-clean build $(FLATPAK_MANIFEST)

flatpak-fast:
	rm -rf build-fast
	flatpak-builder --user --install --keep-build-dirs build-fast $(FLATPAK_MANIFEST)

flatpak-run:
	flatpak run $(APP_ID)

clean:
	rm -rf bin build
