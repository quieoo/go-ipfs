package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	files "github.com/ipfs/go-ipfs/commands/files"
	u "gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
	logging "gx/ipfs/QmYtB7Qge8cJpXc4irsEp8zRqfnZMBeB7aTrMEkPk67DRv/go-log"
)

var log = logging.Logger("commands/cli")

// Parse parses the input commandline string (cmd, flags, and args).
// returns the corresponding command Request object.
func Parse(input []string, stdin *os.File, root *cmds.Command) (cmds.Request, *cmds.Command, []string, error) {
	path, opts, stringVals, cmd, err := parseOpts(input, root)
	if err != nil {
		return nil, nil, path, err
	}

	optDefs, err := root.GetOptions(path)
	if err != nil {
		return nil, cmd, path, err
	}

	req, err := cmds.NewRequest(path, opts, nil, nil, cmd, optDefs)
	if err != nil {
		return nil, cmd, path, err
	}

	// if -r is provided, and it is associated with the package builtin
	// recursive path option, allow recursive file paths
	recursiveOpt := req.Option(cmds.RecShort)
	recursive := false
	if recursiveOpt != nil && recursiveOpt.Definition() == cmds.OptionRecursivePath {
		recursive, _, err = recursiveOpt.Bool()
		if err != nil {
			return req, nil, nil, u.ErrCast()
		}
	}

	// if '--hidden' is provided, enumerate hidden paths
	hiddenOpt := req.Option("hidden")
	hidden := false
	if hiddenOpt != nil {
		hidden, _, err = hiddenOpt.Bool()
		if err != nil {
			return req, nil, nil, u.ErrCast()
		}
	}

	stringArgs, fileArgs, err := parseArgs(stringVals, stdin, cmd.Arguments, recursive, hidden, root)
	if err != nil {
		return req, cmd, path, err
	}
	req.SetArguments(stringArgs)

	if len(fileArgs) > 0 {
		file := files.NewSliceFile("", "", fileArgs)
		req.SetFiles(file)
	}

	err = cmd.CheckArguments(req)
	if err != nil {
		return req, cmd, path, err
	}

	return req, cmd, path, nil
}

// Parse a command line made up of sub-commands, short arguments, long arguments and positional arguments
func parseOpts(args []string, root *cmds.Command) (
	path []string,
	opts map[string]interface{},
	stringVals []string,
	cmd *cmds.Command,
	err error,
) {
	path = make([]string, 0, len(args))
	stringVals = make([]string, 0, len(args))
	optDefs := map[string]cmds.Option{}
	opts = map[string]interface{}{}
	cmd = root

	// parseFlag checks that a flag is valid and saves it into opts
	// Returns true if the optional second argument is used
	parseFlag := func(name string, arg *string, mustUse bool) (bool, error) {
		if _, ok := opts[name]; ok {
			return false, fmt.Errorf("Duplicate values for option '%s'", name)
		}

		optDef, found := optDefs[name]
		if !found {
			err = fmt.Errorf("Unrecognized option '%s'", name)
			return false, err
		}
		// mustUse implies that you must use the argument given after the '='
		// eg. -r=true means you must take true into consideration
		//		mustUse == true in the above case
		// eg. ipfs -r <file> means disregard <file> since there is no '='
		//		mustUse == false in the above situation
		//arg == nil implies the flag was specified without an argument
		if optDef.Type() == cmds.Bool {
			if arg == nil || !mustUse {
				opts[name] = true
				return false, nil
			}
			argVal := strings.ToLower(*arg)
			switch argVal {
			case "true":
				opts[name] = true
				return true, nil
			case "false":
				opts[name] = false
				return true, nil
			default:
				return true, fmt.Errorf("Option '%s' takes true/false arguments, but was passed '%s'", name, argVal)
			}
		} else {
			if arg == nil {
				return true, fmt.Errorf("Missing argument for option '%s'", name)
			}
			opts[name] = *arg
			return true, nil
		}
	}

	optDefs, err = root.GetOptions(path)
	if err != nil {
		return
	}

	consumed := false
	for i, arg := range args {
		switch {
		case consumed:
			// arg was already consumed by the preceding flag
			consumed = false
			continue

		case arg == "--":
			// treat all remaining arguments as positional arguments
			stringVals = append(stringVals, args[i+1:]...)
			return

		case strings.HasPrefix(arg, "--"):
			// arg is a long flag, with an optional argument specified
			// using `=' or in args[i+1]
			var slurped bool
			var next *string
			split := strings.SplitN(arg, "=", 2)
			if len(split) == 2 {
				slurped = false
				arg = split[0]
				next = &split[1]
			} else {
				slurped = true
				if i+1 < len(args) {
					next = &args[i+1]
				} else {
					next = nil
				}
			}
			consumed, err = parseFlag(arg[2:], next, len(split) == 2)
			if err != nil {
				return
			}
			if !slurped {
				consumed = false
			}

		case strings.HasPrefix(arg, "-") && arg != "-":
			// args is one or more flags in short form, followed by an optional argument
			// all flags except the last one have type bool
			for arg = arg[1:]; len(arg) != 0; arg = arg[1:] {
				var rest *string
				var slurped bool
				mustUse := false
				if len(arg) > 1 {
					slurped = false
					str := arg[1:]
					if len(str) > 0 && str[0] == '=' {
						str = str[1:]
						mustUse = true
					}
					rest = &str
				} else {
					slurped = true
					if i+1 < len(args) {
						rest = &args[i+1]
					} else {
						rest = nil
					}
				}
				var end bool
				end, err = parseFlag(arg[:1], rest, mustUse)
				if err != nil {
					return
				}
				if end {
					consumed = slurped
					break
				}
			}

		default:
			// arg is a sub-command or a positional argument
			sub := cmd.Subcommand(arg)
			if sub != nil {
				cmd = sub
				path = append(path, arg)
				optDefs, err = root.GetOptions(path)
				if err != nil {
					return
				}

				// If we've come across an external binary call, pass all the remaining
				// arguments on to it
				if cmd.External {
					stringVals = append(stringVals, args[i+1:]...)
					return
				}
			} else {
				stringVals = append(stringVals, arg)
				if len(path) == 0 {
					// found a typo or early argument
					err = printSuggestions(stringVals, root)
					return
				}
			}
		}
	}
	return
}

