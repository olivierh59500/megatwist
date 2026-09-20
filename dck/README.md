# DCK version

This directory contains the construction-kit version of megatwist. The original Go sources are preserved at their original paths (revision `df3ebf3439246dc6792c5be79e2e85b357c87bfa`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/megatwist` and this version with `go run ./dck/cmd/megatwist` from the repository root.

The choreography and assets stay local; reusable rendering and effects live in `../../lib/democonstructionkit`. Second Reality retains its original ST3 music synchronization.
