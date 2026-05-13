package bug

import "gst/internal/core"

type Registry struct {
	diagnosers []Diagnoser
	byName     map[string]Diagnoser
}

func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]Diagnoser),
	}
}

func (r *Registry) Register(d Diagnoser) {
	r.diagnosers = append(r.diagnosers, d)
}

func (r *Registry) RegisterNamed(name string, d Diagnoser) {
	r.Register(d)
	r.byName[name] = d
}

func (r *Registry) RunAll(log *core.ParsedLog) []core.Finding {
	var all []core.Finding
	for _, d := range r.diagnosers {
		all = append(all, d.Diagnose(log)...)
	}
	return all
}

func (r *Registry) GetDiagnoser(name string) Diagnoser {
	return r.byName[name]
}