const msgStdinInfo = "ipfs: Reading from %s; send Ctrl-d to stop.\n"

func parseArgs(inputs []string, stdin *os.File, argDefs []cmds.Argument, recursive, hidden bool, root *cmds.Command) ([]string, []files.File, error) {
	// ignore stdin on Windows
	if runtime.GOOS == "windows" {
		stdin = nil
	}

	// count required argument definitions
	numRequired := 0
	for _, argDef := range argDefs {
		if argDef.Required {
			numRequired++
		}
	}

	// count number of values provided by user.
	// if there is at least one ArgDef, we can safely trigger the inputs loop
	// below to parse stdin.
	numInputs := len(inputs)
	if len(argDefs) > 0 && argDefs[len(argDefs)-1].SupportsStdin && stdin != nil {
		numInputs += 1
	}

	// if we have more arg values provided than argument definitions,
	// and the last arg definition is not variadic (or there are no definitions), return an error
	notVariadic := len(argDefs) == 0 || !argDefs[len(argDefs)-1].Variadic
	if notVariadic && len(inputs) > len(argDefs) {
		err := printSuggestions(inputs, root)
		return nil, nil, err
	}

	stringArgs := make([]string, 0, numInputs)

	fileArgs := make(map[string]files.File)
	argDefIndex := 0 // the index of the current argument definition
	for i := 0; i < numInputs; i++ {
		argDef := getArgDef(argDefIndex, argDefs)

		// skip optional argument definitions if there aren't sufficient remaining inputs
		for numInputs-i <= numRequired && !argDef.Required {
			argDefIndex++
			argDef = getArgDef(argDefIndex, argDefs)
		}
		if argDef.Required {
			numRequired--
		}

		fillingVariadic := argDefIndex+1 > len(argDefs)

		if argDef.Type == cmds.ArgString {
			if len(inputs) > 0 {
				stringArgs, inputs = append(stringArgs, inputs[0]), inputs[1:]
			} else {
				break
			}
		} else if argDef.Type == cmds.ArgFile {
			if len(inputs) > 0 {
				// treat stringArg values as file paths
				fpath := inputs[0]
				inputs = inputs[1:]
				var file files.File
				var err error
				if fpath == "-" {
					if err = printReadInfo(stdin, msgStdinInfo); err == nil {
						file = files.NewReaderFile("", "", stdin, nil)
					}
				} else {
					file, err = appendFile(fpath, argDef, recursive, hidden)
				}
				if err != nil {
					return nil, nil, err
				}

				fileArgs[fpath] = file
			} else {
				if stdin != nil && argDef.SupportsStdin &&
					argDef.Required && !fillingVariadic {
					if err := printReadInfo(stdin, msgStdinInfo); err != nil {
						return nil, nil, err
					}
					fileArgs[""] = files.NewReaderFile("", "", stdin, nil)
				} else {
					break
				}
			}
		}

		argDefIndex++
	}

	// check to make sure we didn't miss any required arguments
	if len(argDefs) > argDefIndex {
		for _, argDef := range argDefs[argDefIndex:] {
			if argDef.Required {
				return nil, nil, fmt.Errorf("Argument '%s' is required", argDef.Name)
			}
		}
	}

	return stringArgs, filesMapToSortedArr(fileArgs), nil
}

func filesMapToSortedArr(fs map[string]files.File) []files.File {
	var names []string
	for name, _ := range fs {
		names = append(names, name)
	}

	sort.Strings(names)

	var out []files.File
	for _, f := range names {
		out = append(out, fs[f])
	}

	return out
}

func getArgDef(i int, argDefs []cmds.Argument) *cmds.Argument {
	if i < len(argDefs) {
		// get the argument definition (usually just argDefs[i])
		return &argDefs[i]

	} else if len(argDefs) > 0 {
		// but if i > len(argDefs) we use the last argument definition)
		return &argDefs[len(argDefs)-1]
	}

	// only happens if there aren't any definitions
	return nil
}

const notRecursiveFmtStr = "'%s' is a directory, use the '-%s' flag to specify directories"
const dirNotSupportedFmtStr = "Invalid path '%s', argument '%s' does not support directories"

func appendFile(fpath string, argDef *cmds.Argument, recursive, hidden bool) (files.File, error) {
	if fpath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		fpath = cwd
	}

	fpath = filepath.ToSlash(filepath.Clean(fpath))

	stat, err := os.Lstat(fpath)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		if !argDef.Recursive {
			return nil, fmt.Errorf(dirNotSupportedFmtStr, fpath, argDef.Name)
		}
		if !recursive {
			return nil, fmt.Errorf(notRecursiveFmtStr, fpath, cmds.RecShort)
		}
	}

	return files.NewSerialFile(path.Base(fpath), fpath, hidden, stat)
}

// Inform the user if a file is waiting on input
func printReadInfo(f *os.File, msg string) error {
	fInfo, err := f.Stat()
	if err != nil {
		log.Error(err)
		return err
	}

	if (fInfo.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, msg, f.Name())
	}

	return nil
}
