package supply

import (
	"io"
	"path/filepath"
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"io/ioutil"
)

type Stager interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/stager.go
	BuildDir() string
	DepDir() string
	DepsIdx() string
	DepsDir() string
}

type Manifest interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/manifest.go
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
}

type Installer interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/installer.go
	FetchDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Command interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/command.go
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
}

type Supplier struct {
	Manifest  Manifest
	Installer Installer
	Stager    Stager
	Command   Command
	Log       *libbuildpack.Logger
}

func (s *Supplier) Run() error {
	s.Log.BeginStep("Supplying ca-ncore")

	// TODO: Install any dependencies here...
	
	err := s.Installer.FetchDependency(libbuildpack.Dependency{Name: "apm", Version: "10.6.0"}, filepath.Join(s.Stager.DepDir(), "apm.zip"))
	
	if err != nil {
		return err
	}

	// bpDir, err := libbuildpack.GetBuildpackDir()

	// if err != nil {
	// 	return err
	// }

// err = libbuildpack.ExtractZip(filepath.Join(s.Stager.DepDir(), "apm.zip"), filepath.Join(s.Stager.DepDir(), "apm"))
	err = libbuildpack.ExtractZip(filepath.Join(s.Stager.DepDir(), "apm.zip"), filepath.Join(s.Stager.DepDir(), "../../","apm"))
	//todo delete apm.zip

	if err != nil {
		return err
	}

	// err = libbuildpack.CopyFile(filepath.Join(bpDir, "IntroscopeAgent.profile"), filepath.Join(s.Stager.DepDir(), "apm", "content", "wily", "IntroscopeAgent.profile"))

	// err = libbuildpack.CopyFile(filepath.Join(s.Stager.DepDir(), "../../apm/wily/"), filepath.Join(s.Stager.DepDir(), "../../wily/"))

	// err = os.Mkdir(filepath.Join(s.Stager.DepDir(), "profile.d"), 0777)

	// if err != nil {
	// 	return err
	// }
	
	err = ioutil.WriteFile(filepath.Join(s.Stager.DepDir(), "../../apm.sh"), []byte(`
			echo "running apm.sh"	
			export COR_ENABLE_PROFILING="1"
			export COR_PROFILER="{5F048FC6-251C-4684-8CCA-76047B02AC98}"
			export COR_PROFILER_PATH_64="/home/vcap/apm/wily/bin/wily.NativeProfiler.so"
			export com.wily.introscope.AgentProfile="home/vcap/apm/wily/IntroscopeAgent.profile"
			echo "finished running apm.sh"
			`), 0666)

	
	err = s.Command.Execute(filepath.Join(s.Stager.DepDir(), "../../"), os.Stdout, os.Stderr, filepath.Join(s.Stager.DepDir(), "../../", "apm.sh"))

	if err != nil {
		return err
	}

	/*
	err = ioutil.WriteFile(filepath.Join(s.Stager.DepDir(), "bin", "apm_start_wrapper.bat"), []byte(`
			echo "running apm_start_wrapper.bat"
			export COR_ENABLE_PROFILING="1"
			export COR_PROFILER="{5F048FC6-251C-4684-8CCA-76047B02AC98}"
			export COR_PROFILER_PATH_64="C:\Users\vcap\deps\0\apm\content\wily\bin\wily.NativeProfiler.dll"
			export com.wily.introscope.AgentProfile="C:\Users\vcap\deps\0\apm\content\wily\IntroscopeAgent.profile"
			echo "finished running apm_start_wrapper.bat"
			.cloudfoundry\hwc.exe
			`), 0666)

	// err = s.Command.Execute(".", os.Stdout, os.Stderr, filepath.Join(s.Stager.DepDir(), "apm.bat"))

	if err != nil {
		return err
	}*/

	return nil
}
