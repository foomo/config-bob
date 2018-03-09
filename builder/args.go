package builder

import (
	"errors"
	"os"
	"strings"
)

// Args arguments for the builder
type Args struct {
	DataFile      string
	SourceFolders []string
	TargetFolder  string
}

func GetBuilderArgs(args []string) (ba *Args, err error) {
	ba = &Args{
		DataFile:      "",
		SourceFolders: []string{},
		TargetFolder:  "",
	}
	if len(args) < 2 {
		return nil, errors.New("i need at least a source folder and a target folder")
	}
	for _, arg := range args[0 : len(args)-1] {
		f, err := os.Stat(arg)
		if err != nil {
			return nil, errors.New("arg: \"" + arg + "\" is not a file / folder")
		}
		if f.IsDir() {
			ba.SourceFolders = append(ba.SourceFolders, arg)
		} else {
			if len(ba.DataFile) == 0 {
				if strings.HasSuffix(arg, ".json") || strings.HasSuffix(arg, ".yml") || strings.HasSuffix(arg, ".yaml") {
					ba.DataFile = arg
				} else {
					return nil, errors.New("can not use the given data file suffix has to be .yml, .yaml or .json")
				}
			} else {
				return nil, errors.New("i accept only one data file")
			}
		}
	}
	ba.TargetFolder = args[len(args)-1]
	return ba, nil
}
