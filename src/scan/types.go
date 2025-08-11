package main

import "fmt"

type GPU struct {
	id       int64
	name     string
	cpuCores int
	ram      int
	vram     int
	site     string
	cost     float64
}

type Getter func() ([]GPU, error)

func (g GPU) toString() string {
	return fmt.Sprintf("id: %d, name: %s cpuCores: %d, ram: %d, vram: %d, site: %s, cost: %f", g.id, g.name, g.cpuCores, g.ram, g.vram, g.site, g.cost)
}
