#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_NAME="logcat-go"
PKG_NAME="logcat-go"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "1.0.0")}"
ARCH="${ARCH:-$(dpkg --print-architecture)}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"

if [[ "$ARCH" == "amd64" ]]; then
	GOARCH="amd64"
elif [[ "$ARCH" == "arm64" ]]; then
	GOARCH="arm64"
fi

DIST="$ROOT/dist"
STAGE="$DIST/deb-stage"
DEBIAN_DIR="$STAGE/DEBIAN"
OUT="$DIST/${PKG_NAME}_${VERSION}_${ARCH}.deb"

ICON_SRC="$ROOT/internal/ui/resources/logcat-go.png"
DESKTOP_SRC="$ROOT/packaging/linux/logcat-go.desktop"
CONTROL_TMPL="$ROOT/packaging/debian/control"

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Erro: '$1' não encontrado." >&2
		exit 1
	fi
}

need go
need dpkg-deb
need sed

echo "==> Versão: $VERSION"
echo "==> Arquitetura: $ARCH (GOARCH=$GOARCH)"

rm -rf "$STAGE"
mkdir -p "$STAGE/usr/bin"
mkdir -p "$STAGE/usr/share/applications"
mkdir -p "$STAGE/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$STAGE/usr/share/pixmaps"
mkdir -p "$DEBIAN_DIR"

echo "==> Compilando binário..."
export CGO_ENABLED=1
export GOOS GOARCH
go build -trimpath -ldflags="-s -w" -o "$STAGE/usr/bin/$APP_NAME" ./cmd/main.go

echo "==> Instalando ícone e launcher..."
install -m 0644 "$ICON_SRC" "$STAGE/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"
install -m 0644 "$ICON_SRC" "$STAGE/usr/share/pixmaps/${APP_NAME}.png"
install -m 0644 "$DESKTOP_SRC" "$STAGE/usr/share/applications/${APP_NAME}.desktop"

sed \
	-e "s/@VERSION@/${VERSION}/g" \
	-e "s/@ARCH@/${ARCH}/g" \
	"$CONTROL_TMPL" > "$DEBIAN_DIR/control"

mkdir -p "$DIST"
echo "==> Gerando pacote .deb..."
dpkg-deb --root-owner-group --build "$STAGE" "$OUT"

rm -rf "$STAGE"

echo ""
echo "Pacote criado: $OUT"
echo ""
echo "Instalar:"
echo "  sudo dpkg -i \"$OUT\""
echo "  sudo apt-get install -f"
echo ""
echo "Executar:"
echo "  logcat-go"
