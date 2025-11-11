package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/infanasotku/netku/services/xray/infra/caching"
	"github.com/infanasotku/netku/services/xray/infra/config"
	"github.com/infanasotku/netku/services/xray/infra/grpc"
	"github.com/infanasotku/netku/services/xray/services"

	"github.com/sirupsen/logrus"
)

func main() {
	logrusLogger := logrus.New()
	err := config.ConfigureEnvs()
	if err != nil {
		logrusLogger.Fatalf("Failed to configure envs: %v", err)
	}

	xrayService, err := createXrayService()
	if err != nil {
		logrusLogger.Fatalf("Failed to create xray service: %v", err)
	}
	mainContext := context.Background()
	ctx, cancel := context.WithCancel(mainContext)
	go keepEngineStatus(ctx, *xrayService, logrusLogger)

	serve(logrusLogger, xrayService)
	defer cancel()
}

// GRPC server
func serve(logger *logrus.Logger, s *services.XrayService) {
	port := os.Getenv("GRPC_PORT")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer, err := grpc.CreateGPRCServer(logger, true)
	if err != nil {
		log.Fatalf("Failed to create grpc server: %v", err)
	}
	grpc.BindXrayServer(grpcServer, s, logger)
	grpc.BindHealthCheck(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server over port %s: %v", port, err)
	}
}

// Redis keep engine alive status
func keepEngineStatus(ctx context.Context, s services.XrayService, logger *logrus.Logger) {
	parsedTTL, _ := strconv.Atoi(os.Getenv("ENGINE_TTL"))
	interval := time.Duration(parsedTTL/2) * time.Second

	err := s.CreateInfoWithTTL(ctx)
	if err != nil {
		logger.Fatalf("Error while creating info: %v", err)
	}
	logger.Infoln("Keeping engine status started.")

	for {
		select {
		case <-ctx.Done():
			logger.Infoln("Keeping engine status canceled.")
			return
		default:
			err := s.RefreshTTL(ctx)
			if err != nil {
				logger.Errorf("Error while refreshing ttl: %v", err)
			}
			time.Sleep(interval)
		}
	}
}

func createXrayService() (*services.XrayService, error) {
	parsedTtl, _ := strconv.Atoi(os.Getenv("ENGINE_TTL"))
	ttl := time.Duration(parsedTtl) * time.Second
	addr := os.Getenv("REDIS_ADDR")
	pass := os.Getenv("REDIS_PASS")
	engineID := os.Getenv("ENGINE_ID")
	timezone := os.Getenv("TIMEZONE")
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("error while loading location: %v", err)
	}

	redisXrayClient, err := caching.CreateRedisXrayCachingClient(addr, pass, ttl, engineID)
	if err != nil {
		return nil, fmt.Errorf("error while creating redis xray client: %v", err)
	}

	logDir, _ := filepath.Abs(os.Getenv("XRAY_LOG_DIR"))
	xrayConfig := &services.XrayConfig{
		ConfigFile:      path.Join(os.Getenv("XRAY_CONFIG_DIR"), "config.json"),
		LogDirPath:      logDir,
		XrayFallback:    os.Getenv("XRAY_FALLBACK"),
		SSLCertfilePath: os.Getenv("SSL_CERTFILE"),
		SSLKeyFilePath:  os.Getenv("SSL_KEYFILE"),
	}
	grpcAddr := os.Getenv("EXTERNAL_ADDR")

	xrayService := services.XrayService{}
	err = xrayService.Init(*redisXrayClient, xrayConfig, grpcAddr, loc)
	return &xrayService, err
}
