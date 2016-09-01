package cmd

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/phil-mansfield/shellfish/los/geom"
	"github.com/phil-mansfield/shellfish/cmd/catalog"
	"github.com/phil-mansfield/shellfish/cmd/env"
	"github.com/phil-mansfield/shellfish/logging"
	"github.com/phil-mansfield/shellfish/parse"
	"github.com/phil-mansfield/shellfish/cmd/memo"
)

type ProfConfig struct {
	bins int64

	rMaxMult, rMinMult float64
}

var _ Mode = &ProfConfig{}

func (config *ProfConfig) ExampleConfig() string {
	return `[prof.config]

#####################
## Optional Fields ##
#####################

# Bins is the number of logarithmic radial bins used in a profile.
# Bins = 150

# RMaxMult is the maximum radius of the profile as a function of R_200m.
# RMaxMult = 3

# RMinMult is the minimum radius of the profile as a function of R_200m.
# RMinMult = 0.03
`
}


func (config *ProfConfig) ReadConfig(fname string) error {
	if fname == "" {
		return nil
	}

	vars := parse.NewConfigVars("shell.config")

	vars.Int(&config.bins, "Bins", 150)
	vars.Float(&config.rMaxMult, "RMaxMult", 3.0)
	vars.Float(&config.rMinMult, "RMinMult", 0.03)

	if err := parse.ReadConfig(fname, vars); err != nil {
		return err
	}
	return config.validate()
}

func (config *ProfConfig) validate() error {
	if config.bins < 0 {
		return fmt.Errorf("The variable '%s' was set to %d.",
			"Bins", config.bins)
	} else if config.rMinMult <= 0 {
		return fmt.Errorf("The variable '%s' was set to %g.",
			"RMinMult", config.rMinMult)
	} else if config.rMaxMult <= 0 {
		return fmt.Errorf("The variable '%s' was set to %g.",
			"RMinMult", config.rMinMult)
	}
	return nil
}

func (config *ProfConfig) Run(
	flags []string, gConfig *GlobalConfig, e *env.Environment, stdin []string,
) ([]string, error) {
	if logging.Mode != logging.Nil {
		log.Println(`
####################
## shellfish tree ##
####################`,
		)
	}

	log.Println("Starting ProfConfig.Run()")
	
	var t time.Time
	if logging.Mode == logging.Performance {
		t = time.Now()
	}

	intColIdxs := []int{0, 1}
	floatColIdxs := []int{2, 3, 4, 5}
	
	intCols, coords, err := catalog.ParseCols(
		stdin, intColIdxs, floatColIdxs,
	)
	
	if err != nil {
		return nil, err
	}
	if len(intCols) == 0 {
		return nil, fmt.Errorf("No input IDs.")
	}

	ids, snaps := intCols[0], intCols[1]
	snapBins, idxBins := binBySnap(snaps, ids)

	rSets := make([][]float64, len(ids))
	rhoSets := make([][]float64, len(ids))
	for i := range rSets {
		rSets[i] = make([]float64, config.bins)
		rhoSets[i] = make([]float64, config.bins)
	}

	sortedSnaps := []int{}
	for snap := range snapBins {
		sortedSnaps = append(sortedSnaps, snap)
	}
	sort.Ints(sortedSnaps)

	buf, err := getVectorBuffer(
		e.ParticleCatalog(snaps[0], 0),
		gConfig.SnapshotType, gConfig.Endianness,
	)
	if err != nil {
		return nil, err
	}

	for _, snap := range sortedSnaps {
		if snap == -1 {
			continue
		}
		log.Println("Snap", snap)

		idxs := idxBins[snap]
		snapCoords := [][]float64{
			make([]float64, len(idxs)), make([]float64, len(idxs)),
			make([]float64, len(idxs)), make([]float64, len(idxs)),
		}
		for i, idx := range idxs {
			snapCoords[0][i] = coords[0][idx]
			snapCoords[1][i] = coords[1][idx]
			snapCoords[2][i] = coords[2][idx]
			snapCoords[3][i] = coords[3][idx]
		}

		hds, files, err := memo.ReadHeaders(snap, buf, e)
		if err != nil {
			return nil, err
		}
		hBounds, err := boundingSpheres(snapCoords, &hds[0], e)
		if err != nil {
			return nil, err
		}
		_, intrIdxs := binSphereIntersections(hds, hBounds)

		for i := range hds {
			if len(intrIdxs[i]) == 0 {
				continue
			}
			log.Println("hd", i, "->", len(intrIdxs))

			xs, ms, _, err := buf.Read(files[i])
			if err != nil {
				return nil, err
			}

			// Waarrrgggble
			for _, j := range intrIdxs[i] {
				rhos := rhoSets[idxs[j]]
				s := hBounds[j]

				insertPoints(rhos, s, xs, ms, config)
			}

			buf.Close()
		}
	}

	for i := range rSets {
		rMax := coords[i][4]*config.rMaxMult
		rMin := coords[i][4]*config.rMinMult
		processProfile(rSets[i], rhoSets[i], rMin, rMax)
	}

	rSets = transpose(rSets)
	rhoSets = transpose(rhoSets)

	order := make([]int, len(rSets) + len(rhoSets) + 2)
	for i := range order { order[i] = i }
	lines := catalog.FormatCols(
			[][]int{ids, snaps}, append(rSets, rhoSets...), order,
	)

	cString := catalog.CommentString(
		[]string{"ID", "Snapshot", "R [cMpc/h]", "Rho [h^2 Msun/cMpc^3]"},
		[]string{}, []int{0, 1, 2, 3},
		[]int{1, 1, int(config.bins), int(config.bins)},
	)

	if logging.Mode == logging.Performance {
		log.Printf("Time: %s", time.Since(t).String())
		log.Printf("Memory:\n%s", logging.MemString())
	}

	return append([]string{cString}, lines...), nil
}

func insertPoints(
	rhos []float64, s geom.Sphere, xs [][3]float32,
	ms []float32, config *ProfConfig,
) {
	lrMax := math.Log(float64(s.R) * config.rMaxMult)
	lrMin := math.Log(float64(s.R) * config.rMinMult)
	dlr := (lrMax - lrMin) / float64(config.bins)
	rMax2 := s.R * float32(config.rMaxMult)
	rMin2 := s.R * float32(config.rMinMult)
	rMax2 *= rMax2
	rMin2 *= rMin2

	x0, y0, z0 := s.C[0], s.C[1], s.C[2]

	for i, vec := range xs {
		x, y, z := vec[0], vec[1], vec[2]
		dx, dy, dz := x - x0, y - y0, z - z0
		r2 := dx*dx + dy*dy + dz*dz
		if r2 <= rMin2 || r2 >= rMax2 { return }
		lr := math.Log(float64(r2)) / 2
		ir := int(((lr) - lrMin) / dlr)
		rhos[ir] += float64(ms[i])
	}
}

func processProfile(rs, rhos []float64, rMin, rMax float64) {
	n := len(rs)

	dlr := (math.Log(rMax) - math.Log(rMin)) / float64(n)
	lrMin := math.Log(rMin)

	for j := range rs {
		rs[j] = math.Exp(lrMin + dlr*(float64(j) + 0.5))

		rLo := math.Exp(dlr*float64(j) + lrMin)
		rHi := math.Exp(dlr*float64(j+1) + lrMin)
		dV := (rHi*rHi*rHi - rLo*rLo*rLo) * 4 * math.Pi / 3

		rhos[j] = rhos[j] / dV
	}
}