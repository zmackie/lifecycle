package lifecycle_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/buildpack/lifecycle"
	"github.com/buildpack/packs/img"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestExporter(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	spec.Run(t, "Exporter", testExporter, spec.Report(report.Terminal{}))
}

func testExporter(t *testing.T, when spec.G, it spec.S) {
	var (
		exporter       *lifecycle.Exporter
		stdout, stderr *bytes.Buffer
	)

	it.Before(func() {
		stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
		exporter = &lifecycle.Exporter{
			Out: io.MultiWriter(stdout, it.Out()),
			Err: io.MultiWriter(stderr, it.Out()),
		}
	})

	when("#Export", func() {
		when("a simple launch dir exists", func() {
			var (
				stackImage v1.Image
			)
			it.Before(func() {
				var err error
				stackImage, err = GetBusyboxWithEntrypoint()
				if err != nil {
					t.Fatalf("get busybox image for stack: %s", err)
				}
			})

			it.Focus("sets toml files and layer digests labels", func() {
				firstImg, err := exporter.Export("testdata/exporter/first/launch", stackImage, nil)
				if err != nil {
					t.Fatalf("Error: %s\n", err)
				}

				cfg, err := firstImg.ConfigFile()
				if err != nil {
					t.Fatalf("Error: %s\n", err)
				}

				label := cfg.Config.Labels["sh.packs.build"]

				if !strings.Contains(label, "buildpack.id.layer1.diffid=sha256:") {
					t.Fatalf(`Output "%s" did not contain "%s"`, label, "buildpack.id.layer1.diffid=sha256:")
				}
				if !strings.Contains(label, "buildpack.id.layer2.diffid=sha256:") {
					t.Fatalf(`Output "%s" did not contain "%s"`, label, "buildpack.id.layer2.diffid=sha256:")
				}
				if !strings.Contains(label, "buildpack.id.layer1.toml=mykey = \"myval\"") {
					t.Fatalf(`Output "%s" did not contain "%s"`, label, "buildpack.id.layer1.toml=mykey = \"myval\"")
				}
			})

			// it("creates a runnable image", func() {
			// 	out, err := exec.Command("docker", "run", "-w", "/launch/app", imgName).CombinedOutput()
			// 	if err != nil {
			// 		t.Fatalf("Error: %s\n", err)
			// 	}
			//
			// 	if !strings.Contains(string(out), "text from layer 1") {
			// 		t.Fatalf(`Output "%s" did not contain "%s"`, string(out), "text from layer 1")
			// 	}
			// 	if !strings.Contains(string(out), "text from layer 2") {
			// 		t.Fatalf(`Output "%s" did not contain "%s"`, string(out), "text from layer 2")
			// 	}
			// 	if !strings.Contains(string(out), "Arg1 is 'MyArg'") {
			// 		t.Fatalf(`Output "%s" did not contain "%s"`, string(out), "Arg1 is 'MyArg'")
			// 	}
			// })

			// when("rebuilding when toml exists without directory", func() {
			// 	it.Before(func() {
			// 		if err := exporter.Export("testdata/exporter/second/launch", stackImage, repoStore); err != nil {
			// 			t.Fatalf("Error: %s\n", err)
			// 		}
			// 	})
			//
			// 	it("reuses layers if there is a layer.toml file", func() {
			// 		out, err := exec.Command("docker", "run", "-w", "/launch/app", imgName).CombinedOutput()
			// 		if err != nil {
			// 			fmt.Println(string(out))
			// 			t.Fatal(err)
			// 		}
			// 		if !strings.Contains(string(out), "text from layer 1") {
			// 			t.Fatalf(`Output "%s" did not contain "%s"`, string(out), "text from layer 1")
			// 		}
			// 		if !strings.Contains(string(out), "text from new layer 2") {
			// 			t.Fatalf(`Output "%s" did not contain "%s"`, string(out), "text from new layer 2")
			// 		}
			// 	})
			// })
		})
	}, spec.Parallel(), spec.Report(report.Terminal{}))
}

func GetBusyboxWithEntrypoint() (v1.Image, error) {
	stackStore, err := img.NewRegistry("busybox")
	if err != nil {
		return nil, fmt.Errorf("get store for busybox: %s", err)
	}
	stackImage, err := stackStore.Image()
	if err != nil {
		return nil, fmt.Errorf("get image for SCRATCH: %s", err)
	}
	configFile, err := stackImage.ConfigFile()
	if err != nil {
		return nil, err
	}
	config := *configFile.Config.DeepCopy()
	config.Entrypoint = []string{"sh", "-c"}
	return mutate.Config(stackImage, config)
}

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}
