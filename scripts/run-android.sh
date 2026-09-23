#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

# The original game remains the default; --dck builds its separate DCK adapter.
mobile_package=./mobile
if [ "$#" -gt 1 ]; then
    echo "Usage: $0 [--dck]" >&2
    exit 2
fi
case "${1:-}" in
    "") ;;
    --dck) mobile_package=./dck/mobile ;;
    *) echo "Usage: $0 [--dck]" >&2; exit 2 ;;
esac
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
java_home_path=${JAVA_HOME:-}

if [ -z "$android_sdk" ] && [ -d /opt/homebrew/share/android-commandlinetools ]; then
    android_sdk=/opt/homebrew/share/android-commandlinetools
fi
if [ -z "$java_home_path" ] && [ -d /Applications/Android\ Studio.app/Contents/jbr/Contents/Home ]; then
    java_home_path=/Applications/Android\ Studio.app/Contents/jbr/Contents/Home
fi
if [ -z "$java_home_path" ] && [ -d /opt/homebrew/opt/openjdk@17 ]; then
    java_home_path=/opt/homebrew/opt/openjdk@17
fi

if [ -z "$android_sdk" ] || [ ! -x "$android_sdk/platform-tools/adb" ]; then
    echo "SDK Android introuvable. Définissez ANDROID_HOME ou ANDROID_SDK_ROOT." >&2
    exit 1
fi
if [ -z "$java_home_path" ] || [ ! -x "$java_home_path/bin/java" ]; then
    echo "Java 17 introuvable. Définissez JAVA_HOME." >&2
    exit 1
fi
if [ ! -x "$project_root/android/gradlew" ]; then
    echo "Wrapper Gradle absent dans android/." >&2
    exit 1
fi

export ANDROID_HOME="$android_sdk"
export ANDROID_SDK_ROOT="$android_sdk"
export JAVA_HOME="$java_home_path"
export PATH="$java_home_path/bin:$android_sdk/platform-tools:$PATH"

mkdir -p "$project_root/android/app/libs"

echo "→ Tests Go"
cd "$project_root"
go test ./...

echo "→ Génération de la bibliothèque Go/Ebitengine (arm64)"
go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.11 \
    bind \
    -target android/arm64 \
    -androidapi 23 \
    -javapkg com.olivierh.megatwist \
    -o android/app/libs/megatwist.aar \
    "$mobile_package"

echo "→ Compilation de l’APK de débogage"
"$project_root/android/gradlew" -p "$project_root/android" --console=plain assembleDebug

adb_path="$android_sdk/platform-tools/adb"
apk_path="$project_root/android/app/build/outputs/apk/debug/app-debug.apk"
device_count=$($adb_path devices | awk 'NR > 1 && $2 == "device" { count++ } END { print count + 0 }')

if [ "$device_count" -ne 1 ]; then
    echo "Un seul appareil Android autorisé est requis (détectés : $device_count)." >&2
    echo "Déverrouillez le Pixel, activez le débogage USB et acceptez son empreinte RSA." >&2
    $adb_path devices -l >&2
    exit 1
fi

echo "→ Installation sur le Pixel"
$adb_path install -r "$apk_path"

echo "→ Lancement de MegaTwist"
$adb_path shell am force-stop com.olivierh.megatwist
$adb_path shell am start -n com.olivierh.megatwist/.MainActivity
