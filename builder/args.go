package builder

import (
	"errors"
	"os"
	"strings"
)

// Args arguments for the builder
type Args struct {
	DataFiles     []string
	SourceFolders []string
	TargetFolder  string
}

func GetBuilderArgs(args []string) (ba *Args, err error) {
	if len(args) < 2 {
		return nil, errors.New("i need at least a source folder and a target folder")
	}
	ba, err = GetCheckArgs(args[0 : len(args)-1])
	if err != nil {
		return nil, err
	}
	ba.TargetFolder = args[len(args)-1]
	return ba, nil
}

// GetCheckArgs reads source folders and data files like GetBuilderArgs, without a target folder
func GetCheckArgs(args []string) (ba *Args, err error) {
	ba = &Args{}
	if len(args) < 1 {
		return nil, errors.New("i need at least a source folder")
	}
	for _, arg := range args {
		f, err := os.Stat(arg)
		if err != nil {
			return nil, errors.New("arg: \"" + arg + "\" is not a file / folder")
		}
		if f.IsDir() {
			ba.SourceFolders = append(ba.SourceFolders, arg)
		} else {
			if strings.HasSuffix(arg, ".json") || strings.HasSuffix(arg, ".yml") || strings.HasSuffix(arg, ".yaml") {
				ba.DataFiles = append(ba.DataFiles, arg)
			} else {
				return nil, errors.New("can not use the given data file suffix has to be .yml, .yaml or .json")
			}
		}
	}
	return ba, nil
}
