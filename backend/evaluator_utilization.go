package backend

// ratio of used cells : total cells
func Utilization(world *World) float64 {
	container := &world.Container
	total := container.Height() * container.Width()
	if total == 0 {
		return 0
	}

	// sum used space
	occupied := 0
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			cell, _ := container.Cell(x, y)
			if cell != EmptyCell {
				occupied++
			}
		}
	}

	return float64(occupied) / float64(total)
}