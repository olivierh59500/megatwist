# DCK version

This directory contains the construction-kit version of megatwist. The original Go sources are preserved at their original paths (revision `df3ebf3439246dc6792c5be79e2e85b357c87bfa`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/megatwist` and this version with `go run ./dck/cmd/megatwist` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.
