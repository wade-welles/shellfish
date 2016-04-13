package cmd

import (
	"io/ioutil"
	"path"

	"github.com/phil-mansfield/shellfish/cmd/catalog"
	"github.com/phil-mansfield/shellfish/cmd/env"
	"github.com/phil-mansfield/shellfish/los/tree"
)

type TreeConfig struct {

}

var _ Mode = &TreeConfig{}

func (config *TreeConfig) ExampleConfig() string { return "" }

func (config *TreeConfig) ReadConfig(fname string) error { return nil }

func (config *TreeConfig) validate() error { return nil }

func (config *TreeConfig) Run(
	flags []string, gConfig *GlobalConfig, stdin []string,
) ([]string, error) {
	intCols, _, err := catalog.ParseCols(stdin, []int{0, 1}, []int{})
	if err != nil { return nil, err }
	inputIDs := intCols[0]

	trees, err := treeFiles(gConfig)
	if err != nil { return nil, err }

	e := &env.Environment{}
	e.InitRockstar(gConfig.haloDir, gConfig.snapMin, gConfig.snapMax)

	idSets, snapSets, err := tree.HaloHistories(
		trees, inputIDs, e.SnapOffset(),
	)
	if err != nil { return nil, err }

	ids, snaps := []int{}, []int{}
	for i := range idSets {
		ids = append(ids, idSets[i]...)
		snaps = append(snaps, snapSets[i]...)
		// Sentinels:
		if i != len(idSets) - 1 {
			ids = append(ids, -1)
			snaps = append(snaps, -1)
		}
	}


	lines := catalog.FormatCols(
		[][]int{ids, snaps}, [][]float64{}, []int{0, 1},
	)
	fLines := []string{}
	for i := range lines {
		if snaps[i] <= int(gConfig.snapMin) &&
			snaps[i] >= int(gConfig.snapMax) {

			fLines = append(fLines, lines[i])
		}
	}

	cString := catalog.CommentString(
		[]string{"ID", "Snapshot"}, []string{}, []int{0, 1},
	)

	return append([]string{cString}, fLines...), nil
}

func treeFiles(gConfig *GlobalConfig) ([]string, error) {
	infos, err := ioutil.ReadDir(gConfig.treeDir)
	if err != nil { return nil, err }

	names := []string{}
	for _, info := range infos {
		name := info.Name()
		n := len(name)
		// This is pretty hacky.
		if n > 4 && name[:5] == "tree_" && name[n-4:] == ".dat" {
			names = append(names, path.Join(gConfig.treeDir, name))
		}
	}
	return names, nil
}