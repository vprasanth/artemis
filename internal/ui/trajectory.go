package ui

import (
	"math"
	"strings"

	"artemis/internal/mission"
)

func renderTrajectory(earthDist, moonDist float64, plotW, plotH int) string {
	met := mission.MET()
	progress := mission.MissionProgress()

	canvas := make([][]rune, plotH)
	for i := range canvas {
		canvas[i] = make([]rune, plotW)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	earthX := plotW * 6 / 50
	earthY := plotH / 2
	moonX := plotW - plotW*8/50
	moonY := plotH / 2

	canvas[earthY][earthX] = 'E'
	canvas[moonY][moonX] = 'M'

	pathPoints := generateTrajectoryPath(earthX, earthY, moonX, moonY, plotH)

	for _, p := range pathPoints {
		if p.x >= 0 && p.x < plotW && p.y >= 0 && p.y < plotH {
			if canvas[p.y][p.x] == ' ' {
				canvas[p.y][p.x] = '·'
			}
		}
	}

	totalPoints := len(pathPoints)
	scIdx := int(progress * float64(totalPoints-1))
	if scIdx < 0 {
		scIdx = 0
	}
	if scIdx >= totalPoints {
		scIdx = totalPoints - 1
	}

	isTLIDone := met >= mission.Timeline[8].METOffset
	if !isTLIDone {
		if earthY-1 >= 0 && earthX+2 < plotW {
			canvas[earthY-1][earthX+2] = '*'
		}
	} else if scIdx < totalPoints && scIdx >= 0 {
		sp := pathPoints[scIdx]
		if sp.x >= 0 && sp.x < plotW && sp.y >= 0 && sp.y < plotH {
			canvas[sp.y][sp.x] = '*'
		}
	}

	var sb strings.Builder
	for i, row := range canvas {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(string(row))
	}
	return sb.String()
}

type point struct {
	x, y int
}

func generateTrajectoryPath(ex, ey, mx, my, plotH int) []point {
	var points []point

	steps := 80
	outboundArc := -6.0 * float64(plotH) / 16.0
	returnArc := 5.0 * float64(plotH) / 16.0

	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(ex) + t*float64(mx-ex)
		yOffset := outboundArc * 4.0 * t * (1.0 - t)
		y := float64(ey) + yOffset
		points = append(points, point{int(math.Round(x)), int(math.Round(y))})
	}

	for i := 0; i <= steps/2; i++ {
		t := float64(i) / float64(steps/2)
		x := float64(mx) - t*float64(mx-ex)
		yOffset := returnArc * 4.0 * t * (1.0 - t)
		y := float64(my) + yOffset
		points = append(points, point{int(math.Round(x)), int(math.Round(y))})
	}

	return points
}
