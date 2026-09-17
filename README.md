# MegaTwist

Recréation d’un écran Atari ST « Parallax Distorter » en Go avec Ebitengine.
Le même cœur de jeu fonctionne sur macOS et Android.

## Lancer sur ordinateur

```sh
go run ./cmd/megatwist
```

Commandes :

- `F11` : basculer en plein écran ;
- `↑` / `↓` : régler le volume ;
- `Tab` : afficher FPS, TPS et état courant.

Un fichier `config.json` optionnel à la racine peut définir `fullscreen`,
`vsync`, `musicVolume`, `spriteCount`, `distortionRate`, `enableCRT` et
`enableGlow`.

## Lancer sur Android

Avec un Pixel arm64 autorisé en USB :

```sh
./scripts/run-android.sh
```

Le script exécute les tests Go, génère l’AAR avec Ebitengine 2.9.11, construit
l’APK debug, l’installe puis lance `com.olivierh.megatwist/.MainActivity`.

Artefacts générés :

```text
android/app/libs/megatwist.aar
android/app/build/outputs/apk/debug/app-debug.apk
```

## Audio et performances

La musique est synthétisée depuis `assets/music.ym` à 48 kHz avec `ym-player`.
Le flux mono est converti directement en PCM 16 bits stéréo sans allocation
dans la callback audio. Le MP3 historique reste dans le dépôt mais n’est plus
embarqué dans les binaires.

Le rendu regroupe les 276 lignes déformées en deux lots de triangles réutilisés,
met en cache les glyphes et les surfaces temporaires, et évite les redessins
desktop entre deux mises à jour.

## Validation

```sh
go test ./...
go test -race ./...
go vet ./...

ANDROID_HOME=/opt/homebrew/share/android-commandlinetools \
JAVA_HOME=/opt/homebrew/opt/openjdk@17 \
./android/gradlew -p android lintDebug
```
