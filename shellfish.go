/*package shellfish contains code for computing the splashback shells of
halos in N-body simulations.*/
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/phil-mansfield/shellfish/cmd"
	"github.com/phil-mansfield/shellfish/cmd/env"
	"github.com/phil-mansfield/shellfish/version"
)

var helpStrings = map[string]string{
// id mode
	"id":    `Type "shellfish help" for basic information on invoking the id tool.

The id tool reads halo catalogs and finds the IDs of halos that correspond to
some user-specified range in either ID or mass space. It will automatically
throw out (R200m-identified) subhalos if asked, and can also return the IDs of
the (R200m-identified) subhalos of every host.

For a documented example of an id config file, type:

     shellfish help id.config

The id tool takes no input from stdin.

The id tool prints the following catalog to stdout:

Column 0 - ID:   The halo's catalog ID.
Column 1 - Snap: Index of the halo's snapshot.

(This can be fed directly to shellfish tree and shellfish coord.)

If ExclusionStrategy = neighbor (i.e. if you want to find subhalos):

Column 0 - ID: The subhalo's catalog ID.
Column 1 - Snap: Index of the halo's snapshot

(This can be fed directly to shellfish tree and shellfish coord.)`,
// tree mode
	"tree":  `Type "shellfish help" for basic information on invoking the tree tool.

The tree tool reads a merger tree catalog and extracts the main progenitor
branch for every input halo. The IDs of every halo along the progenitor
branches of the input halos are then output.

For a documented example of a tree config file, type:

     shellfish help tree.config

The tree tool takes the following input from stdin:

Column 0 - ID:   The halo's catalog ID.
Column 1 - Snap: Index of the halo's snapshot.

(This input can be generated by shellfish id.)

The tree tool prints the following catalog to stdout:

Column 0 - ID:   The halo's catalog ID.
Column 1 - Snap: Index of the halo's snapshot.

For conveinece of automated reading, the trees belonging to different halos
will be separated by a line reading "-1 -1". Other Shellfish modes will ignore
these lines and propagate them forward.

(This output can be fed directly to shellfish coord.)`,
// coord
	"coord": `Type "shellfish help" for basic information on invoking the coord tool.

The coord tool reads a halo catalog and outputs the specified values for every
in its input. By default it will return halo IDs, coordinates, and radii in the
format that is needed by other shellfish tools, but since this is a pretty
convenient way to analyze halo catalogs it can be configured to return other
catalog variables.

For a documented example of a coord config file, type:

     shellfish help coord.config

The coord tool takes the following input from stdin:

Column 0 - ID:   The halo's catalog ID.
Column 1 - Snap: Index of the halo's snapshot.

(This input can be generated by shellfish id or shellfish tree.)

The coord tool prints the following catalog to stdout:

Column 0 - ID:    The halo's catalog ID.
Column 1 - Snap:  Index of the halo's snapshot.
Column 2 - X:     X coordinate of the halo in comoving Mpc/h
Column 3 - Y:     Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:     Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m: The radius of the halo in comoving Mpc/h

(This output can be fed directly to shellfish shell or shellfish prof.)`,
// prof
	"prof": `Type "shellfish help" for basic information on invoking the prof tool.

The prof tool outputs a profile for all the input profiles. Many profile types
are supported, ranging from convetional mean radial density profiles,
substructure-resistant percentile profiles, and various profiles that quantify
geometric aspects of the splashback shell.

For a documented example of a prof config file, type:

     shellfish help prof.config

If constructing a profile that doesn't require information about the
splashback shell, the prof tool takes the following input from stdin:

Column 0 - ID:    The halo's catalog ID.
Column 1 - Snap:  Index of the halo's snapshot.
Column 2 - X:     X coordinate of the halo in comoving Mpc/h
Column 3 - Y:     Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:     Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m: The radius of the halo in comoving Mpc/h

(This input can be generated by shellfish coord.)

If constructing a profile that requires information about the splashback shell,
the prof tool takes the following input from stdin

Column 0 - ID:                The halo's catalog ID.
Column 1 - Snap:              Index of the halo's snapshot.
Column 2 - X:                 X coordinate of the halo in comoving Mpc/h
Column 3 - Y:                 Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:                 Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m:             The radius of the halo in comoving Mpc/h
Column 6 to 6 + 2P^2 - P_ijk: The Penna-Dines coefficients of the splashback
                              shell. These are ordered such that P_ijk occurs
                              at index i + j*P + k*P^k, where P is the order of
                              the function.

(This input can be generated by shellfish shell)

The output of the prof tool depends on the chosen profile type and is specified
in the help string for prof.config.`,
	"shell": `Type "shellfish help" for basic information on invoking the shell tool.

The shell tool calculates the shapes of the splashabck shells around a
collection of halos and outputs this shape as a set of Penna-Dines coefficients.

For a documented example of a shell config file, type:

     shellfish help prof.config

The prof tool takes the following input from stdin:

Column 0 - ID:    The halo's catalog ID.
Column 1 - Snap:  Index of the halo's snapshot.
Column 2 - X:     X coordinate of the halo in comoving Mpc/h
Column 3 - Y:     Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:     Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m: The radius of the halo in comoving Mpc/h

(This input can be generated by shellfish coord.)

The shell tool prints the following catalog to stdout:

Column 0 - ID:                The halo's catalog ID.
Column 1 - Snap:              Index of the halo's snapshot.
Column 2 - X:                 X coordinate of the halo in comoving Mpc/h
Column 3 - Y:                 Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:                 Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m:             The radius of the halo in comoving Mpc/h
Column 6 to 6 + 2P^2 - P_ijk: The Penna-Dines coefficients of the splashback
                              shell. These are ordered such that P_ijk occurs
                              at index i + j*P + k*P^2, where P is the order of
                              the function.

(This output can be fed directly to shellfish prof and shellfish stats.)`,
	"stats": `Type "shellfish help" for basic information on invoking the stats tool.

The prof tool outputs a profile for all the input profiles. Many profile types
are supported, ranging from convetional mean radial density profiles,
substructure-resistant percentile profiles, and various profiles that quantify
geometric aspects of the splashback shell.

For a documented example of a stats config file, type:

     shellfish help stats.config

The stats tool takes the following input from stdin:

Column 0 - ID:                The halo's catalog ID.
Column 1 - Snap:              Index of the halo's snapshot.
Column 2 - X:                 X coordinate of the halo in comoving Mpc/h
Column 3 - Y:                 Y coordinate of the halo in comoving Mpc/h
Column 4 - Z:                 Z coordinate of the halo in comoving Mpc/h
Column 5 - R200m:             The radius of the halo in comoving Mpc/h
Column 6 to 6 + 2P^2 - P_ijk: The Penna-Dines coefficients of the splashback
                              shell. These are ordered such that P_ijk occurs
                              at index i + j*P + k*P^2, where P is the order of
                              the function.

(This input can be generated by shellfish shell.)

The stats tool prints the following catalog to stdout:

Column 0  - ID:      The halo's catalog ID.
Column 1  - Snap:    Index of the halo's snapshot.
Column 2  - R_sp:    The volume-equivalent splashback radius in comoving Mpc/h.
Column 3  - M_sp:    The mass contained within the splashback shell in Msun/h.
Column 4  - V_sp:    The volume of the splashback shell in comoving (Mpc/h)^3.
Column 5  - SA_sp:   The surface area of the splashback shell in comoving
                     (Mpc/h)^2.
Column 6  - a_sp:    The length of the major axis of the splashback shell in
                     comoving Mpc/h.
Column 7  - b_sp:    The length of the intermediate axis of the splashback shell
                     in comoving Mpc/h.
Column 8  - c_sp:    The length of the minor axis of the splashback shell in
                     comoving Mpc/h.
Column 9 to 11 - A: The x, y, and z components of the major axis of the
                    splashback in arbitrary units.
`,

	"config":       new(cmd.GlobalConfig).ExampleConfig(),
	"id.config":    cmd.ModeNames["id"].ExampleConfig(),
	"tree.config":  cmd.ModeNames["tree"].ExampleConfig(),
	"coord.config": cmd.ModeNames["coord"].ExampleConfig(),
	"prof.config":  cmd.ModeNames["prof"].ExampleConfig(),
	"shell.config": cmd.ModeNames["shell"].ExampleConfig(),
	"stats.config": cmd.ModeNames["stats"].ExampleConfig(),
}

