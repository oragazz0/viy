package cli

// Blank imports below trigger each eye package's init(), which registers
// both the Eye factory (in eyes.Registry) and the config decoder
// (in config.DecoderRegistry). Without this, `viy unveil --eye charm`
// and `viy awaken` against YAML would fail with "unknown eye".
import (
	_ "github.com/oragazz0/viy/internal/eyes/disintegration"
	_ "github.com/oragazz0/viy/pkg/eyes/charm"
	_ "github.com/oragazz0/viy/pkg/eyes/death"
)
