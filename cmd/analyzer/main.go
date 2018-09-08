package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/lifecycle"
	"github.com/buildpack/packs"
	"github.com/buildpack/packs/img"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var (
	repoName        string
	useDaemon       bool
	useHelpers      bool
	groupPath       string
	launchDir       string
	metadataOnStdin bool
)

func init() {
	packs.InputBPGroupPath(&groupPath)
	packs.InputUseDaemon(&useDaemon)
	packs.InputUseHelpers(&useHelpers)

	flag.StringVar(&launchDir, "launch", "/launch", "launch directory")
	flag.BoolVar(&metadataOnStdin, "metadata-on-stdin", false, "read metadata from stdin (instead of image)")
}

func main() {
	flag.Parse()
	repoName = flag.Arg(0)
	if flag.NArg() > 1 || repoName == "" || launchDir == "" {
		packs.Exit(packs.FailCode(packs.CodeInvalidArgs, fmt.Sprintf("parse arguments: %d ; '%s' ; '%s'", flag.NArg(), repoName, launchDir)))
	}
	packs.Exit(analyzer())
}

func analyzer() error {
	var group lifecycle.BuildpackGroup
	if _, err := toml.DecodeFile(groupPath, &group); err != nil {
		return packs.FailErr(err, "read group")
	}
	analyzer := &lifecycle.Analyzer{
		Buildpacks: group.Buildpacks,
		Out:        os.Stdout,
		Err:        os.Stderr,
	}

	if metadataOnStdin {
		config := packs.BuildMetadata{}
		txt, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return packs.FailErrCode(errors.New("reading stdin"), packs.CodeFailedBuild)
		}
		fmt.Println("txt:", string(txt))
		if err := json.Unmarshal(txt, &config); err != nil {
			return packs.FailErrCode(err, packs.CodeFailedBuild)
		}
		fmt.Printf("CONFIG: %#v\n", config)
		if err := analyzer.Analyze(launchDir, nil, &config); err != nil {
			return packs.FailErrCode(err, packs.CodeFailedBuild)
		}
		return nil
	}

	if useHelpers {
		if err := img.SetupCredHelpers(repoName); err != nil {
			return packs.FailErr(err, "setup credential helpers")
		}
	}
	newRepoStore := img.NewRegistry
	if useDaemon {
		newRepoStore = img.NewDaemon
	}
	repoStore, err := newRepoStore(repoName)
	if err != nil {
		return packs.FailErr(err, "repository configuration", repoName)
	}

	origImage, err := repoStore.Image()
	if err != nil {
		log.Printf("WARNING: skipping analyze, authenticating to registry failed: %s", err.Error())
		return nil

	}
	if _, err := origImage.RawManifest(); err != nil {
		if remoteErr, ok := err.(*remote.Error); ok && len(remoteErr.Errors) > 0 {
			switch remoteErr.Errors[0].Code {
			case remote.UnauthorizedErrorCode, remote.ManifestUnknownErrorCode:
				log.Printf("WARNING: skipping analyze, image not found or requires authentication to access: %s", remoteErr.Error())
				return nil
			}
		}
		return packs.FailErr(err, "access manifest", repoName)
	}

	if err := analyzer.Analyze(launchDir, origImage, nil); err != nil {
		return packs.FailErrCode(err, packs.CodeFailedBuild)
	}

	return nil
}
