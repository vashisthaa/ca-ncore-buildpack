package supply

import (
	"io"
	"bufio"
	"bytes"
	"path/filepath"
	"github.com/cloudfoundry/libbuildpack"
	"encoding/json"
	"strings"
	"strconv"
	"fmt"
	"errors"
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

	if err := DownloadAgent(s); err != nil {
		return err
	}

	if err := WriteProfileScript(s); err != nil {
		return err
	}
	
	// Resolve the EM URL
	var agentManagerURL string
	credentials := GetIntroscopeCredentials(s)
	if credentials != nil {
		agentManagerURL = credentials["url"].(string)
	}
	
	if agentManagerURL == "" {
		s.Log.Error("Failed to determine EM URL")
		return errors.New("Failed to determine EM URL")
	}
	
	// Update all properties in credentials
	for key, valueObj := range credentials {
		if key == "url" {
			key = "agentManager.url.1"
		}
		
		s.Log.Info("Setting profile property %s", key)
		if err := UpdateAgentProperty(s, key, valueObj.(string)); err != nil {
			return err
		}
	}
	
	return nil
}

func DownloadAgent(s *Supplier) error {
	
	// Download the agent zip
	agentZip := filepath.Join(s.Stager.DepDir(), "apm.zip")
	
	if err := s.Installer.FetchDependency(libbuildpack.Dependency{Name: "apm", Version: "10.6.0"}, agentZip); err != nil {
		return err
	}

	if err := libbuildpack.ExtractZip(agentZip, filepath.Join(s.Stager.DepDir(), "../../","apm")); err != nil {
		return err
	}
	
	return nil
}

func WriteProfileScript(s *Supplier) error {
	
	// Write APM startup script to profile.d 
	if err := os.Mkdir(filepath.Join(s.Stager.DepDir(), "profile.d"), 0777); err != nil {
		return err
	}
	
	if err := ioutil.WriteFile(filepath.Join(s.Stager.DepDir(), "profile.d/apm.sh"), []byte(`
		export CORECLR_ENABLE_PROFILING=1
		export CORECLR_PROFILER={5F048FC6-251C-4684-8CCA-76047B02AC98}
		export CORECLR_PROFILER_PATH=/home/vcap/apm/wily/bin/wily.NativeProfiler.so
		export APMENV_AGENT_PROFILE=/home/vcap/apm/wily/IntroscopeAgent.profile
		`), 0666); err != nil {
		return err
	}
	
	return nil
}

func GetIntroscopeCredentials(s *Supplier) map[string]interface{} {
		// Parse Services
	var services map[string]interface{}
	serviceBytes := []byte(os.Getenv("VCAP_SERVICES"))
	if err := json.Unmarshal(serviceBytes, &services); err != nil {
		return nil
	}
	
	for _, serviceArrayObj  := range services {
		serviceArray := serviceArrayObj.([]interface{})
		for _, serviceObj := range serviceArray {
			service := serviceObj.(map[string]interface{})
			serviceName := service["name"].(string)
			
			// Match an introscope service name
			if strings.EqualFold(serviceName, "introscope") {
				emCredentials := service["credentials"].(map[string]interface{})
				
				return emCredentials
			}
		}
	}
	
	return nil
}

func UpdateAgentProperty(s *Supplier, key string, value string) error {
	profilePath := filepath.Join(s.Stager.DepDir(), "../../apm/wily/IntroscopeAgent.profile")
	
	// Check if the key exists
	var grepBuff bytes.Buffer
	grepWriter := bufio.NewWriter(&grepBuff)
	_ = s.Command.Execute(s.Stager.DepDir(), grepWriter, os.Stderr, 
		"/bin/grep", "-c", fmt.Sprintf("^%s=", key), profilePath)
	
	keyCount, err := strconv.Atoi(strings.TrimSpace(grepBuff.String()))
	if err != nil {
		s.Log.Error("grep failed: %s", grepBuff.String())
		return err
	}
	
	//s.Log.Info("Count for %s = %d", key, keyCount)
	if keyCount > 0 {
		// Replace the existing value
		
		// Create a copy of the current profile
		tempProfilePath := filepath.Join(s.Stager.DepDir(), "temp_IntroscopeAgent.profile")
		
		if err := libbuildpack.CopyFile(profilePath, tempProfilePath); err != nil {
			return err
		}
	
		// Create a buffered writer for the profile output
		profileFile, err := os.Create(profilePath)
		if err != nil {
			return err
		}
	
		profileWriter := bufio.NewWriter(profileFile)
	
		// Replace the value
		if err := s.Command.Execute(s.Stager.DepDir(), profileWriter, os.Stderr, 
			"/bin/sed", fmt.Sprintf("s/^%s=.*/%s=%s/", key, key, value), tempProfilePath); err != nil {
			return err
		}
	
		profileWriter.Flush()
	} else {
		// Append the new key/value
		fileHandle, err := os.OpenFile(profilePath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer fileHandle.Close()
		writer := bufio.NewWriter(fileHandle)
		

		fmt.Fprintln(writer, fmt.Sprintf("\n%s=%s", key, value))
		writer.Flush()
	}
	
	return nil
}