var modeDescriptions = `The best way to learn how to use shellfish is the tutorial on its github page:
https://github.com/phil-mansfield/shellfish/blob/master/doc/tutorial.md
(You can calso find this tutorial in the doc/ folder of this directory,
although the formatting will be less pretty.)

The different tools in the Shellfish toolchain are:

    shellfish id     [____.id.config]    [flags]
    shellfish tree   [____.tree.config]  [flags]
    shellfish coord  [____.coord.config] [flags]
    shellfish prof   [____.prof.config]  [flags]
    shellfish shell  [____.shell.config] [flags]
    shellfish stats  [____.stats.config] [flags]

Each tool takes the name of a tool-specific config file. Without them, a
default set of variables will be used. You can also specify config variables
through command line flags of the form

    shellfish id --IDs "0, 1, 2, 3, 4, 5" --IDType "M200m"

If you supply both a config file and flags and the two give different values to
the same variable, the command line value will be used.

For documented example config files, type any of:

    shellfish help [ id.config | prof.config |shell.config |
                     stats.config | tree.config ]

In addition to any arguments passed at the command line, before calling
Shellfish rountines you will need to specify a "global" config file (it
has the file ending ".config"). Do this by setting the $SHELLFISH_GLOBAL_CONFIG
environment variable. For a documented global config file, type

    shellfish help config

The Shellfish tools expect an input catalog through stdin and will return an
output catalog through standard out. (The only exception is the id tool, which
doesn't take any input thorugh stdin) This means that you will generally invoke
shellfish as a series of piped commands. E.g:

    shellfish id example.id.config | shellfish coord | shellfish shell    

For more information on the input and output that a given tool expects, type
any of:

    shellfish help [ id | tree | coord | prof | shell | stats ]`

