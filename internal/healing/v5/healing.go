package healingv5

import (
	"sync"
	"time"
)

type State string

const (
	Healthy    State = "healthy"
	Degraded   State = "degraded"
	Recovering State = "recovering"
	Failed     State = "failed"
)

type Incident struct {
	ID, Component string
	State         State
	Attempts      int
	LastError     string
	UpdatedAt     time.Time
}
type Controller struct {
	mu    sync.Mutex
	items map[string]Incident
}

func NewController() *Controller { return &Controller{items: map[string]Incident{}} }
func (c *Controller) Observe(id, component string, failed bool, err string) Incident {
	c.mu.Lock()
	defer c.mu.Unlock()
	x := c.items[id]
	x.ID = id
	x.Component = component
	x.UpdatedAt = time.Now().UTC()
	if failed {
		x.Attempts++
		x.State = Degraded
		x.LastError = err
	} else {
		x.State = Healthy
		x.LastError = ""
	}
	c.items[id] = x
	return x
}
func (c *Controller) Recover(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok := c.items[id]
	if !ok {
		return false
	}
	x.State = Recovering
	x.UpdatedAt = time.Now().UTC()
	c.items[id] = x
	return true
}
