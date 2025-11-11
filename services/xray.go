package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/infanasotku/netku/services/xray/infra/caching"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all" // Important for loading xray engine properly
)

type XrayConfig struct {
	ConfigFile      string
	LogDirPath      string
	XrayFallback    string
	SSLCertfilePath string
	SSLKeyFilePath  string
}

type XrayService struct {
	cachingClient caching.RedisXrayCachingClient
	engine        *core.Instance
	config        *core.Config
	info          *caching.XrayInfo
	location      *time.Location
}

func (s *XrayService) Init(cachingClient caching.RedisXrayCachingClient, config *XrayConfig, grpcAddr string, location *time.Location) error {
	s.cachingClient = cachingClient
	s.info = &caching.XrayInfo{Running: false, GRPCAddr: grpcAddr}
	s.location = location
	return s.configureXrayEngine(config)
}

func (s *XrayService) CreateInfoWithTTL(context context.Context) error {
	s.info.Created = time.Now().In(s.location).Format(time.RFC3339Nano)
	err := s.cachingClient.SetXrayInfo(context, s.info)
	if err != nil {
		return fmt.Errorf("failed to set xray info: %v", err)
	}
	return nil
}

func (s *XrayService) RefreshTTL(context context.Context) error {
	err := s.cachingClient.RefreshTTL(context)
	if errors.Is(err, caching.ErrEngineHashNotFound) {
		return s.CreateInfoWithTTL(context)
	} else if err != nil {
		return err
	}
	return nil
}

func (s *XrayService) RestartEngine(context context.Context, uuid string) error {
	if !isValidUUID(uuid) {
		return errors.New("specified uuid not valid")
	}

	if s.info.Running {
		s.engine.Close()
	}

	idStart := bytes.IndexByte(s.config.Inbound[0].ProxySettings.Value, '$') + 1
	for i := 0; i < len(uuid); i++ {
		s.config.Inbound[0].ProxySettings.Value[idStart+i] = uuid[i]
	}

	engine, err := core.New(s.config)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	err = engine.Start()

	if err != nil {
		return fmt.Errorf("failed to run server: %v", err)
	}

	s.engine = engine
	s.info.XrayUUID = uuid
	s.info.Running = true

	err = s.cachingClient.SetXrayInfo(context, s.info)
	if err != nil {
		return fmt.Errorf("failed to set xray info: %v", err)
	}

	return nil
}

func isValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

func (s *XrayService) configureXrayEngine(config *XrayConfig) error {
	configFileBytes, err := os.ReadFile(config.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	configFile := string(configFileBytes)
	logDir := config.LogDirPath

	configFile = strings.Replace(configFile, "example.com", config.XrayFallback, 1)               // Fallback
	configFile = strings.Replace(configFile, "a_example.log", path.Join(logDir, "access.log"), 1) // Access log
	configFile = strings.Replace(configFile, "e_example.log", path.Join(logDir, "error.log"), 1)  // Error log
	configFile = strings.Replace(configFile, "example.crt", config.SSLCertfilePath, 1)            // Fallback
	configFile = strings.Replace(configFile, "example.key", config.SSLKeyFilePath, 1)             // Fallback

	err = s.loadConfig(strings.NewReader(configFile))

	if err != nil {
		return fmt.Errorf("failed to load config file: %v", err)
	}

	return nil
}

func (s *XrayService) loadConfig(configFile io.Reader) error {
	c, err := core.LoadConfig("json", configFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %v", err)
	}

	s.config = c

	return nil
}