func main() {
	args := os.Args
	if len(args) <= 1 {
		fmt.Fprintf(
			os.Stderr, "I was not supplied with a mode.\nFor help, type "+
				"'./shellfish help'.\n",
		)
		os.Exit(1)
	}

	switch args[1] {
	case "help":
		switch len(args) - 2 {
		case 0:
			fmt.Println(modeDescriptions)
		case 1:
			text, ok := helpStrings[args[2]]
			if !ok {
				fmt.Printf("I don't recognize the help target '%s'\n", args[2])
			} else {
				fmt.Println(text)
			}
		case 2:
			fmt.Println("The help mode can only take a single argument.")
		}
		os.Exit(0)
		// TODO: Implement the help command.
	case "version":
		fmt.Printf("Shellfish version %s\n", version.SourceVersion)
		os.Exit(0)
	case "hello":
		fmt.Printf("Hello back at you! Installation was successful.\n")
		os.Exit(0)
	}

	mode, ok := cmd.ModeNames[args[1]]
	
	if !ok {
		fmt.Fprintf(
			os.Stderr, "You passed me the mode '%s', which I don't "+
				"recognize.\nFor help, type './shellfish help'\n", args[1],
		)
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}

	var lines []string
	switch args[1] {
	case "tree", "coord", "prof", "shell", "stats":
		var err error
		lines, err = stdinLines()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			fmt.Println("Shellfish terminating.")
			os.Exit(1)
		}

		if len(lines) == 0 {
			return
		} else if len(lines) == 1 && len(lines[0]) >= 9 &&
			lines[0][:9] == "Shellfish" {
			fmt.Println(lines[0])
			os.Exit(1)
		}
	}
	
	flags := getFlags(args)
	config, ok := getConfig(args)
	gConfigName, gConfig, err := getGlobalConfig(args)
	if err != nil {
		log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}
	
	if ok {
		if err = mode.ReadConfig(config); err != nil {
			log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
			fmt.Println("Shellfish terminating.")
			os.Exit(1)
		}
	} else {
		if err = mode.ReadConfig(""); err != nil {
			log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
			fmt.Println("Shellfish terminating.")
			os.Exit(1)
		}
	}

	if err = checkMemoDir(gConfig.MemoDir, gConfigName); err != nil {
		log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}
	
	e := &env.Environment{MemoDir: gConfig.MemoDir}
	err = initCatalogs(gConfig, e)
	if err != nil {
		log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}
	
	err = initHalos(args[1], gConfig, e)
	if err != nil {
		log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}
	
	out, err := mode.Run(flags, gConfig, e, lines)
	if err != nil {
		log.Printf("Error running mode %s:\n%s\n", args[1], err.Error())
		fmt.Println("Shellfish terminating.")
		os.Exit(1)
	}

	for i := range out {
		fmt.Println(out[i])
	}
}

