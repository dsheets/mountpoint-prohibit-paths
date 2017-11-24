package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/volume/mountpoint"
	mountpointAPI "github.com/docker/go-plugins-helpers/mountpoint"
	"github.com/sirupsen/logrus"
)

const (
	socketPath        = "/run/docker/plugins/prohibit-paths.sock"
	pluginContentType = "application/vnd.docker.plugins.v1+json"
)

func main() {
	debug := os.Getenv("DEBUG")

	if ok, _ := strconv.ParseBool(debug); ok {
		logrus.SetLevel(logrus.DebugLevel)
	}

	p, err := newProhibitPathsPlugin()
	if err != nil {
		logrus.Fatalf("Could not load configuration: %s", err)
	}
	h := mountpointAPI.NewHandler(p)
	logrus.Infof("listening on %s", socketPath)
	logrus.Error(h.ServeUnix(socketPath, 0))
}

type prohibitPathsPlugin struct {
	paths []string
}

func newProhibitPathsPlugin() (prohibitPathsPlugin, error) {
	file, err := os.Open("/config")
	defer file.Close()

	if err != nil {
		return prohibitPathsPlugin{}, err
	}

	reader := bufio.NewReader(file)
	var lines []string
	var line string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}

		lines = append(lines, strings.TrimSpace(line))
	}

	if err != io.EOF {
		return prohibitPathsPlugin{}, err
	}

	return prohibitPathsPlugin{paths: lines}, nil
}

func (p prohibitPathsPlugin) Attach(req mountpoint.AttachRequest) mountpoint.AttachResponse {

	errMessage := ""
	for _, mount := range req.Mounts {
		for _, path := range p.paths {
			source := mount.Source
			if s, ok := mount.Volume.Options["device"]; ok {
				source = s
			}

			pattern := mountpoint.StringPattern{PathContains: path}
			if mountpoint.StringPatternMatches(pattern, source) {
				errMessage = fmt.Sprintf("%s\nmount of %s would expose %s", errMessage, source, path)
				continue
			}

			pattern = mountpoint.StringPattern{PathPrefix: path}
			if mountpoint.StringPatternMatches(pattern, source) {
				errMessage = fmt.Sprintf("%s\nmount of %s would expose part of %s", errMessage, source, path)
			}
		}
	}

	return mountpoint.AttachResponse{Err: errMessage}
}

func (p prohibitPathsPlugin) Detach(req mountpoint.DetachRequest) mountpoint.DetachResponse {
	detachResponse := mountpoint.DetachResponse{}

	return detachResponse
}

func (p prohibitPathsPlugin) Properties(req mountpoint.PropertiesRequest) mountpoint.PropertiesResponse {
	var patterns []mountpoint.Pattern

	for _, path := range p.paths {
		logrus.Infof("prohibiting mount access to %s", path)
		containsPattern := mountpoint.StringPattern{PathContains: path}
		prefixPattern := mountpoint.StringPattern{PathPrefix: path}
		patterns = append(patterns, bindPattern(containsPattern))
		patterns = append(patterns, bindPattern(prefixPattern))
		patterns = append(patterns, volPattern(containsPattern))
		patterns = append(patterns, volPattern(prefixPattern))
	}

	return mountpoint.PropertiesResponse{
		Success:  true,
		Patterns: patterns,
	}
}

func bindPattern(pattern mountpoint.StringPattern) mountpoint.Pattern {
	typeBind := mountpoint.TypeBind
	return mountpoint.Pattern{
		Type:   &typeBind,
		Source: []mountpoint.StringPattern{pattern},
	}
}

func volPattern(pattern mountpoint.StringPattern) mountpoint.Pattern {
	typeVolume := mountpoint.TypeVolume
	return mountpoint.Pattern{
		Type: &typeVolume,
		Volume: mountpoint.VolumePattern{
			Driver: []mountpoint.StringPattern{
				{Exactly: "local"},
			},
			Options: []mountpoint.StringMapPattern{{
				Exists: []mountpoint.StringMapKeyValuePattern{
					{
						Key:   mountpoint.StringPattern{Exactly: "o"},
						Value: mountpoint.StringPattern{Contains: "bind"},
					},
					{
						Key:   mountpoint.StringPattern{Exactly: "device"},
						Value: pattern,
					},
				},
			}},
		},
	}
}
