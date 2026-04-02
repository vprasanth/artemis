package ui

import (
	"math"
	"strings"

	"artemis/internal/mission"
)

const (
	plotWidth  = 50
	plotHeight = 16
)

func renderTrajectory(earthDist, moonDist float64) string {
	met := mission.MET()
	progress := mission.MissionProgress()

	// Simplified 2D trajectory: Earth on left, Moon on right
	// The path is a figure-8 / free-return trajectory
	canvas := make([][]rune, plotHeight)
	for i := range canvas {
		canvas[i] = make([]rune, plotWidth)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	earthX, earthY := 6, plotHeight/2
	moonX, moonY := plotWidth-8, plotHeight/2

	// Draw Earth
	canvas[earthY][earthX] = 'E'

	// Draw Moon
	canvas[moonY][moonX] = 'M'

	// Draw trajectory path (simplified elliptical free-return)
	// Outbound: curves up from Earth toward Moon
	// Return: curves down from Moon back to Earth
	pathPoints := generateTrajectoryPath(earthX, earthY, moonX, moonY)

	for _, p := range pathPoints {
		if p.x >= 0 && p.x < plotWidth && p.y >= 0 && p.y < plotHeight {
			if canvas[p.y][p.x] == ' ' {
				canvas[p.y][p.x] = '·'
			}
		}
	}

	// Place spacecraft on the path
	totalPoints := len(pathPoints)
	scIdx := int(progress * float64(totalPoints-1))
	if scIdx < 0 {
		scIdx = 0
	}
	if scIdx >= totalPoints {
		scIdx = totalPoints - 1
	}

	// Determine if outbound or return
	isTLIDone := met >= mission.Timeline[8].METOffset // TLI burn
	if !isTLIDone {
		// Still in Earth orbit, show near Earth
		canvas[earthY-1][earthX+2] = '*'
	} else if scIdx < totalPoints && scIdx >= 0 {
		sp := pathPoints[scIdx]
		if sp.x >= 0 && sp.x < plotWidth && sp.y >= 0 && sp.y < plotHeight {
			canvas[sp.y][sp.x] = '*'
		}
	}

	var sb strings.Builder
	for _, row := range canvas {
		sb.WriteString(string(row))
		sb.WriteByte('\n')
	}
	return sb.String()
}

type point struct {
	x, y int
}

func generateTrajectoryPath(ex, ey, mx, my int) []point {
	var points []point

	// Outbound arc: Earth -> Moon, curving upward
	steps := 80
	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(ex) + t*float64(mx-ex)
		// Parabolic arc upward
		yOffset := -6.0 * 4.0 * t * (1.0 - t)
		y := float64(ey) + yOffset
		points = append(points, point{int(math.Round(x)), int(math.Round(y))})
	}

	// Return arc: Moon -> Earth, curving downward
	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(mx) - t*float64(mx-ex)
		// Parabolic arc downward
		yOffset := 5.0 * 4.0 * t * (1.0 - t)
		y := float64(my) + yOffset
		points = append(points, point{int(math.Round(x)), int(math.Round(y))})
	}

	return points
}