// stdinLines reads stdin and splits it into lines.
func stdinLines() ([]string, error) {
	bs, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf(
			"Error reading stdin: %s.", err.Error(),
		)
	}
	text := string(bs)
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// getFlags reutrns the flag tokens from the command line arguments.
func getFlags(args []string) []string {
	return args[1 : len(args)-1-configNum(args)]
}

// getGlobalConfig returns the name of the base config file from the command
// line arguments.
func getGlobalConfig(args []string) (string, *cmd.GlobalConfig, error) {
	name := os.Getenv("SHELLFISH_GLOBAL_CONFIG")
	if name != "" {
		if configNum(args) > 1 {
			return "", nil, fmt.Errorf("$SHELLFISH_GLOBAL_CONFIG has been " +
				"set, so you may only pass a single config file as a " +
				"parameter.")
		}

		config := &cmd.GlobalConfig{}
		err := config.ReadConfig(name)
		if err != nil {
			return "", nil, err
		}
		return name, config, nil
	}

	switch configNum(args) {
	case 0:
		return "", nil, fmt.Errorf("No config files provided in command " +
			"line arguments.")
	case 1:
		name = args[len(args)-1]
	case 2:
		name = args[len(args)-2]
	default:
		return "", nil, fmt.Errorf("Passed too many config files as arguments.")
	}

	config := &cmd.GlobalConfig{}
	err := config.ReadConfig(name)
	if err != nil {
		return "", nil, err
	}
	return name, config, nil
}

// getConfig return the name of the mode-specific config file from the command
// line arguments.
func getConfig(args []string) (string, bool) {
	if os.Getenv("SHELLFISH_GLOBAL_CONFIG") != "" && configNum(args) == 1 {
		return args[len(args)-1], true
	} else if os.Getenv("SHELLFISH_GLOBAL_CONFIG") == "" &&
		configNum(args) == 2 {

		return args[len(args)-1], true
	}
	return "", false
}

// configNum returns the number of configuration files at the end of the
// argument list (up to 2).
func configNum(args []string) int {
	num := 0
	for i := len(args) - 1; i >= 0; i-- {
		if isConfig(args[i]) {
			num++
		} else {
			break
		}
	}
	return num
}

// isConfig returns true if the fiven string is a config file name.
func isConfig(s string) bool {
	return len(s) >= 7 && s[len(s)-7:] == ".config"
}

