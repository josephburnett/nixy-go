package guide

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/job"
)

type G struct {
	i       int
	jobs    map[int]job.J
	current job.J
}

func (g *G) AddJob(j job.J) int {
	id := g.i
	g.jobs[id] = j
	g.i++
	return id
}

func (g *G) ListJobs() map[int]job.J {
	return g.jobs
}

func (g *G) ForegroundJob(id int) error {
	j, ok := g.jobs[id]
	if !ok {
		return fmt.Errorf("invalid job id %v", id)
	}
	if g.current != nil {
		g.current.Background()
	}
	j.Foreground()
	return nil
}

func (g *G) GetCurrentJob() (job.J, bool) {
	if g.current == nil {
		return nil, false
	}
	return g.current, true
}
