package bug

import "gst/internal/core"

type Diagnoser interface {
	Diagnose(log *core.ParsedLog) []core.Finding
}