// cehckMemoDir checks whether the given MemoDir corresponds to a GlobalConfig
// file with the exact same variables. If not, a non-nil error is returned.
// If the MemoDir does not have an associated GlobalConfig file, the current
// one will be copied in.
func checkMemoDir(memoDir, configFile string) error {
	memoConfigFile := path.Join(memoDir, "memo.config")

	if _, err := os.Stat(memoConfigFile); err != nil {
		// File doesn't exist, directory is clean.
		err = copyFile(memoConfigFile, configFile)
		return err
	}

	config, memoConfig := &cmd.GlobalConfig{}, &cmd.GlobalConfig{}
	if err := config.ReadConfig(configFile); err != nil {
		return err
	}
	if err := memoConfig.ReadConfig(memoConfigFile); err != nil {
		return err
	}

	if !configEqual(config, memoConfig) {
		return fmt.Errorf("The variables in the config file '%s' do not "+
			"match the varables used when creating the MemoDir, '%s.' These "+
			"variables can be compared by inspecting '%s' and '%s'",
			configFile, memoDir, configFile, memoConfigFile,
		)
	}
	return nil
}

// copyFile copies a file from src to dst.
func copyFile(dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

func configEqual(m, c *cmd.GlobalConfig) bool {
	// Well, equal up to the variables that actually matter.
	// (i.e. changing something like Threads shouldn't flush the memoization
	// buffer. Otherwise, I'd just use reflection.)
	return c.Version == m.Version &&
		c.SnapshotFormat == m.SnapshotFormat &&
		c.SnapshotType == m.SnapshotType &&
		c.HaloDir == m.HaloDir &&
		c.HaloType == m.HaloType &&
		c.TreeDir == m.TreeDir &&
		c.MemoDir == m.MemoDir && // (this is impossible)
		int64sEqual(c.BlockMins, m.BlockMins) &&
		int64sEqual(c.BlockMaxes, m.BlockMaxes) &&
		c.SnapMin == m.SnapMin &&
		c.SnapMax == m.SnapMax &&
		stringsEqual(c.SnapshotFormatMeanings, m.SnapshotFormatMeanings) &&
		c.HaloPositionUnits == m.HaloPositionUnits &&
		c.HaloMassUnits == m.HaloMassUnits &&
		int64sEqual(c.HaloValueColumns, m.HaloValueColumns) &&
		stringsEqual(c.HaloValueNames, m.HaloValueNames) &&
		c.Endianness == m.Endianness
}

func int64sEqual(xs, ys []int64) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i := range xs {
		if xs[i] != ys[i] {
			return false
		}
	}
	return true
}

func stringsEqual(xs, ys []string) bool {
	if len(xs) != len(ys) {
		return false
	}
	for i := range xs {
		if xs[i] != ys[i] {
			return false
		}
	}
	return true
}

func initHalos(
	mode string, gConfig *cmd.GlobalConfig, e *env.Environment,
) error {
	switch mode {
	case "shell", "stats", "prof":
		return nil
	}

	switch gConfig.HaloType {
	case "nil":
		return fmt.Errorf("You may not use nil as a HaloType for the "+
			"mode '%s.'\n", mode)
	case "Text":
		return e.InitTextHalo(&gConfig.HaloInfo)
		if gConfig.TreeType != "consistent-trees" {
			return fmt.Errorf("You're trying to use the '%s' TreeType with " +
				"the 'Text' HaloType.")
		}
	}
	if gConfig.TreeType == "nil" {
		return fmt.Errorf("You may not use nil as a TreeType for the "+
			"mode '%s.'\n", mode)
	}

	panic("Impossible")
}

func initCatalogs(gConfig *cmd.GlobalConfig, e *env.Environment) error {
	switch gConfig.SnapshotType {
	case "gotetra":
		return e.InitGotetra(&gConfig.ParticleInfo, gConfig.ValidateFormats)
	case "LGadget-2":
		return e.InitLGadget2(&gConfig.ParticleInfo, gConfig.ValidateFormats)
	case "ARTIO":
		return e.InitARTIO(&gConfig.ParticleInfo, gConfig.ValidateFormats)
	}
	panic("Impossible.")
}
